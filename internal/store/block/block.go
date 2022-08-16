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

//go:generate mockgen -source=block.go  -destination=mock_block.go -package=block
package block

import (
	// standard libraries.
	"context"
	"errors"

	// first-party libraries.
	metapb "github.com/linkall-labs/vanus/proto/pkg/meta"

	// this project.
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
)

var (
	ErrNotEnoughSpace = errors.New("not enough space")
	ErrFull           = errors.New("full")
	ErrNotLeader      = errors.New("not leader")
	ErrOffsetExceeded = errors.New("the offset exceeded")
	ErrOffsetOnEnd    = errors.New("the offset on end")
)

type Appender interface {
	IDStr() string
	Append(ctx context.Context, entries ...Entry) ([]Entry, error)
}

type AppendContext interface {
	WriteOffset() uint32
	Full() bool
	MarkFull()
	FullEntry() Entry
}

type TwoPCAppender interface {
	NewAppendContext(last *Entry) AppendContext
	PrepareAppend(ctx context.Context, appendCtx AppendContext, entries ...Entry) ([]Entry, error)
	CommitAppend(ctx context.Context, entries ...Entry) error
	MarkFull(ctx context.Context) error
}

type Reader interface {
	IDStr() string
	Read(context.Context, int, int) ([]Entry, error)
}

type Block interface {
	ID() vanus.ID
	Close(context.Context) error
	HealthInfo() *metapb.SegmentHealthInfo
}

type ClusterInfoSource interface {
	FillClusterInfo(info *metapb.SegmentHealthInfo)
}
