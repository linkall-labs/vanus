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

package primitive

import (
	"github.com/linkall-labs/vanus/internal/primitive/info"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"time"
)

type URI string

type SubscriptionPhase string

const (
	SubscriptionPhaseCreated   = "created"
	SubscriptionPhasePending   = "pending"
	SubscriptionPhaseScheduled = "scheduled"
	SubscriptionPhaseRunning   = "running"
	SubscriptionPhaseToDelete  = "toDelete"
)

type SubscriptionData struct {
	ID               vanus.ID              `json:"id"`
	Source           string                `json:"source,omitempty"`
	Types            []string              `json:"types,omitempty"`
	Config           map[string]string     `json:"config,omitempty"`
	Filters          []*SubscriptionFilter `json:"filters,omitempty"`
	Sink             URI                   `json:"sink,omitempty"`
	Protocol         string                `json:"protocol,omitempty"`
	ProtocolSettings map[string]string     `json:"protocolSettings,omitempty"`
	EventBus         string                `json:"eventBus"`
	Phase            SubscriptionPhase     `json:"phase"`
	TriggerWorker    string                `json:"triggerWorker,omitempty"`
	InputTransformer *InputTransformer     `json:"inputTransformer,omitempty"`
	HeartbeatTime    time.Time             `json:"-"`
}

type Subscription struct {
	ID               vanus.ID              `json:"id"`
	Filters          []*SubscriptionFilter `json:"filters,omitempty"`
	Sink             URI                   `json:"sink,omitempty"`
	EventBus         string                `json:"eventBus"`
	Offsets          info.ListOffsetInfo   `json:"offsets"`
	InputTransformer *InputTransformer     `json:"inputTransformer,omitempty"`
}

type SubscriptionFilter struct {
	Exact  map[string]string     `json:"exact,omitempty"`
	Prefix map[string]string     `json:"prefix,omitempty"`
	Suffix map[string]string     `json:"suffix,omitempty"`
	CeSQL  string                `json:"ceSql,omitempty"`
	Not    *SubscriptionFilter   `json:"not,omitempty"`
	All    []*SubscriptionFilter `json:"all,omitempty"`
	Any    []*SubscriptionFilter `json:"any,omitempty"`
	CEL    string                `json:"CEL,omitempty"`
}

type InputTransformer struct {
	Define   map[string]string `json:"define,omitempty"`
	Template string            `json:"template,omitempty"`
}

type SinkSpec struct {
	Type   string
	Name   string // TODO use id or name? ID used in CloudEvents Specification
	Weight float32
	Config map[string]interface{}
}

type DialectType string

const (
	ExactDialect  = "exact"
	PrefixDialect = "prefix"
	SuffixDialect = "suffix"
	AllDialect    = "all"
	AnyDialect    = "any"
	NotDialect    = "not"
	SQLDialect    = "ce-sql"
)

type FilterSpec struct {
	Name    string
	Exp     interface{}
	ApplyTo []string
	Target  TargetSpec
}

type SubscriptionSpec struct {
	EventBuses []string
	Sinks      []SinkSpec
	Filters    []FilterSpec
}

type TargetSpec struct {
	LBStrategy string
	Sinks      []string
}
