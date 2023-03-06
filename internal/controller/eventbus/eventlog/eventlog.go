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

//go:generate mockgen -source=eventlog.go -destination=mock_eventlog.go -package=eventlog
package eventlog

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/huandu/skiplist"

	"github.com/vanus-labs/vanus/observability/log"
	"github.com/vanus-labs/vanus/observability/metrics"
	"github.com/vanus-labs/vanus/pkg/errors"
	"github.com/vanus-labs/vanus/proto/pkg/segment"

	"github.com/vanus-labs/vanus/internal/controller/eventbus/block"
	"github.com/vanus-labs/vanus/internal/controller/eventbus/metadata"
	"github.com/vanus-labs/vanus/internal/controller/eventbus/volume"
	"github.com/vanus-labs/vanus/internal/kv"
	"github.com/vanus-labs/vanus/internal/primitive/vanus"
)

const (
	defaultAppendableSegmentNumber     = 2
	defaultSegmentReplicaNumber        = 3
	defaultSegmentExpiredTime          = 72 * time.Hour
	defaultScaleInterval               = time.Second
	defaultCleanInterval               = time.Second
	defaultCheckExpiredSegmentInterval = time.Minute
)

type Manager interface {
	Run(ctx context.Context, kvClient kv.Client, startTask bool) error
	Stop()
	AcquireEventLog(ctx context.Context, eventbusID vanus.ID, eventbusName string) (*metadata.Eventlog, error)
	GetEventLog(ctx context.Context, id vanus.ID) *metadata.Eventlog
	DeleteEventlog(ctx context.Context, id vanus.ID)
	GetBlock(id vanus.ID) *metadata.Block
	UpdateSegmentReplicas(ctx context.Context, segID vanus.ID, term uint64) error
	GetEventLogSegmentList(elID vanus.ID) []Segment
	GetAppendableSegment(ctx context.Context, eli *metadata.Eventlog, num int) ([]Segment, error)
	UpdateSegment(ctx context.Context, m map[string][]Segment)
	GetSegmentByBlockID(block *metadata.Block) (Segment, error)
}

var mgr = &eventlogManager{
	segmentReplicaNum:           defaultSegmentReplicaNumber,
	scaleInterval:               defaultScaleInterval,
	cleanInterval:               defaultCleanInterval,
	checkSegmentExpiredInterval: defaultCheckExpiredSegmentInterval,
	segmentExpiredTime:          defaultSegmentExpiredTime,
}

type eventlogManager struct {
	allocator block.Allocator

	// string, *eventlog
	eventLogMap sync.Map

	// blockID, *metadata.Block
	globalBlockMap sync.Map

	// segmentID, *segment
	globalSegmentMap sync.Map

	volMgr   volume.Manager
	kvClient kv.Client
	cancel   func()
	mutex    sync.Mutex
	// vanus.ID *Segment
	segmentNeedBeClean          sync.Map
	segmentReplicaNum           uint
	scaleInterval               time.Duration
	cleanInterval               time.Duration
	checkSegmentExpiredInterval time.Duration
	segmentExpiredTime          time.Duration
	createSegmentMutex          sync.Mutex
}

func NewManager(volMgr volume.Manager, replicaNum uint, defaultBlockSize int64) Manager {
	mgr.volMgr = volMgr
	if replicaNum > 0 {
		mgr.segmentReplicaNum = replicaNum
	}
	mgr.allocator = block.NewAllocator(defaultBlockSize,
		block.NewVolumeRoundRobin(mgr.volMgr.GetAllActiveVolumes))
	return mgr
}

func (mgr *eventlogManager) Run(ctx context.Context, kvClient kv.Client, startTask bool) error {
	if mgr.checkSegmentExpiredInterval == 0 {
		mgr.checkSegmentExpiredInterval = defaultCheckExpiredSegmentInterval
	}
	if mgr.scaleInterval == 0 {
		mgr.scaleInterval = defaultScaleInterval
	}
	if mgr.cleanInterval == 0 {
		mgr.cleanInterval = defaultCleanInterval
	}
	mgr.kvClient = kvClient
	if err := mgr.allocator.Run(ctx, mgr.kvClient, true); err != nil {
		return err
	}
	pairs, err := mgr.kvClient.List(ctx, metadata.EventlogKeyPrefixInKVStore)
	if err != nil {
		return err
	}
	for idx := range pairs {
		pair := pairs[idx]
		elMD := &metadata.Eventlog{}
		err1 := json.Unmarshal(pair.Value, elMD)
		if err1 != nil {
			return err1
		}
		el, err2 := newEventlog(ctx, elMD, mgr.kvClient, true)
		if err2 != nil {
			return err2
		}
		mgr.eventLogMap.Store(elMD.ID.Key(), el)

		for _, ptr := range el.getAllSegments() {
			mgr.globalSegmentMap.Store(ptr.ID.Key(), ptr)
			for _, v := range ptr.Replicas.Peers {
				mgr.globalBlockMap.Store(v.ID.Key(), v)
			}
		}
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	mgr.cancel = cancel
	if startTask {
		go mgr.dynamicScaleUpEventLog(cancelCtx)
		go mgr.cleanAbnormalSegment(cancelCtx)
		go mgr.checkSegmentExpired(cancelCtx)
	}
	go mgr.recordMetrics(cancelCtx)
	return nil
}

func (mgr *eventlogManager) Stop() {
	mgr.stop()
	mgr.allocator.Stop()
}

func (mgr *eventlogManager) AcquireEventLog(ctx context.Context, eventbusID vanus.ID,
	eventbusName string,
) (*metadata.Eventlog, error) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	id, err := vanus.NewID()
	if err != nil {
		log.Warning(ctx, "failed to create eventlog ID", map[string]interface{}{
			log.KeyError:  err,
			"eventbus_id": eventbusID,
		})
		return nil, err
	}
	elMD := &metadata.Eventlog{
		ID:           id,
		EventbusName: eventbusName,
		EventbusID:   eventbusID,
	}
	data, _ := json.Marshal(elMD)
	if err := mgr.kvClient.Set(ctx, metadata.GetEventlogMetadataKey(elMD.ID), data); err != nil {
		return nil, err
	}

	el, err := mgr.initializeEventLog(ctx, elMD)
	if err != nil {
		elMD.EventbusID = vanus.EmptyID()
		return nil, err
	}

	mgr.eventLogMap.Store(el.md.ID.Key(), el)
	log.Info(ctx, "an eventlog created", map[string]interface{}{
		"eventbus_id": elMD.EventbusID.Key(),
		"eventlog_id": elMD.ID.Key(),
	})
	return elMD, nil
}

