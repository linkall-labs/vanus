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

package eventlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/huandu/skiplist"
	"github.com/linkall-labs/vanus/internal/controller/eventbus/block"
	"github.com/linkall-labs/vanus/internal/controller/eventbus/metadata"
	"github.com/linkall-labs/vanus/internal/controller/eventbus/volume"
	"github.com/linkall-labs/vanus/internal/kv"
	ctrlpb "github.com/linkall-labs/vsproto/pkg/controller"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrEventLogNotFound = errors.New("eventlog not found")
)

const (
	defaultAutoCreatedSegmentNumber = 3
	eventlogKeyPrefixInKVStore      = "/vanus/internal/resource/eventlog"
)

type Manager interface {
	Init(ctx context.Context, kvClient kv.Client) error
	AcquireEventLog(ctx context.Context) (*metadata.Eventlog, error)
	UpdateEventLog(ctx context.Context, els ...*metadata.Eventlog) error
	GetEventLog(ctx context.Context, id uint64) *metadata.Eventlog
	GetEventLogSegmentList(elID uint64) []*volume.Segment
	GetAppendableSegment(ctx context.Context, eli *metadata.Eventlog,
		num int) ([]*volume.Segment, error)
	UpdateSegment(ctx context.Context, req *ctrlpb.SegmentHeartbeatRequest) error
}

type eventlogManager struct {
	kvStore          kv.Client
	segmentAllocator block.Allocator
	// string, *metadata.Eventlog
	eventLogMap      sync.Map
	boundEventLogMap sync.Map

	freeEventLogMap *skiplist.SkipList
	kvMutex         sync.Mutex
	volMgr          volume.Manager
	kvClient        kv.Client
	// key: EventlogID, value is a skiplist, the id of it is Segment.ID and value if *Segment
	logMap map[uint64]*skiplist.SkipList
}

func NewManager(volMgr volume.Manager) Manager {
	return &eventlogManager{
		volMgr:          volMgr,
		freeEventLogMap: skiplist.New(skiplist.String),
	}
}

func (mgr *eventlogManager) Init(ctx context.Context, kvClient kv.Client) error {
	mgr.kvClient = kvClient
	mgr.segmentAllocator = block.NewAllocator(block.NewVolumeRoundRobin(mgr.volMgr.GetAllVolume))
	if err := mgr.segmentAllocator.Init(ctx, mgr.kvStore); err != nil {
		return err
	}
	pairs, err := mgr.kvStore.List(ctx, eventlogKeyPrefixInKVStore)
	if err != nil {
		return err
	}
	for idx := range pairs {
		pair := pairs[idx]
		elInfo := &metadata.Eventlog{}
		err := json.Unmarshal(pair.Value, elInfo)
		if err != nil {
			return err
		}
		mgr.eventLogMap.Store(filepath.Base(pair.Key), elInfo)
	}
	return mgr.initVolumeInfo(ctx)
}

func (mgr *eventlogManager) initVolumeInfo(ctx context.Context) error {
	var err error
	//ctx := context.Background()
	mgr.eventLogMap.Range(func(key, value interface{}) bool {
		elInfo := value.(*metadata.Eventlog)
		sbList := mgr.GetEventLogSegmentList(elInfo.ID)
		for idx := 0; idx < len(sbList); idx++ {
			//sb := sbList[idx]
			//sb.VolumeMeta = *(volumeMgr.GetVolumeInstanceByID(sb.VolumeID).GetMeta())
		}
		return true
	})
	return err
}

func (mgr *eventlogManager) stop(ctx context.Context) {
}

func (mgr *eventlogManager) AcquireEventLog(ctx context.Context) (*metadata.Eventlog, error) {
	ele := mgr.freeEventLogMap.Front()
	var el *metadata.Eventlog
	if ele == nil {
		_el, err := mgr.createEventLog(ctx)
		if err != nil {
			return nil, err
		}
		el = _el
	} else {
		el = ele.Value.(*metadata.Eventlog)
	}
	mgr.boundEventLogMap.Store(el.ID, el)
	if err := mgr.initializeEventLog(ctx, el); err != nil {
		return nil, err
	}
	return el, nil
}

