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

//go:generate mockgen -source=worker.go  -destination=mock_worker.go -package=worker
package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/linkall-labs/vanus/internal/controller/errors"
	"github.com/linkall-labs/vanus/internal/controller/trigger/metadata"
	"github.com/linkall-labs/vanus/internal/controller/trigger/subscription"
	"github.com/linkall-labs/vanus/internal/convert"
	"github.com/linkall-labs/vanus/internal/primitive"
	"github.com/linkall-labs/vanus/internal/primitive/queue"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/observability/log"
	"github.com/linkall-labs/vanus/proto/pkg/trigger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	errNoInit = fmt.Errorf("trigger worker not init")
)

type TriggerWorker interface {
	Init(ctx context.Context) error
	RemoteStart(ctx context.Context) error
	RemoteStop(ctx context.Context) error
	Close() error
	IsActive() bool
	Reset()
	GetInfo() metadata.TriggerWorkerInfo
	GetAddr() string
	SetPhase(metadata.TriggerWorkerPhase)
	GetPhase() metadata.TriggerWorkerPhase
	GetPendingTime() time.Time
	GetHeartbeatTime() time.Time
	Polish()
	AssignSubscription(id vanus.ID)
	UnAssignSubscription(id vanus.ID)
	GetAssignSubscriptions() []vanus.ID
	ResetOffsetToTimestamp(id vanus.ID, timestamp uint64) error
}

// triggerWorker send subscription to trigger worker server.
type triggerWorker struct {
	info                  *metadata.TriggerWorkerInfo
	cc                    *grpc.ClientConn
	client                trigger.TriggerWorkerClient
	lock                  sync.RWMutex
	assignSubscriptionIDs sync.Map
	pendingTime           time.Time
	heartbeatTime         time.Time
	ctx                   context.Context
	stop                  context.CancelFunc
	subscriptionManager   subscription.Manager
	subscriptionQueue     queue.Queue
}

func NewTriggerWorkerByAddr(addr string, subscriptionManager subscription.Manager) TriggerWorker {
	tw := NewTriggerWorker(metadata.NewTriggerWorkerInfo(addr), subscriptionManager)
	return tw
}

func NewTriggerWorker(twInfo *metadata.TriggerWorkerInfo, subscriptionManager subscription.Manager) TriggerWorker {
	tw := &triggerWorker{
		info:                twInfo,
		subscriptionManager: subscriptionManager,
		subscriptionQueue:   queue.New(),
		pendingTime:         time.Now(),
	}
	tw.ctx, tw.stop = context.WithCancel(context.Background())
	tw.start()
	return tw
}

func (tw *triggerWorker) start() {
	go func() {
		ctx := tw.ctx
		for {
			subscriptionID, stop := tw.subscriptionQueue.Get()
			if stop {
				break
			}
			log.Info(ctx, "trigger worker begin hand subscription", map[string]interface{}{
				log.KeyTriggerWorkerAddr: tw.info.Addr,
				log.KeySubscriptionID:    subscriptionID,
			})
			err := tw.handler(ctx, subscriptionID)
			if err == nil {
				tw.subscriptionQueue.Done(subscriptionID)
				tw.subscriptionQueue.ClearFailNum(subscriptionID)
				log.Warning(ctx, "trigger worker handle subscription sucess", map[string]interface{}{
					log.KeyTriggerWorkerAddr: tw.info.Addr,
					log.KeySubscriptionID:    subscriptionID,
				})
			} else {
				tw.subscriptionQueue.ReAdd(subscriptionID)
				log.Warning(ctx, "trigger worker handle subscription has error", map[string]interface{}{
					log.KeyError:             err,
					log.KeyTriggerWorkerAddr: tw.info.Addr,
					log.KeySubscriptionID:    subscriptionID,
				})
			}
		}
	}()
}
func (tw *triggerWorker) handler(ctx context.Context, subscriptionID vanus.ID) error {
	_, exist := tw.assignSubscriptionIDs.Load(subscriptionID)
	if !exist {
		// no assign to this trigger worker,remove subscription
		return tw.removeSubscription(ctx, subscriptionID)
	}
	subData := tw.subscriptionManager.GetSubscription(ctx, subscriptionID)
	if subData == nil {
		return nil
	}
	offsets, err := tw.subscriptionManager.GetOffset(ctx, subscriptionID)
	if err != nil {
		return err
	}
	err = tw.addSubscription(ctx, &primitive.Subscription{
		ID:               subData.ID,
		Filters:          subData.Filters,
		Sink:             subData.Sink,
		EventBus:         subData.EventBus,
		Offsets:          offsets,
		InputTransformer: subData.InputTransformer,
		Config:           subData.Config,
	})
	if err != nil {
		return err
	}
	// modify subscription to running
	subData.Phase = metadata.SubscriptionPhaseRunning
	err = tw.subscriptionManager.UpdateSubscription(ctx, subData)
	if err != nil {
		return err
	}
	return nil
}

func (tw *triggerWorker) IsActive() bool {
	tw.lock.RLock()
	defer tw.lock.RUnlock()
	if tw.info.Phase != metadata.TriggerWorkerPhaseRunning {
		return false
	}
	if tw.heartbeatTime.IsZero() {
		return false
	}
	return true
}