func (mgr *eventlogManager) GetEventLog(_ context.Context, id vanus.ID) *metadata.Eventlog {
	el := mgr.getEventLog(id)
	if el != nil {
		el.md.SegmentNumber = el.size()
		return el.md
	}
	return nil
}

func (mgr *eventlogManager) getEventLog(id vanus.ID) *eventlog {
	v, exist := mgr.eventLogMap.Load(id.Key())

	if exist {
		return v.(*eventlog)
	}
	return nil
}

func (mgr *eventlogManager) DeleteEventlog(ctx context.Context, id vanus.ID) {
	v, exist := mgr.eventLogMap.LoadAndDelete(id.Key())
	if !exist {
		return
	}
	el, _ := v.(*eventlog)
	head := el.head()
	for head != nil {
		err := el.deleteHead(ctx)
		if err != nil {
			log.Warning(ctx, "delete eventlog error when deleting segment", map[string]interface{}{
				log.KeyError:  err,
				"head_id":     head.ID.Key(),
				"eventlog_id": id.Key(),
			})
		}
		_, ok := mgr.segmentNeedBeClean.LoadOrStore(head.ID.Key(), head)
		if ok {
			metrics.SegmentDeletedCounterVec.WithLabelValues(
				metrics.LabelSegmentDeletedBecauseDeleted).Inc()
		}
		head = el.head()
	}
	if err := mgr.kvClient.Delete(ctx, metadata.GetEventlogMetadataKey(id)); err != nil {
		log.Warning(ctx, "deleting eventlog metadata in kv error", map[string]interface{}{
			log.KeyError:  err,
			"eventlog_id": id.Key(),
		})
	}
}

func (mgr *eventlogManager) GetAppendableSegment(ctx context.Context,
	eli *metadata.Eventlog, num int,
) ([]Segment, error) {
	result := make([]Segment, 0, num)
	if eli == nil || num == 0 {
		return result, nil
	}

	v, exist := mgr.eventLogMap.Load(eli.ID.Key())
	if !exist {
		return nil, errors.ErrEventLogNotFound
	}

	el, _ := v.(*eventlog)
	s := el.currentAppendableSegment()
	if s == nil && len(result) == 0 {
		seg, err := mgr.createSegment(ctx, el)
		if err != nil {
			return nil, err
		}

		if err = el.add(ctx, seg); err != nil {
			// preparing to cleaning
			log.Warning(ctx, "add new segment to eventlog failed when initialized", map[string]interface{}{
				log.KeyError:  err,
				"eventlog_id": el.md.ID,
			})
			_, ok := mgr.segmentNeedBeClean.LoadOrStore(seg.ID.Key(), seg)
			if !ok {
				metrics.SegmentDeletedCounterVec.WithLabelValues(
					metrics.LabelSegmentDeletedBecauseCreateFailed).Inc()
			}
			return nil, err
		}
		metrics.SegmentCreatedByCacheMissing.Inc()
		s = el.currentAppendableSegment()
	}

	return el.listOfRight(s, true), nil
}

