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

package info

import (
	"fmt"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/internal/util"
	"time"
)

type TriggerWorkerPhase string

const (
	TriggerWorkerPhasePending    = "pending"
	TriggerWorkerPhaseRunning    = "running"
	TriggerWorkerPhasePaused     = "paused"
	TriggerWorkerPhaseDisconnect = "disconnect"
)

type TriggerWorkerInfo struct {
	ID            string                 `json:"-"`
	Addr          string                 `json:"addr"`
	Phase         TriggerWorkerPhase     `json:"phase"`
	AssignSubIds  map[vanus.ID]time.Time `json:"-"`
	ReportSubIds  map[vanus.ID]struct{}  `json:"-"`
	PendingTime   time.Time              `json:"-"`
	HeartbeatTime *time.Time             `json:"-"`
}

func NewTriggerWorkerInfo(addr string) *TriggerWorkerInfo {
	twInfo := &TriggerWorkerInfo{
		Addr:  addr,
		ID:    util.GetIdByAddr(addr),
		Phase: TriggerWorkerPhasePending,
	}
	twInfo.Init()
	return twInfo
}

func (tw *TriggerWorkerInfo) Init() {
	tw.PendingTime = time.Now()
	tw.ReportSubIds = map[vanus.ID]struct{}{}
	tw.AssignSubIds = map[vanus.ID]time.Time{}
}

func (tw *TriggerWorkerInfo) String() string {
	return fmt.Sprintf("addr:%s,phase:%v,subIds:%v", tw.Addr, tw.Phase, tw.AssignSubIds)
}
