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

package block

import (
	// standard libraries
	"context"
	"errors"
	"os"
	"path/filepath"

	// third-party libraries
	"go.uber.org/atomic"

	// first-party libraries
	"github.com/linkall-labs/vsproto/pkg/meta"

	// this project
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/observability"
)

var (
	ErrNoEnoughCapacity = errors.New("no enough capacity")
	ErrFull             = errors.New("full")
	ErrNotLeader        = errors.New("not leader")
	ErrOffsetExceeded   = errors.New("the offset exceeded")
	ErrOffsetOnEnd      = errors.New("the offset on end")
)

type SegmentBlockWriter interface {
	Append(context.Context, ...Entry) error
	CloseWrite(context.Context) error
	IsAppendable() bool
}

type SegmentBlockReader interface {
	Read(context.Context, int, int) ([]Entry, error)
	CloseRead(context.Context) error
	IsReadable() bool
}

type SegmentBlock interface {
	SegmentBlockWriter
	SegmentBlockReader

	Path() string
	IsFull() bool
	IsEmpty() bool
	SegmentBlockID() vanus.ID
	Close(context.Context) error
	Initialize(context.Context) error
	HealthInfo() *meta.SegmentHealthInfo
}

func CreateFileSegmentBlock(ctx context.Context, id vanus.ID, path string, capacity int64) (SegmentBlock, error) {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)

	b := &fileBlock{
		id:   id,
		path: path,
		cap:  capacity,
		wo:   *atomic.NewInt64(fileBlockHeaderSize),
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err = f.Truncate(capacity); err != nil {
		return nil, err
	}
	if _, err = f.Seek(fileBlockHeaderSize, 0); err != nil {
		return nil, err
	}
	b.appendable.Store(true)
	b.readable.Store(true)
	b.full.Store(false)
	b.f = f
	if err = b.persistHeader(ctx); err != nil {
		return nil, err
	}

	return b, nil
}

func OpenFileSegmentBlock(ctx context.Context, path string) (SegmentBlock, error) {
	observability.EntryMark(ctx)
	defer observability.LeaveMark(ctx)
	id, err := vanus.NewIDFromString(filepath.Base(path))
	if err != nil {
		return nil, err
	}
	b := &fileBlock{
		id:   id,
		path: path,
	}
	b.appendable.Store(true)
	b.readable.Store(true)
	b.full.Store(false)
	// TODO: use direct IO
	f, err := os.OpenFile(path, os.O_RDWR|os.O_SYNC, 0666)
	if err != nil {
		return nil, err
	}
	b.f = f
	return b, nil
}
