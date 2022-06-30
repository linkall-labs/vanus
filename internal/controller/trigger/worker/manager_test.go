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

package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/linkall-labs/vanus/internal/controller/trigger/info"
	"github.com/linkall-labs/vanus/internal/controller/trigger/storage"
	"github.com/linkall-labs/vanus/internal/controller/trigger/subscription"
	"github.com/linkall-labs/vanus/internal/primitive"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/observability/log"
	. "github.com/smartystreets/goconvey/convey"
)

func getTestSubscription() *primitive.SubscriptionData {
	return &primitive.SubscriptionData{
		ID:    vanus.NewID(),
		Phase: primitive.SubscriptionPhaseCreated,
	}
}

func getTestTriggerWorkerRemoveSubscription() OnTriggerWorkerRemoveSubscription {
	return func(ctx context.Context, subscriptionID vanus.ID, addr string) error {
		log.Info(ctx, "trigger worker leave remove subscription", map[string]interface{}{
			log.KeySubscriptionID: subscriptionID,
		})
		return nil
	}
}

func getTestTriggerWorkerRemoveSubscriptionWithErr() OnTriggerWorkerRemoveSubscription {
	return func(ctx context.Context, subId vanus.ID, addr string) error {
		return fmt.Errorf("trigger worker leave remove subscription %s fail", subId)
	}
}

func TestInit(t *testing.T) {
	Convey("test init", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		sub := getTestSubscription()
		sub.TriggerWorker = addr
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager, nil)
		subManager.EXPECT().GetSubscription(gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf("error"))
		workerStorage.EXPECT().ListTriggerWorker(ctx).Return([]*info.TriggerWorkerInfo{
			{Addr: addr},
		}, nil)
		subManager.EXPECT().ListSubscription(ctx).Return([]*primitive.SubscriptionData{
			sub,
		})
		err := twManager.Init(ctx)
		So(err, ShouldBeNil)
		tWorker := twManager.GetTriggerWorker(addr)
		So(tWorker, ShouldNotBeNil)
		subIds := tWorker.GetAssignSubscriptions()
		So(len(subIds), ShouldEqual, 1)
		So(subIds[0], ShouldEqual, sub.ID)
		time.Sleep(time.Millisecond * 10)
		twManager.Stop()
	})

}

func TestAddTriggerWorker(t *testing.T) {
	Convey("test add trigger worker", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		subManager := subscription.NewMockManager(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager, nil)
		workerStorage.EXPECT().SaveTriggerWorker(ctx, gomock.Any()).AnyTimes().Return(nil)
		err := twManager.AddTriggerWorker(ctx, addr)
		So(err, ShouldBeNil)
		tWorker := twManager.GetTriggerWorker(addr)
		So(tWorker, ShouldNotBeNil)
		Convey("test repeat add trigger worker", func() {
			err = twManager.AddTriggerWorker(ctx, addr)
			So(err, ShouldBeNil)
			tWorker = twManager.GetTriggerWorker(addr)
			So(tWorker, ShouldNotBeNil)
		})
	})
}

func TestRemoveTriggerWorker(t *testing.T) {
	Convey("remove worker", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		sub := getTestSubscription()
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, nil, getTestTriggerWorkerRemoveSubscription()).(*manager)
		Convey("test remove not exist", func() {
			twManager.RemoveTriggerWorker(ctx, addr)
			So(twManager.GetTriggerWorker(addr), ShouldBeNil)
		})
		tWorker.EXPECT().GetAddr().AnyTimes().Return(addr)
		tWorker.EXPECT().GetPhase().AnyTimes().Return(info.TriggerWorkerPhaseRunning)
		tWorker.EXPECT().GetInfo().AnyTimes().Return(info.TriggerWorkerInfo{})
		workerStorage.EXPECT().SaveTriggerWorker(ctx, gomock.Any()).AnyTimes().Return(nil)
		workerStorage.EXPECT().DeleteTriggerWorker(ctx, gomock.Any()).AnyTimes().Return(nil)
		tWorker.EXPECT().SetPhase(info.TriggerWorkerPhasePaused).AnyTimes().Return()
		tWorker.EXPECT().GetAssignSubscriptions().AnyTimes().Return([]vanus.ID{sub.ID})
		Convey("test remove subscription no error", func() {
			twManager.triggerWorkers[addr] = tWorker
			So(twManager.GetTriggerWorker(addr), ShouldNotBeNil)
			twManager.RemoveTriggerWorker(ctx, addr)
			So(twManager.GetTriggerWorker(addr), ShouldBeNil)
		})

		Convey("test remove subscription with error", func() {
			twManager.onRemoveSubscription = getTestTriggerWorkerRemoveSubscriptionWithErr()
			twManager.triggerWorkers[addr] = tWorker
			So(twManager.GetTriggerWorker(addr), ShouldNotBeNil)
			twManager.RemoveTriggerWorker(ctx, addr)
			So(twManager.GetTriggerWorker(addr), ShouldNotBeNil)
		})
	})
}