func (mgr *eventlogManager) UpdateSegment(ctx context.Context, m map[string][]Segment) {
	mgr.mutex.Lock()
	defer mgr.mutex.Unlock()

	// iterate eventlog
	for eventlogID, segments := range m {
		v, exist := mgr.eventLogMap.Load(eventlogID)
		if !exist {
			segmentIDs := make([]string, 0)
			for idx := range segments {
				segmentIDs = append(segmentIDs, segments[idx].ID.String())
			}
			log.Warning(ctx, "eventlog not found", map[string]interface{}{
				"id":       eventlogID,
				"segments": segmentIDs,
			})
			continue
		}
		el, _ := v.(*eventlog)
		el.lock()
		for idx := range segments {
			newSeg := segments[idx]
			seg := el.get(newSeg.ID)
			if seg == nil {
				log.Warning(ctx, "segment not found in eventlog", map[string]interface{}{
					"id":      eventlogID,
					"segment": seg.String(),
				})
				continue
			}
			// TODO(wenfeng.wang) Don't update state in isNeedUpdate, rename?
			if seg.isNeedUpdate(newSeg) {
				err := el.updateSegment(ctx, seg)
				if err != nil {
					log.Warning(ctx, "update segment's metadata failed", map[string]interface{}{
						log.KeyError: err,
						"segment":    seg.String(),
						"eventlog":   el.md.ID.String(),
					})
				}
			}
		}
		el.unlock()
	}
}

func (mgr *eventlogManager) GetEventLogSegmentList(elID vanus.ID) []Segment {
	result := make([]Segment, 0)
	v, exist := mgr.eventLogMap.Load(elID.Key())
	if !exist {
		return result
	}

	el, _ := v.(*eventlog)
	list := el.getAllSegments()
	el.lock()
	defer el.unlock()

	for _, v := range list {
		result = append(result, v.Copy())
	}
	return result
}

func (mgr *eventlogManager) GetBlock(id vanus.ID) *metadata.Block {
	v, exist := mgr.globalBlockMap.Load(id.Key())
	if !exist {
		return nil
	}
	return v.(*metadata.Block)
}

func (mgr *eventlogManager) getSegment(id vanus.ID) *Segment {
	v, exist := mgr.globalSegmentMap.Load(id.Key())
	if !exist {
		return nil
	}
	return v.(*Segment)
}

func (mgr *eventlogManager) UpdateSegmentReplicas(ctx context.Context, leaderID vanus.ID, term uint64) error {
	blk := mgr.GetBlock(leaderID)
	if blk == nil {
		return errors.ErrBlockNotFound
	}

	seg := mgr.getSegment(blk.SegmentID)
	if seg == nil {
		return errors.ErrSegmentNotFound
	}

	if seg.Replicas.Term >= term {
		return nil
	}

	el := mgr.getEventLog(seg.EventLogID)
	if el == nil {
		return errors.ErrEventLogNotFound
	}
	el.lock()
	defer el.unlock()

	seg.Replicas.Leader = leaderID.Uint64()
	seg.Replicas.Term = term
	data, _ := json.Marshal(seg)
	key := filepath.Join(metadata.SegmentKeyPrefixInKVStore, seg.ID.String())
	if err := mgr.kvClient.Set(ctx, key, data); err != nil {
		log.Warning(ctx, "update segment's metadata failed", map[string]interface{}{
			log.KeyError: err,
			"segment":    seg.String(),
		})
		return errors.ErrInvalidSegment.WithMessage("update segment to etcd error").Wrap(err)
	}
	return nil
}

func (mgr *eventlogManager) GetSegmentByBlockID(block *metadata.Block) (Segment, error) {
	v, exist := mgr.eventLogMap.Load(block.EventlogID.Key())
	if !exist {
		return Segment{}, errors.ErrEventLogNotFound
	}
	el, _ := v.(*eventlog)
	el.rLock()
	defer el.rUnlock()
	return el.get(block.SegmentID).Copy(), nil
}

func (mgr *eventlogManager) stop() {
	mgr.cancel()
}

func (mgr *eventlogManager) getSegmentTopology(_ context.Context, seg Segment) map[uint64]string {
	addrs := map[uint64]string{}
	for _, v := range seg.Replicas.Peers {
		ins := mgr.volMgr.GetVolumeInstanceByID(v.VolumeID)
		if ins == nil {
			log.Error(context.TODO(), "the volume of block not found", map[string]interface{}{
				"segment_id": seg.ID,
				"block_id":   v.ID,
				"volume_id":  v.VolumeID,
			})
			return map[uint64]string{}
		}
		addrs[v.ID.Uint64()] = ins.Address()
	}
	return addrs
}

func (mgr *eventlogManager) initializeEventLog(ctx context.Context, md *metadata.Eventlog) (*eventlog, error) {
	el, err := newEventlog(ctx, md, mgr.kvClient, false)
	if err != nil {
		return nil, err
	}
	for idx := 0; idx < defaultAppendableSegmentNumber; idx++ {
		seg, err := mgr.createSegment(ctx, el)
		if err != nil {
			return nil, err
		}
		if err = el.add(ctx, seg); err != nil {
			log.Warning(ctx, "add new segment to eventlog failed when initialized", map[string]interface{}{
				log.KeyError:  err,
				"eventlog_id": el.md.ID,
			})
			_, ok := mgr.segmentNeedBeClean.LoadOrStore(seg.ID.Key(), seg)
			if !ok {
				metrics.SegmentDeletedCounterVec.WithLabelValues(
					metrics.LabelSegmentDeletedBecauseCreateFailed).Inc()
			}
			return nil, err
		}
		metrics.SegmentCreatedByCacheMissing.Inc()
	}
	return el, nil
}