// Reset when trigger worker restart and re-connect.
func (tw *triggerWorker) Reset() {
	tw.lock.Lock()
	defer tw.lock.Unlock()
	tw.info.Phase = metadata.TriggerWorkerPhasePending
	tw.pendingTime = time.Now()
}

func (tw *triggerWorker) GetInfo() metadata.TriggerWorkerInfo {
	return *tw.info
}

func (tw *triggerWorker) GetAddr() string {
	return tw.info.Addr
}

func (tw *triggerWorker) SetPhase(phase metadata.TriggerWorkerPhase) {
	tw.lock.Lock()
	defer tw.lock.Unlock()
	tw.info.Phase = phase
}

func (tw *triggerWorker) GetPhase() metadata.TriggerWorkerPhase {
	tw.lock.RLock()
	defer tw.lock.RUnlock()
	return tw.info.Phase
}

func (tw *triggerWorker) Polish() {
	tw.lock.Lock()
	defer tw.lock.Unlock()
	tw.heartbeatTime = time.Now()
}

func (tw *triggerWorker) AssignSubscription(id vanus.ID) {
	_, exist := tw.assignSubscriptionIDs.Load(id)
	var msg string
	if !exist {
		msg = "trigger worker assign a subscription"
	} else {
		msg = "trigger worker reassign a subscription"
	}
	log.Info(context.Background(), msg, map[string]interface{}{
		log.KeyTriggerWorkerAddr: tw.info.Addr,
		log.KeySubscriptionID:    id,
	})
	tw.assignSubscriptionIDs.Store(id, time.Now())
	tw.subscriptionQueue.Add(id)
}

func (tw *triggerWorker) UnAssignSubscription(id vanus.ID) {
	log.Info(context.Background(), "trigger worker remove a subscription", map[string]interface{}{
		log.KeyTriggerWorkerAddr: tw.info.Addr,
		log.KeySubscriptionID:    id,
	})
	tw.assignSubscriptionIDs.Delete(id)
	if tw.info.Phase == metadata.TriggerWorkerPhaseRunning {
		err := tw.removeSubscription(tw.ctx, id)
		if err != nil {
			log.Warning(context.Background(), "trigger worker remove subscription error", map[string]interface{}{
				log.KeyError:             err,
				log.KeyTriggerWorkerAddr: tw.info.Addr,
				log.KeySubscriptionID:    id,
			})
			tw.subscriptionQueue.Add(id)
		}
	}
}

func (tw *triggerWorker) GetAssignSubscriptions() []vanus.ID {
	ids := make([]vanus.ID, 0)
	tw.assignSubscriptionIDs.Range(func(key, value interface{}) bool {
		id, _ := key.(vanus.ID)
		ids = append(ids, id)
		return true
	})
	return ids
}

func (tw *triggerWorker) GetPendingTime() time.Time {
	tw.lock.RLock()
	defer tw.lock.RUnlock()
	return tw.pendingTime
}

func (tw *triggerWorker) GetHeartbeatTime() time.Time {
	tw.lock.RLock()
	defer tw.lock.RUnlock()
	return tw.heartbeatTime
}

func (tw *triggerWorker) Init(ctx context.Context) error {
	if tw.cc != nil {
		return nil
	}
	var err error
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	tw.cc, err = grpc.DialContext(ctx, tw.info.Addr, opts...)
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("grpc dial error").Wrap(err)
	}
	tw.client = trigger.NewTriggerWorkerClient(tw.cc)
	return nil
}

func (tw *triggerWorker) Close() error {
	if tw.cc != nil {
		tw.lock.Lock()
		defer tw.lock.Unlock()
		return tw.cc.Close()
	}
	tw.stop()
	tw.subscriptionQueue.ShutDown()
	return nil
}

func (tw *triggerWorker) RemoteStop(ctx context.Context) error {
	if tw.client == nil {
		return errNoInit
	}
	_, err := tw.client.Stop(ctx, &trigger.StopTriggerWorkerRequest{})
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("stop error").Wrap(err)
	}
	return nil
}

func (tw *triggerWorker) RemoteStart(ctx context.Context) error {
	_, err := tw.client.Start(ctx, &trigger.StartTriggerWorkerRequest{})
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("start error").Wrap(err)
	}
	return nil
}

func (tw *triggerWorker) ResetOffsetToTimestamp(id vanus.ID, timestamp uint64) error {
	if tw.client == nil {
		return errNoInit
	}
	request := &trigger.ResetOffsetToTimestampRequest{SubscriptionId: id.Uint64(), Timestamp: timestamp}
	_, err := tw.client.ResetOffsetToTimestamp(tw.ctx, request)
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("reset offset to timestamp").Wrap(err)
	}
	return nil
}

func (tw *triggerWorker) addSubscription(ctx context.Context, sub *primitive.Subscription) error {
	if tw.client == nil {
		return errNoInit
	}
	request := convert.ToPbAddSubscription(sub)
	_, err := tw.client.AddSubscription(ctx, request)
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("add subscription error").Wrap(err)
	}
	return nil
}

func (tw *triggerWorker) removeSubscription(ctx context.Context, id vanus.ID) error {
	if tw.client == nil {
		return errNoInit
	}
	request := &trigger.RemoveSubscriptionRequest{SubscriptionId: uint64(id)}
	_, err := tw.client.RemoveSubscription(ctx, request)
	if err != nil {
		return errors.ErrTriggerWorker.WithMessage("remove subscription error").Wrap(err)
	}
	return nil
}