func (mgr *eventlogManager) createEventLog(ctx context.Context) (*metadata.Eventlog, error) {
	el := &metadata.Eventlog{
		// TODO use new uuid generator
		ID: uint64(time.Now().UnixNano()),
	}
	data, _ := json.Marshal(el)
	mgr.kvMutex.Lock()
	defer mgr.kvMutex.Unlock()
	if err := mgr.kvStore.Set(ctx, mgr.getEventLogKeyInKVStore(el.ID), data); err != nil {
		return nil, err
	}
	mgr.eventLogMap.Store(el.ID, el)
	return el, nil
}

func (mgr *eventlogManager) GetEventLog(ctx context.Context, id uint64) *metadata.Eventlog {
	v, exist := mgr.eventLogMap.Load(id)

	if exist {
		return v.(*metadata.Eventlog)
	}
	return nil
}

func (mgr *eventlogManager) UpdateEventLog(ctx context.Context, els ...*metadata.Eventlog) error {
	mgr.kvMutex.Lock()
	defer mgr.kvMutex.Unlock()
	for idx := range els {
		el := els[idx]
		data, _ := json.Marshal(el)
		if err := mgr.kvStore.Set(ctx, mgr.getEventLogKeyInKVStore(el.ID), data); err != nil {
			return err
		}
	}
	return nil
}

func (mgr *eventlogManager) initializeEventLog(ctx context.Context, el *metadata.Eventlog) error {
	_, err := mgr.segmentAllocator.Pick(ctx, defaultAutoCreatedSegmentNumber, 64*1024*1024)
	if err != nil {
		return err
	}
	return nil
}

func (mgr *eventlogManager) dynamicScaleUpEventLog() error {
	return nil
}

func (mgr *eventlogManager) getEventLogKeyInKVStore(elID uint64) string {
	return strings.Join([]string{eventlogKeyPrefixInKVStore, fmt.Sprintf("%d", elID)}, "/")
}

func (mgr *eventlogManager) GetAppendableSegment(ctx context.Context,
	eli *metadata.Eventlog, num int) ([]*volume.Segment, error) {
	// TODO the HA of block can't be guaranteed before block support multiple replicas
	sl := mgr.logMap[eli.ID]
	if sl == nil {
		return nil, ErrEventLogNotFound
	}
	arr := make([]*volume.Segment, 0)
	next := sl.Front()
	hit := 0
	for hit < num && next != nil {
		sbi := next.Value.(*volume.Segment)
		next = next.Next()
		if sbi.IsAppendable() {
			continue
		}
		hit++
		arr = append(arr, sbi)
	}

	if len(arr) == 0 {
		// TODO
		//return mgr.bindSegment(ctx, eli, 1)
	}
	return arr, nil
}

func (mgr *eventlogManager) UpdateSegment(ctx context.Context, req *ctrlpb.SegmentHeartbeatRequest) error {
	//for idx := range req.HealthInfo {
	//hInfo := req.HealthInfo[idx]

	//// TODO there is problem in data structure design OPTIMIZE
	//v, exist := mgr.segmentMap.Load(hInfo.Id)
	//if !exist {
	//	log.Warning(ctx, "the block not found when heartbeat", map[string]interface{}{
	//		"segment_id": hInfo.Id,
	//	})
	//	continue
	//}
	//in := v.(*block.Segment)
	//if hInfo.IsFull {
	//	in.IsFull = true
	//
	//	next := mgr.getSegmentBlockByID(ctx, in.NextSegmentId)
	//	if next != nil {
	//		next.StartOffsetInLog = in.StartOffsetInLog + int64(in.Number)
	//		if err := mgr.updateSegmentBlockInKV(ctx, next); err != nil {
	//			log.Warning(ctx, "update the block's start_offset failed ", map[string]interface{}{
	//				"segment_id":   hInfo.Id,
	//				"next_segment": next.ID,
	//				log.KeyError:   err,
	//			})
	//			return err
	//		}
	//	}
	//}
	//in.Size = hInfo.Size
	//in.Number = hInfo.EventNumber
	//if err := mgr.updateSegmentBlockInKV(ctx, in); err != nil {
	//	log.Warning(ctx, "update the block failed ", map[string]interface{}{
	//		"segment_id": hInfo.Id,
	//		log.KeyError: err,
	//	})
	//	return err
	//}
	//}
	return nil
}

func (mgr *eventlogManager) GetEventLogSegmentList(elID uint64) []*volume.Segment {
	el := mgr.logMap[elID]
	if el == nil {
		return nil
	}
	var arr []*volume.Segment
	next := el.Front()
	for next != nil {
		arr = append(arr, next.Value.(*volume.Segment))
		next = next.Next()
	}
	return arr
}