func (mgr *eventlogManager) dynamicScaleUpEventLog(ctx context.Context) {
	ticker := time.NewTicker(mgr.scaleInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "the task of dynamic-scale stopped", nil)
			return
		case <-ticker.C:
			count := 0
			mgr.eventLogMap.Range(func(key, value interface{}) bool {
				el, ok := value.(*eventlog)
				if !ok {
					log.Error(ctx, "assert failed in dynamicScaleUpEventLog", map[string]interface{}{
						"key": key,
					})
					return true
				}
				for el.appendableSegmentNumber() < defaultAppendableSegmentNumber {
					seg, err := mgr.createSegment(ctx, el)
					if err != nil {
						log.Warning(ctx, "create new segment failed", map[string]interface{}{
							log.KeyError:  err,
							"eventlog_id": el.md.ID,
						})
						return true
					}

					if err = el.add(ctx, seg); err != nil {
						log.Warning(ctx, "add new segment to eventlog failed when scale", map[string]interface{}{
							log.KeyError:  err,
							"eventlog_id": el.md.ID,
						})
						_, ok := mgr.segmentNeedBeClean.LoadOrStore(seg.ID.Key(), seg)
						if !ok {
							metrics.SegmentDeletedCounterVec.WithLabelValues(
								metrics.LabelSegmentDeletedBecauseCreateFailed).Inc()
						}
						return true
					}

					metrics.SegmentCreatedByScaleTask.Inc()
					log.Info(ctx, "the new segment created", map[string]interface{}{
						"segment_id":  seg.ID.Key(),
						"eventlog_id": el.md.ID.Key(),
					})
					count++
				}
				return true
			})
			log.Debug(ctx, "finished to provisioning segments", map[string]interface{}{
				"count": count,
			})
		}
	}
}

func (mgr *eventlogManager) cleanAbnormalSegment(ctx context.Context) {
	ticker := time.NewTicker(mgr.cleanInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "the task of clean-abnormal-segment stopped", nil)
			return
		case <-ticker.C:
			count := 0
			mgr.segmentNeedBeClean.Range(func(key, value interface{}) bool {
				v, ok := value.(*Segment)
				if !ok {
					return true
				}

				for _, blk := range v.Replicas.Peers {
					ins := mgr.volMgr.GetVolumeInstanceByID(blk.VolumeID)
					infos := map[string]interface{}{
						"segment_id":   v.ID.Key(),
						"size":         v.Size,
						"number":       v.Number,
						"start_offset": v.StartOffsetInLog,
						"block_id":     blk.ID,
					}
					err := ins.DeleteBlock(ctx, blk.ID)
					if err != nil {
						infos[log.KeyError] = err
						log.Warning(ctx, "delete block failed", infos)
						continue
					}
					mgr.globalBlockMap.Delete(blk.ID.Key())
					err = mgr.kvClient.Delete(ctx, metadata.GetBlockMetadataKey(blk.VolumeID, blk.ID))
					if err != nil {
						infos[log.KeyError] = err
						log.Warning(ctx, "delete block metadata in kv failed", infos)
						continue
					}
					log.Info(ctx, "the block has been deleted", infos)
				}

				if err := mgr.kvClient.Delete(ctx, metadata.GetSegmentMetadataKey(v.ID)); err != nil {
					log.Warning(ctx, "clean segment data in KV failed", map[string]interface{}{
						log.KeyError: err,
						"segment":    v.String(),
					})
				}
				mgr.globalSegmentMap.Delete(v.ID.Key())
				mgr.segmentNeedBeClean.Delete(key)
				log.Info(ctx, "the segment has been deleted", map[string]interface{}{
					"segment_id":   v.ID.Key(),
					"size":         v.Size,
					"number":       v.Number,
					"start_offset": v.StartOffsetInLog,
					"eventlog_id":  v.EventLogID.Key(),
				})
				count++
				return true
			})
			log.Debug(ctx, "clean segment task completed", map[string]interface{}{
				"segment_cleaned": count,
			})
		}
	}
}

