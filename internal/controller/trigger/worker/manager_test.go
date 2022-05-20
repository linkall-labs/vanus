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

	"github.com/linkall-labs/vanus/internal/controller/trigger/info"
	"github.com/linkall-labs/vanus/internal/controller/trigger/storage"
	"github.com/linkall-labs/vanus/internal/controller/trigger/subscription"
	"github.com/linkall-labs/vanus/internal/primitive"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/internal/util"

	"github.com/golang/mock/gomock"
	. "github.com/smartystreets/goconvey/convey"
)

func getTestSubscription() *primitive.SubscriptionData {
	return &primitive.SubscriptionData{
		ID:    1,
		Phase: primitive.SubscriptionPhaseCreated,
	}
}

func getTestTriggerWorkerRemoveSubscription() OnTriggerWorkerRemoveSubscription {
	return func(ctx context.Context, subId vanus.ID, addr string) error {
		fmt.Println(fmt.Sprintf("trigger worker leave remove subscription %s", subId))
		return nil
	}
}

func TestInit(t *testing.T) {
	ctx := context.Background()
	addr := "test"
	storage := storage.NewFakeStorage()
	storage.SaveTriggerWorker(ctx, info.TriggerWorkerInfo{
		ID:   util.GetIdByAddr(addr),
		Addr: addr,
	})
	ctrl := gomock.NewController(t)
	subManager := subscription.NewMockManager(ctrl)
	sub := getTestSubscription()
	sub.TriggerWorker = addr
	subManager.EXPECT().ListSubscription(ctx).Return(map[vanus.ID]*primitive.SubscriptionData{
		sub.ID: sub,
	})
	twManager := NewTriggerWorkerManager(Config{}, storage, subManager, nil).(*manager)
	Convey("test init", t, func() {
		twManager.Init(ctx)
		triggerWorkers := twManager.triggerWorkers
		So(len(triggerWorkers), ShouldEqual, 1)
		tWorker, exist := triggerWorkers[addr]
		So(exist, ShouldBeTrue)
		So(tWorker, ShouldNotBeNil)
		subIds := tWorker.GetAssignSubIds()
		_, exist = subIds[sub.ID]
		So(exist, ShouldBeTrue)
	})
}

func TestAddTriggerWorker(t *testing.T) {
	ctx := context.Background()
	addr := "test"
	storage := storage.NewFakeStorage()
	twManager := NewTriggerWorkerManager(Config{}, storage, nil, nil).(*manager)

	Convey("test add", t, func() {
		twManager.AddTriggerWorker(ctx, addr)
		triggerWorkers := twManager.triggerWorkers
		So(len(triggerWorkers), ShouldEqual, 1)
		tWorker, exist := triggerWorkers[addr]
		So(exist, ShouldBeTrue)
		So(tWorker, ShouldNotBeNil)
		Convey("test repeat add", func() {
			twManager.AddTriggerWorker(ctx, addr)
			triggerWorkers = twManager.triggerWorkers
			So(len(triggerWorkers), ShouldEqual, 1)
			tWorker, exist = triggerWorkers[addr]
			So(exist, ShouldBeTrue)
			So(tWorker, ShouldNotBeNil)
		})
	})
}

func TestRemoveTriggerWorker(t *testing.T) {
	ctx := context.Background()
	addr := "test"
	storage := storage.NewFakeStorage()
	sub := getTestSubscription()
	twManager := NewTriggerWorkerManager(Config{}, storage, nil, getTestTriggerWorkerRemoveSubscription()).(*manager)
	Convey("test remove not exist", t, func() {
		twManager.RemoveTriggerWorker(ctx, addr)
		triggerWorkers := twManager.triggerWorkers
		So(len(triggerWorkers), ShouldEqual, 0)
		Convey("test remove", func() {
			twManager.AddTriggerWorker(ctx, addr)
			triggerWorkers = twManager.triggerWorkers
			tWorker := triggerWorkers[addr]
			tWorker.AddAssignSub(sub.ID)
			twManager.RemoveTriggerWorker(ctx, addr)
			triggerWorkers = twManager.triggerWorkers
			So(len(triggerWorkers), ShouldEqual, 0)
			So(len(tWorker.GetAssignSubIds()), ShouldEqual, 0)
		})
	})
}

func TestAssignSubscription(t *testing.T) {
	ctx := context.Background()
	addr := "test"
	storage := storage.NewFakeStorage()
	sub := getTestSubscription()
	ctrl := gomock.NewController(t)
	subManager := subscription.NewMockManager(ctrl)
	sub.TriggerWorker = addr
	twManager := NewTriggerWorkerManager(Config{}, storage, subManager, getTestTriggerWorkerRemoveSubscription()).(*manager)
	Convey("assign subscription", t, func() {
		twManager.AddTriggerWorker(ctx, addr)
		twManager.UpdateTriggerWorkerInfo(ctx, addr, map[vanus.ID]struct{}{sub.ID: {}})
		tWorker := twManager.GetTriggerWorker(ctx, addr)
		subManager.EXPECT().GetSubscription(ctx, sub.ID).Return(&primitive.Subscription{
			ID: sub.ID,
		}, nil)
		twManager.AssignSubscription(ctx, tWorker, sub.ID)
		So(len(tWorker.GetAssignSubIds()), ShouldEqual, 1)
		_, exist := tWorker.GetAssignSubIds()[sub.ID]
		So(exist, ShouldBeTrue)
	})
}
