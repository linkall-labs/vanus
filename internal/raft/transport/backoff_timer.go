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

package transport

import (
	"context"
	"time"

	"github.com/linkall-labs/vanus/observability/log"
)

type BackoffTimer struct {
	startInterval  int64
	curInterval    int64
	maxInterval    int64
	expire         int64
	useCanTryFirst bool
}

func NewBackoffTimer(startMicroSecond int64, maxMirocSecond int64) *BackoffTimer {
	return &BackoffTimer{
		startInterval:  startMicroSecond,
		curInterval:    startMicroSecond,
		maxInterval:    maxMirocSecond,
		expire:         0,
		useCanTryFirst: false,
	}
}

func (t *BackoffTimer) OriginalSetting(ctx context.Context) {
	t.curInterval = t.startInterval
	t.useCanTryFirst = false
	t.expire = 0
}

func (t *BackoffTimer) SuccessHit(ctx context.Context) {
	// when connect or send successfully, call this once
	if !t.useCanTryFirst {
		log.Error(context.Background(), "first use canTry() to judge", map[string]interface{}{})
	}
	t.curInterval = t.startInterval
	t.expire = 0
	t.useCanTryFirst = false
}

func (t *BackoffTimer) FailedHit(ctx context.Context) {
	// when connect or send failed, call this once
	if !t.useCanTryFirst {
		log.Error(context.Background(), "first use canTry() to judge", map[string]interface{}{})
	}
	t.expire = t.curInterval + time.Now().UnixMicro()
	t.curInterval *= 2
	if t.curInterval > t.maxInterval {
		t.curInterval = t.maxInterval
	}
	t.useCanTryFirst = false
}

func (t *BackoffTimer) CanTry() bool {
	// judge whether retry
	// Should call CanTry before SuccessHit or FailedHit
	t.useCanTryFirst = true
	return time.Now().UnixMicro() > t.expire
}