func (mgr *eventlogManager) checkSegmentExpired(ctx context.Context) {
	ticker := time.NewTicker(mgr.checkSegmentExpiredInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Info(ctx, "the task of check-segment-expired stopped", nil)
			return
		case <-ticker.C:
			count := 0
			executionID := uuid.NewString()
			mgr.eventLogMap.Range(func(key, value interface{}) bool {
				elog, _ := value.(*eventlog)
				head := elog.head()
				checkCtx := context.Background()
				for head != nil {
					switch {
					case head.LastEventBornTime.Second() == 0:
						// TODO(wenfeng.wang) fix if set
						head.LastEventBornTime = time.Now().Add(mgr.segmentExpiredTime)
						elog.lock()
						if err := elog.updateSegment(checkCtx, head); err != nil {
							log.Warning(ctx, "update segment's metadata failed", map[string]interface{}{
								log.KeyError: err,
								"segment":    head.String(),
								"eventlog":   elog.md.ID.String(),
							})
							head.LastEventBornTime = time.Time{}
						}
						elog.unlock()
						return true
					case !head.isFull():
						return true
					case time.Since(head.LastEventBornTime.Add(mgr.segmentExpiredTime)) > 0:
						err := elog.deleteHead(ctx)
						if err != nil {
							log.Warning(ctx, "delete segment error", map[string]interface{}{
								log.KeyError:       err,
								"execution_id":     executionID,
								"last_event_time":  head.LastEventBornTime,
								"first_event_time": head.FirstEventBornTime,
								"time":             time.Now(),
							})
							return true
						}
						count++
						log.Info(ctx, "delete segment success", map[string]interface{}{
							"execution_id":     executionID,
							"log_id":           elog.md.ID.String(),
							"last_event_time":  head.LastEventBornTime,
							"first_event_time": head.FirstEventBornTime,
							"time":             time.Now(),
							"number":           head.Number,
						})
						_, ok := mgr.segmentNeedBeClean.LoadOrStore(head.ID.Key(), head)
						if !ok {
							metrics.SegmentDeletedCounterVec.WithLabelValues(
								metrics.LabelSegmentDeletedBecauseExpired).Inc()
						}
					default:
						return true
					}
					head = elog.head()
				}
				return true
			})
			log.Info(ctx, "check-segment-expired completed", map[string]interface{}{
				"count":        count,
				"execution_id": executionID,
			})
		}
	}
}

func (mgr *eventlogManager) recordMetrics(ctx context.Context) {
	t := time.NewTicker(time.Second)
	for {
		select {
		case <-t.C:
			mgr.mutex.Lock()

			nameMap := map[string]string{}
			metrics.EventlogGaugeVec.Reset()
			mgr.eventLogMap.Range(func(_, value any) bool {
				el, _ := value.(*eventlog)
				metrics.EventlogGaugeVec.WithLabelValues(el.md.Eventbus()).Inc()
				nameMap[el.md.ID.Key()] = el.md.EventbusName
				return true
			})

			metrics.SegmentGaugeVec.Reset()
			metrics.SegmentCapacityGaugeVec.Reset()
			metrics.SegmentSizeGaugeVec.Reset()
			metrics.SegmentEventNumberGaugeVec.Reset()
			mgr.globalSegmentMap.Range(func(key, value any) bool {
				seg, _ := value.(*Segment)
				elID := seg.EventLogID.Key()

				metrics.SegmentGaugeVec.WithLabelValues(nameMap[elID], elID, string(seg.State)).Inc()
				metrics.SegmentCapacityGaugeVec.WithLabelValues(nameMap[elID], elID,
					string(seg.State)).Add(float64(seg.Capacity))
				metrics.SegmentSizeGaugeVec.WithLabelValues(nameMap[elID], elID,
					string(seg.State)).Add(float64(seg.Size))
				metrics.SegmentEventNumberGaugeVec.WithLabelValues(nameMap[elID],
					elID, string(seg.State)).Add(float64(seg.Number))
				return true
			})
			mgr.mutex.Unlock()
		case <-ctx.Done():
			log.Info(ctx, "record leadership exiting...", nil)
			return
		}
	}
}

func (mgr *eventlogManager) createSegment(ctx context.Context, el *eventlog) (*Segment, error) {
	mgr.createSegmentMutex.Lock()
	defer mgr.createSegmentMutex.Unlock()

	seg, err := mgr.generateSegment(ctx, el)
	defer func() {
		// preparing to cleaning
		if err != nil {
			if seg != nil {
				_, ok := mgr.segmentNeedBeClean.LoadOrStore(seg.ID.Key(), seg)
				if !ok {
					metrics.SegmentDeletedCounterVec.WithLabelValues(
						metrics.LabelSegmentDeletedBecauseCreateFailed).Inc()
				}
			}
		}
	}()
	if err != nil {
		return nil, err
	}
	seg.EventLogID = el.md.ID

	cur := el.currentAppendableSegment()
	if cur == nil {
		blk, err := mgr.whichIsLeader(seg.Replicas.Peers)
		if err != nil {
			return nil, err
		}
		seg.Replicas.Leader = blk.ID.Uint64()
	} else {
		set := false
		for _, blk := range seg.Replicas.Peers {
			if blk.VolumeID.Equals(cur.GetLeaderBlock().VolumeID) {
				seg.Replicas.Leader = blk.ID.Uint64()
				set = true
				break
			}
		}
		if !set {
			log.Error(ctx, "failed to set leader block", map[string]interface{}{
				"eventlog":                   el.md.ID.Key(),
				"eventbus":                   el.md.EventbusName,
				"previous_segment_volume_id": cur.GetLeaderBlock().VolumeID,
				"replicas":                   fmt.Sprintf("%v", seg.Replicas),
				"segment":                    seg.ID.Key(),
			})
			return nil, errors.ErrInvalidSegment
		}
	}

	ins := mgr.volMgr.GetVolumeInstanceByID(seg.GetLeaderBlock().VolumeID)
	srv := ins.GetServer()
	if srv == nil {
		return nil, errors.ErrVolumeInstanceNoServer
	}
	_, err = srv.GetClient().ActivateSegment(ctx, &segment.ActivateSegmentRequest{
		EventLogId:     seg.EventLogID.Uint64(),
		ReplicaGroupId: seg.Replicas.ID.Uint64(),
		Replicas:       mgr.getSegmentTopology(ctx, *seg),
	})

	if err != nil {
		log.Warning(context.TODO(), "activate segment failed", map[string]interface{}{
			log.KeyError: err,
		})
		return nil, err
	}

	for _, v := range seg.Replicas.Peers {
		v.SegmentID = seg.ID
		v.EventlogID = seg.EventLogID
		data, _ := json.Marshal(v)
		key := filepath.Join(metadata.BlockKeyPrefixInKVStore, v.VolumeID.Key(), v.ID.String())
		if err = mgr.kvClient.Set(ctx, key, data); err != nil {
			log.Error(ctx, "save segment's data to kv store failed", map[string]interface{}{
				log.KeyError: err,
				"segment":    seg.String(),
			})
			return nil, err
		}
	}

	seg.State = StateWorking
	data, _ := json.Marshal(seg)
	key := filepath.Join(metadata.SegmentKeyPrefixInKVStore, seg.ID.String())
	if err = mgr.kvClient.Set(ctx, key, data); err != nil {
		log.Error(ctx, "update segment's metadata failed", map[string]interface{}{
			log.KeyError: err,
			"segment":    seg.String(),
		})
		return nil, err
	}

	for _, v := range seg.Replicas.Peers {
		mgr.globalBlockMap.Store(v.ID.Key(), v)
	}
	mgr.globalSegmentMap.Store(seg.ID.Key(), seg)
	return seg, nil
}

