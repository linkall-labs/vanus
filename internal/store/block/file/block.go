// Copyright 2022 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package file

import (
	// standard libraries.
	"context"
	"encoding/binary"
	"os"
	"sync"
	stdatomic "sync/atomic"

	// third-party libraries.
	"go.uber.org/atomic"

	// first-party libraries.
	metapb "github.com/linkall-labs/vanus/proto/pkg/meta"

	// this project.
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/internal/store/block"
	"github.com/linkall-labs/vanus/observability"
)

const (
	headerBlockSize = 4 * 1024
	// version + capacity + size + number + full.
	headerSize = 4 + 8 + 8 + 4 + 1
)

// Block
//
// The layout of Block is:
//   ┌────────────────┬─────────────────────┬───────────────┐
//   │  Header Block  │  Data Blocks ...    │  Index Block  │
//   └────────────────┴─────────────────────┴───────────────┘
// An index Block contains one entry per data Block.
type Block struct {
	id   vanus.ID
	path string

	version int32
	cap     int64

	actx appendContext
	fo   atomic.Int64
	so   int64 // sync offset

	mu sync.RWMutex

	f *os.File

	cis     block.ClusterInfoSource
	indexes []index
}

// Make sure block implements block.Block.
var _ block.Block = (*Block)(nil)

func (b *Block) ID() vanus.ID {
	return b.id
}

func (b *Block) Path() string {
	return b.path
}

func (b *Block) full() bool {
	return stdatomic.LoadUint32(&b.actx.full) != 0
}

func (b *Block) Appendable() bool {
	return !b.full()
}

func (b *Block) Close(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)
	if err := b.sync(ctx); err != nil {
		return err
	}
	return b.f.Close()
}

func (b *Block) HealthInfo() *metapb.SegmentHealthInfo {
	info := &metapb.SegmentHealthInfo{
		Id:                   b.id.Uint64(),
		Size:                 b.size(),
		EventNumber:          int32(b.actx.num),
		SerializationVersion: b.version,
		IsFull:               b.full(),
	}
	// Fill cluster information.
	if cis := b.cis; cis != nil {
		cis.FillClusterInfo(info)
	}
	return info
}

func (b *Block) size() int64 {
	return b.fo.Load() - headerBlockSize
}

func (b *Block) remaining(length, num uint32) uint32 {
	// capacity - headerBlockSize - dataLength - indexLength.
	return uint32(b.cap) - headerBlockSize - length - num*indexSize
}

func (b *Block) persistHeader(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)

	buf := make([]byte, headerSize)
	binary.BigEndian.PutUint32(buf[0:4], uint32(b.version))
	binary.BigEndian.PutUint64(buf[4:12], uint64(b.cap))
	binary.BigEndian.PutUint64(buf[12:20], uint64(b.size()))
	binary.BigEndian.PutUint32(buf[20:24], b.actx.num)
	if b.full() {
		buf[24] = 1
	}

	// TODO: does it safe when concurrent write and append?
	if _, err := b.f.WriteAt(buf, 0); err != nil {
		return err
	}

	return nil
}

func (b *Block) loadHeader(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)

	buf := make([]byte, headerSize)
	if _, err := b.f.ReadAt(buf, 0); err != nil {
		return err
	}

	b.version = int32(binary.BigEndian.Uint32(buf[0:4]))
	b.cap = int64(binary.BigEndian.Uint64(buf[4:12]))
	size := int64(binary.BigEndian.Uint64(buf[12:20]))
	b.actx.offset = uint32(size + headerBlockSize)
	b.actx.num = binary.BigEndian.Uint32(buf[20:24])
	b.actx.full = uint32(buf[24])
	b.fo.Store(int64(b.actx.offset))
	b.so = int64(b.actx.offset)

	return nil
}

func (b *Block) persistIndex(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)

	if !b.full() {
		return nil
	}

	length := indexSize * len(b.indexes)
	buf := make([]byte, length)
	for i := range b.indexes {
		idx := &b.indexes[i]
		off := length - (i+1)*indexSize
		_, _ = idx.MarshalTo(buf[off : off+indexSize])
	}

	if _, err := b.f.WriteAt(buf, b.cap-int64(length)); err != nil {
		return err
	}
	return nil
}

func (b *Block) loadIndex(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)

	// rebuild index
	if !b.full() {
		return b.rebuildIndex()
	}

	// read index directly
	return b.loadIndexFromFile()
}

func (b *Block) loadIndexFromFile() error {
	num := b.actx.num
	length := num * indexSize

	// Read index data from file.
	data := make([]byte, length)
	if _, err := b.f.ReadAt(data, b.cap-int64(length)); err != nil {
		return err
	}

	// Decode indexes.
	b.indexes = make([]index, num)
	for i := range b.indexes {
		off := int(length) - (i+1)*indexSize
		b.indexes[i], _ = unmarshalIndex(data[off : off+indexSize])
	}

	return nil
}

func (b *Block) rebuildIndex() error {
	num := b.actx.num
	b.indexes = make([]index, 0, num)

	num = 0
	buf := make([]byte, block.EntryLengthSize)
	off := int64(headerBlockSize)
	for {
		if _, err := b.f.ReadAt(buf, off); err != nil {
			return err
		}
		length := block.EntryLength(buf)
		if length == 0 {
			break
		}
		idx := index{
			offset: off,
			length: int32(length) + block.EntryLengthSize,
		}
		b.indexes = append(b.indexes, idx)
		off = idx.EndOffset()
		num++
	}

	// Reset meta data.
	b.actx.offset = uint32(off)
	b.fo.Store(int64(b.actx.offset))
	b.actx.num = num

	return nil
}

func (b *Block) sync(ctx context.Context) error {
	// TODO(james.yin): lock
	if fo := b.fo.Load(); b.so < fo {
		if err := b.persistHeader(ctx); err != nil {
			return err
		}
		if b.full() {
			if err := b.persistIndex(ctx); err != nil {
				return err
			}
		}
		b.so = fo
	}
	return nil
}

func (b *Block) validate(ctx context.Context) error {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)
	return nil
}

func (b *Block) SetClusterInfoSource(cis block.ClusterInfoSource) {
	b.cis = cis
}