func TestManager_UpdateTriggerWorkerInfo(t *testing.T) {
	Convey("test update trigger worker info", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		sub := getTestSubscription()
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, nil, getTestTriggerWorkerRemoveSubscription()).(*manager)
		Convey("trigger worker not exist", func() {
			err := twManager.UpdateTriggerWorkerInfo(ctx, addr, []vanus.ID{sub.ID})
			So(err, ShouldNotBeNil)
		})
		tWorker.EXPECT().ReportSubscription(gomock.Any()).AnyTimes().Return()
		tWorker.EXPECT().GetInfo().AnyTimes().Return(info.TriggerWorkerInfo{})
		Convey("trigger worker running", func() {
			twManager.triggerWorkers[addr] = tWorker
			tWorker.EXPECT().GetPhase().AnyTimes().Return(info.TriggerWorkerPhaseRunning)
			err := twManager.UpdateTriggerWorkerInfo(ctx, addr, []vanus.ID{sub.ID})
			So(err, ShouldBeNil)
		})

		Convey("trigger worker not running", func() {
			twManager.triggerWorkers[addr] = tWorker
			tWorker.EXPECT().GetPhase().AnyTimes().Return(info.TriggerWorkerPhasePending)
			tWorker.EXPECT().SetPhase(info.TriggerWorkerPhaseRunning).AnyTimes().Return()
			tWorker.EXPECT().GetAddr().Return(addr)
			workerStorage.EXPECT().SaveTriggerWorker(gomock.Any(), gomock.Any()).Return(nil)
			err := twManager.UpdateTriggerWorkerInfo(ctx, addr, []vanus.ID{sub.ID})
			So(err, ShouldBeNil)
			workerStorage.EXPECT().SaveTriggerWorker(gomock.Any(), gomock.Any()).Return(fmt.Errorf("error"))
			err = twManager.UpdateTriggerWorkerInfo(ctx, addr, []vanus.ID{sub.ID})
			So(err, ShouldBeNil)
		})

	})
}

func TestPendingTriggerWorkerHandler(t *testing.T) {
	Convey("pending worker handler", t, func() {
		ctx := context.Background()
		addr := "test"
		sub := getTestSubscription()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager,
			getTestTriggerWorkerRemoveSubscription()).(*manager)
		twManager.triggerWorkers[addr] = tWorker
		tWorker.EXPECT().GetAddr().AnyTimes().Return(addr)
		tWorker.EXPECT().GetInfo().AnyTimes().Return(info.TriggerWorkerInfo{})
		Convey("pending worker start", func() {
			tWorker.EXPECT().GetPendingTime().AnyTimes().Return(time.Now().Add(twManager.config.StartWorkerDuration * -1))
			time.Sleep(time.Millisecond)
			tWorker.EXPECT().RemoteStart(ctx).Return(nil)
			twManager.pendingTriggerWorkerHandler(ctx, tWorker)
			tWorker.EXPECT().RemoteStart(ctx).Return(fmt.Errorf("start trigget worker error"))
			twManager.pendingTriggerWorkerHandler(ctx, tWorker)
		})
		Convey("pending worker clean", func() {
			tWorker.EXPECT().GetPendingTime().Return(time.Now().Add(twManager.config.WaitRunningTimeout * -1))
			tWorker.EXPECT().SetPhase(info.TriggerWorkerPhasePaused).Return()
			tWorker.EXPECT().GetAssignSubscriptions().Return([]vanus.ID{sub.ID})
			workerStorage.EXPECT().DeleteTriggerWorker(ctx, gomock.Any()).Return(nil)
			time.Sleep(time.Millisecond)
			twManager.pendingTriggerWorkerHandler(ctx, tWorker)
			So(twManager.GetTriggerWorker(addr), ShouldBeNil)
		})
	})
}