func (mgr *eventlogManager) generateSegment(ctx context.Context, el *eventlog) (*Segment, error) {
	var seg *Segment
	cur := el.currentAppendableSegment()
	var blocks []*metadata.Block
	var err error
	if cur == nil {
		blocks, err = mgr.allocator.Pick(ctx, int(mgr.segmentReplicaNum))
	} else {
		// make sure segments of one eventlog located in one SegmentServer
		volumes := make([]vanus.ID, 0)
		for _, peer := range cur.Replicas.Peers {
			volumes = append(volumes, peer.VolumeID)
		}
		blocks, err = mgr.allocator.PickByVolumes(ctx, volumes)
	}

	if err != nil {
		return nil, err
	}
	blockMap := make(map[uint64]*metadata.Block)
	for _, v := range blocks {
		blockMap[v.ID.Uint64()] = v
	}

	id1, err := vanus.NewID()
	if err != nil {
		log.Warning(ctx, "failed to create segment ID", map[string]interface{}{
			log.KeyError: err,
		})
		return nil, err
	}
	id2, err := vanus.NewID()
	if err != nil {
		log.Warning(ctx, "failed to create raft replica ID", map[string]interface{}{
			log.KeyError: err,
		})
		return nil, err
	}
	seg = &Segment{
		ID:       id1,
		Capacity: blocks[0].Capacity,
		Replicas: &ReplicaGroup{
			ID:        id2,
			Peers:     blockMap,
			CreateAt:  time.Now(),
			DestroyAt: time.Now(),
		},
		State: StateCreated,
	}

	data, _ := json.Marshal(seg)
	log.Debug(context.TODO(), "generate segment", map[string]interface{}{
		"data": string(data),
	})
	if err = mgr.kvClient.Set(ctx, filepath.Join(metadata.SegmentKeyPrefixInKVStore, seg.ID.String()), data); err != nil {
		log.Error(ctx, "save segment's data to kv store failed", map[string]interface{}{
			log.KeyError: err,
			"segment":    seg.String(),
		})
		_, ok := mgr.segmentNeedBeClean.LoadOrStore(seg.ID.Key(), seg)
		if !ok {
			metrics.SegmentDeletedCounterVec.WithLabelValues(
				metrics.LabelSegmentDeletedBecauseCreateFailed).Inc()
		}
		return nil, err
	}
	return seg, nil
}

func (mgr *eventlogManager) whichIsLeader(raftGroup map[uint64]*metadata.Block) (*metadata.Block, error) {
	countMap := map[uint64]int{}
	mgr.globalSegmentMap.Range(func(key, value any) bool {
		seg, _ := value.(*Segment)
		if seg.State == StateWorking {
			vID := seg.GetLeaderBlock().VolumeID.Uint64()
			count := countMap[vID]
			countMap[vID] = count + 1
		}
		return true
	})

	volumeMap := make(map[uint64]*metadata.Block, len(raftGroup))
	orderArr := make([]uint64, 0)
	for _, node := range raftGroup {
		volumeMap[node.VolumeID.Uint64()] = node
		orderArr = append(orderArr, node.VolumeID.Uint64())
	}
	sort.Slice(orderArr, func(i, j int) bool {
		return countMap[orderArr[i]] < countMap[orderArr[j]] ||
			orderArr[i] < orderArr[j]
	})

	return volumeMap[orderArr[0]], nil
}

type eventlog struct {
	// uint64, *Segment
	segmentList *skiplist.SkipList
	md          *metadata.Eventlog
	writePtr    *Segment
	kvClient    kv.Client
	mutex       sync.RWMutex
	// Why
	segments []vanus.ID
}

