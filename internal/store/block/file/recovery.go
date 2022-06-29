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
	"os"
	"path/filepath"

	// this project.
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/observability/log"
)

const (
	defaultDirPerm = 0o755
)

func Recover(blockDir string) (map[vanus.ID]*Block, error) {
	// Make sure the block directory exists.
	if err := os.MkdirAll(blockDir, defaultDirPerm); err != nil {
		return nil, err
	}

	files, err := os.ReadDir(blockDir)
	if err != nil {
		return nil, err
	}
	files = filterRegularBlock(files)

	blocks := make(map[vanus.ID]*Block, len(files))
	for _, file := range files {
		filename := file.Name()
		blockID, err2 := vanus.NewIDFromString(filename[:len(filename)-len(blockExt)])
		if err2 != nil {
			err = err2
		}

		path := filepath.Join(blockDir, filename)
		block, err2 := Open(context.Background(), path)
		if err2 != nil {
			err = err2
			break
		}
		blocks[blockID] = block
	}

	if err != nil {
		for _, block := range blocks {
			_ = block.Close(context.Background())
		}
		return nil, err
	}

	return blocks, nil
}

func filterRegularBlock(entries []os.DirEntry) []os.DirEntry {
	if len(entries) == 0 {
		return entries
	}

	n := 0
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		if filepath.Ext(entry.Name()) != blockExt {
			continue
		}
		entries[n] = entry
		n++
	}
	entries = entries[:n]
	return entries
}

func (b *Block) recover(ctx context.Context) error {
	if err := b.loadHeader(ctx); err != nil {
		return err
	}

	if err := b.loadIndex(ctx); err != nil {
		return err
	}

	if err := b.correctMeta(); err != nil {
		return err
	}

	if err := b.validate(ctx); err != nil {
		return err
	}

	log.Debug(ctx, "The block was loaded.", map[string]interface{}{
		"block_id": b.id,
	})

	return nil
}