func TestRunningTriggerWorkerHandler(t *testing.T) {
	Convey("running worker handler", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager,
			getTestTriggerWorkerRemoveSubscription()).(*manager)
		tWorker.EXPECT().GetAddr().AnyTimes().Return(addr)
		tWorker.EXPECT().GetInfo().AnyTimes().Return(info.TriggerWorkerInfo{})
		Convey("running worker heartbeat timeout", func() {
			tWorker.EXPECT().IsActive().Return(true)
			hbTime := time.Now().Add(twManager.config.HeartbeatTimeout * -1)
			tWorker.EXPECT().GetHeartbeatTime().Return(hbTime)
			tWorker.EXPECT().SetPhase(info.TriggerWorkerPhaseDisconnect).Return()
			workerStorage.EXPECT().SaveTriggerWorker(ctx, gomock.Any()).Return(nil)
			time.Sleep(time.Millisecond)
			twManager.runningTriggerWorkerHandler(ctx, tWorker)
		})

		Convey("running worker lost heartbeat ", func() {
			tWorker.EXPECT().IsActive().Return(false)
			tWorker.EXPECT().GetPendingTime().Return(time.Now().Add(twManager.config.LostHeartbeatTime * -1))
			twManager.runningTriggerWorkerHandler(ctx, tWorker)
		})
	})
}

func TestManagerCheck(t *testing.T) {
	Convey("check", t, func() {
		ctx := context.Background()
		addr := "test"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager,
			getTestTriggerWorkerRemoveSubscription()).(*manager)
		twManager.triggerWorkers[addr] = tWorker
		tWorker.EXPECT().GetAddr().AnyTimes().Return(addr)
		tWorker.EXPECT().GetInfo().AnyTimes().Return(info.TriggerWorkerInfo{})
		Convey("pending check", func() {
			tWorker.EXPECT().GetPhase().Return(info.TriggerWorkerPhasePending)
			tWorker.EXPECT().GetPendingTime().Return(time.Now())
			twManager.check(ctx)
		})
		Convey("running check", func() {
			tWorker.EXPECT().GetPhase().Return(info.TriggerWorkerPhaseRunning)
			tWorker.EXPECT().IsActive().Return(true)
			tWorker.EXPECT().GetHeartbeatTime().Return(time.Now())
			twManager.check(ctx)
		})
		Convey("disconnect check", func() {
			tWorker.EXPECT().GetPhase().Return(info.TriggerWorkerPhaseDisconnect)
			tWorker.EXPECT().IsActive().Return(true)
			hbTime := time.Now().Add(twManager.config.DisconnectCleanTime * -1)
			tWorker.EXPECT().GetHeartbeatTime().Return(hbTime)
			tWorker.EXPECT().GetAssignSubscriptions().Return(nil)
			time.Sleep(time.Millisecond)
			workerStorage.EXPECT().DeleteTriggerWorker(ctx, gomock.Any()).Return(nil)
			twManager.check(ctx)
		})
		Convey("pause check", func() {
			tWorker.EXPECT().GetPhase().Return(info.TriggerWorkerPhasePaused)
			tWorker.EXPECT().GetAssignSubscriptions().Return(nil)
			workerStorage.EXPECT().DeleteTriggerWorker(ctx, gomock.Any()).Return(nil)
			twManager.check(ctx)
		})
	})
}

func TestGetActiveWorker(t *testing.T) {
	Convey("get active worker", t, func() {
		addr := "test"
		addr2 := "test2"
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		tWorker := NewMockTriggerWorker(ctrl)
		tWorker2 := NewMockTriggerWorker(ctrl)
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager,
			getTestTriggerWorkerRemoveSubscription()).(*manager)
		tWorker.EXPECT().GetPhase().Return(info.TriggerWorkerPhaseRunning)
		tWorker.EXPECT().IsActive().Return(true)
		tWorker.EXPECT().GetHeartbeatTime().Return(time.Now())
		tWorker.EXPECT().GetInfo().Return(info.TriggerWorkerInfo{Addr: addr})
		tWorker2.EXPECT().GetPhase().AnyTimes().Return(info.TriggerWorkerPhaseRunning)
		tWorker2.EXPECT().IsActive().Return(false)
		twManager.triggerWorkers[addr] = tWorker
		twManager.triggerWorkers[addr2] = tWorker2
		Convey("active worker", func() {
			tWorker1 := twManager.GetTriggerWorker(addr)
			So(tWorker1, ShouldNotBeNil)
			tWorker2 := twManager.GetTriggerWorker(addr2)
			So(tWorker2, ShouldNotBeNil)
			tWorkers := twManager.GetActiveRunningTriggerWorker()
			So(len(tWorkers), ShouldEqual, 1)
			So(tWorkers[0].Addr, ShouldEqual, addr)
		})
	})
}

func TestManagerStartStop(t *testing.T) {
	Convey("manager start stop", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		subManager := subscription.NewMockManager(ctrl)
		workerStorage := storage.NewMockTriggerWorkerStorage(ctrl)
		twManager := NewTriggerWorkerManager(Config{}, workerStorage, subManager,
			getTestTriggerWorkerRemoveSubscription())
		twManager.Start()
		time.Sleep(time.Millisecond)
		twManager.Stop()
	})
}