// newEventlog create an object in memory. if needLoad is true, there will read metadata
// from kv store. the eventlog.segmentList should be built after call this method.
func newEventlog(ctx context.Context, md *metadata.Eventlog, kvClient kv.Client, needLoad bool) (*eventlog, error) {
	el := &eventlog{
		segmentList: skiplist.New(skiplist.Uint64),
		md:          md,
		kvClient:    kvClient,
		segments:    []vanus.ID{},
	}
	if !needLoad {
		return el, nil
	}

	pairs, err := kvClient.List(ctx, filepath.Join(metadata.EventlogSegmentsKeyPrefixInKVStore, md.ID.String()))
	if err != nil {
		return nil, err
	}
	segmentIDs := make([]vanus.ID, 0)
	for _, v := range pairs {
		id := new(struct {
			SegmentID vanus.ID `json:"segment_id"`
		})
		if err1 := json.Unmarshal(v.Value, id); err1 != nil {
			return nil, errors.ErrUnmarshall.Wrap(err1)
		}
		segmentIDs = append(segmentIDs, id.SegmentID)
	}
	el.segments = segmentIDs
	for _, v := range el.segments {
		data, err := kvClient.Get(ctx, filepath.Join(metadata.SegmentKeyPrefixInKVStore, v.String()))
		if err != nil {
			return nil, err
		}
		seg := &Segment{}

		if err = json.Unmarshal(data, seg); err != nil {
			return nil, err
		}
		el.segmentList.Set(seg.ID.Uint64(), seg)
	}
	return el, nil
}

func (el *eventlog) get(segID vanus.ID) *Segment {
	v := el.segmentList.Get(segID.Uint64())
	if v == nil {
		return nil
	}
	return v.Value.(*Segment)
}

func (el *eventlog) appendableSegmentNumber() int {
	cur := el.currentAppendableSegment()
	if cur == nil {
		return 0
	}

	return len(el.listOfRight(cur, true))
}

func (el *eventlog) currentAppendableSegment() *Segment {
	el.mutex.Lock()
	defer el.mutex.Unlock()
	if el.size() == 0 {
		return nil
	}

	if el.writePtr == nil {
		head := el.segmentList.Front()
		for head != nil {
			s, _ := head.Value.(*Segment)
			if s.IsAppendable() {
				el.writePtr = s
				break
			}
			head = head.Next()
		}
	}
	if el.writePtr != nil && !el.writePtr.IsAppendable() {
		node := el.segmentList.Get(el.writePtr.ID.Uint64())
		if node == nil {
			return nil
		}
		next := node.Next()
		if next == nil {
			return nil
		}
		el.writePtr, _ = next.Value.(*Segment)
	}
	return el.writePtr
}

// add a segment to eventlog, the metadata of this eventlog will be updated, but the segment's metadata should be
// updated after call this method.
func (el *eventlog) add(ctx context.Context, seg *Segment) error {
	el.mutex.Lock()
	defer el.mutex.Unlock()

	if !seg.isReady() {
		return errors.ErrInvalidSegment
	}
	if el.get(seg.ID) != nil {
		return nil
	}
	last := el.tail()
	if last != nil {
		last.NextSegmentID = seg.ID
		seg.PreviousSegmentID = last.ID
		if last.State == StateFrozen {
			seg.StartOffsetInLog = last.StartOffsetInLog + int64(last.Number)
		}
	}
	s := new(struct {
		SegmentID vanus.ID `json:"segment_id"`
	})
	s.SegmentID = seg.ID
	data, _ := json.Marshal(s)
	if err := el.kvClient.Set(ctx, metadata.GetEventlogSegmentsMetadataKey(el.md.ID, seg.ID), data); err != nil {
		last.NextSegmentID = vanus.EmptyID()
		seg.PreviousSegmentID = vanus.EmptyID()
		return err
	}

	if last != nil {
		data, _ = json.Marshal(last)
		if err := el.kvClient.Set(ctx, metadata.GetSegmentMetadataKey(last.ID), data); err != nil {
			// TODO clean when failed
			last.NextSegmentID = vanus.EmptyID()
			seg.PreviousSegmentID = vanus.EmptyID()
			return err
		}
	}
	el.segments = append(el.segments, seg.ID)
	el.segmentList.Set(seg.ID.Uint64(), seg)
	return nil
}

func (el *eventlog) markSegmentFull(ctx context.Context, seg *Segment) error {
	// because sync.RWMutex isn't reentrant, so here have to implement *eventlog.nextOf again
	node := el.segmentList.Get(seg.ID.Uint64())
	nextNode := node.Next()
	if nextNode == nil {
		return nil
	}
	next, _ := nextNode.Value.(*Segment)
	next.StartOffsetInLog = seg.StartOffsetInLog + int64(seg.Number)
	data, _ := json.Marshal(next)
	// TODO(wenfeng.wang) update block info at the same time
	log.Debug(context.TODO(), "segment is full", map[string]interface{}{
		"data": string(data),
	})
	if err := el.kvClient.Set(ctx, metadata.GetSegmentMetadataKey(next.ID), data); err != nil {
		return err
	}
	return nil
}

func (el *eventlog) head() *Segment {
	el.mutex.RLock()
	defer el.mutex.RUnlock()

	if el.size() == 0 {
		return nil
	}
	ptr := el.segmentList.Front()
	return ptr.Value.(*Segment)
}

func (el *eventlog) tail() *Segment {
	if el.size() == 0 {
		return nil
	}
	ptr := el.segmentList.Back()
	return ptr.Value.(*Segment)
}

func (el *eventlog) indexAt(idx int) *Segment {
	el.mutex.RLock()
	defer el.mutex.RUnlock()

	if el.size() < idx {
		return nil
	}
	ptr := el.segmentList.Front()
	for i := 0; i < idx; i++ {
		if ptr == nil {
			return nil
		}
		ptr = ptr.Next()
	}
	if ptr == nil {
		return nil
	}
	return ptr.Value.(*Segment)
}

func (el *eventlog) size() int {
	return el.segmentList.Len()
}

func (el *eventlog) getAllSegments() []*Segment {
	el.mutex.RLock()
	defer el.mutex.RUnlock()

	segs := make([]*Segment, 0)

	if el.size() == 0 {
		return segs
	}
	ptr := el.segmentList.Front()
	for ptr != nil {
		next, _ := ptr.Value.(*Segment)
		segs = append(segs, next)
		ptr = ptr.Next()
	}
	return segs
}

func (el *eventlog) listOfRight(seg *Segment, includeSelf bool) []Segment {
	el.mutex.RLock()
	defer el.mutex.RUnlock()
	segs := make([]Segment, 0)
	if seg == nil {
		return segs
	}
	if includeSelf {
		segs = append(segs, *seg)
	}
	node := el.segmentList.Get(seg.ID.Uint64())
	if node == nil {
		return segs
	}
	v := node.Next()
	for v != nil {
		next, _ := v.Value.(*Segment)
		segs = append(segs, *next)
		v = v.Next()
	}

	return segs
}

func (el *eventlog) listOfPrevious(seg *Segment) []*Segment { //nolint:unused // ok
	el.mutex.RLock()
	defer el.mutex.RUnlock()
	segs := make([]*Segment, 0)
	node := el.segmentList.Get(seg.ID.Uint64())
	if node == nil {
		return segs
	}
	v := node.Prev()
	for v != nil {
		next, _ := v.Value.(*Segment)
		segs = append(segs, next)
		v = v.Prev()
	}

	// reverse slice
	for i, j := 0, len(segs)-1; i < j; i, j = i+1, j-1 {
		segs[i], segs[j] = segs[j], segs[i]
	}

	return segs
}

func (el *eventlog) deleteHead(ctx context.Context) error {
	el.mutex.Lock()
	defer el.mutex.Unlock()
	if el.segmentList.Len() == 0 {
		return nil
	}
	headV := el.segmentList.Front()
	nextV := headV.Next()
	head, _ := headV.Value.(*Segment)
	segments := make([]vanus.ID, 0, len(el.segments)-1)
	for _, v := range el.segments {
		if v.Uint64() != head.ID.Uint64() {
			segments = append(segments, v)
		}
	}
	if err := el.kvClient.Delete(ctx, metadata.GetEventlogSegmentsMetadataKey(el.md.ID, head.ID)); err != nil {
		log.Warning(ctx, "delete segment failed when delete head", map[string]interface{}{
			log.KeyError:  err,
			"segment_id":  head.ID.String(),
			"eventlog_id": el.md.ID.String(),
		})
		return err
	}

	if nextV != nil {
		next, _ := nextV.Value.(*Segment)
		next.PreviousSegmentID = vanus.EmptyID()
		data, _ := json.Marshal(next)
		if err := el.kvClient.Set(ctx, metadata.GetSegmentMetadataKey(next.ID), data); err != nil {
			log.Warning(ctx, "update segment failed when delete head", map[string]interface{}{
				log.KeyError:  err,
				"segment_id":  next.ID.String(),
				"eventlog_id": el.md.ID.String(),
			})
			return err
		}
	}
	if el.writePtr == head {
		el.writePtr = nil
	}
	_ = el.segmentList.RemoveFront()
	el.segments = segments
	return nil
}

func (el *eventlog) updateSegment(ctx context.Context, seg *Segment) error {
	// TODO(wenfeng.wang) use TXN to make sure that update block info at the same time
	data, _ := json.Marshal(seg)
	key := filepath.Join(metadata.SegmentKeyPrefixInKVStore, seg.ID.String())
	if err := el.kvClient.Set(ctx, key, data); err != nil {
		return err
	}
	if seg.isFull() {
		err := el.markSegmentFull(ctx, seg)
		if err != nil {
			return err
		}
	}
	return nil
}

func (el *eventlog) lock() {
	el.mutex.Lock()
}

func (el *eventlog) unlock() {
	el.mutex.Unlock()
}

func (el *eventlog) rLock() {
	el.mutex.RLock()
}

func (el *eventlog) rUnlock() {
	el.mutex.RUnlock()
}
