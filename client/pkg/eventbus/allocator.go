// Copyright 2022 Linkall Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eventbus

import "sync"

func NewAllocator() *Allocator {
	return &Allocator{
		eventbuses: make(map[string]EventBus),
		mu:         sync.RWMutex{},
	}
}

type Allocator struct {
	eventbuses map[string]EventBus
	mu         sync.RWMutex
}

// Get acquire EventBus.
func (a *Allocator) Get(vrn string) (EventBus, error) {
	cfg, err := ParseVRN(vrn)
	if err != nil {
		return nil, err
	}

	s := cfg.VRN.String()
	eb := func() EventBus {
		a.mu.RLock()
		defer a.mu.RUnlock()
		eb := a.eventbuses[s]
		if eb != nil {
			eb.Acquire()
		}
		return eb
	}()

	if eb == nil {
		a.mu.Lock()
		defer a.mu.Unlock()

		eb = a.eventbuses[s]
		if eb == nil { // double check
			eb = newEventBus(cfg)
			a.eventbuses[s] = eb
		}
		eb.Acquire()
	}

	return eb, nil
}

// Put release EventBus.
func (a *Allocator) Put(eb EventBus) {
	d := false
	if eb.Release() {
		func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			if eb.UseCount() == 0 { // double check
				delete(a.eventbuses, eb.VRN().String())
				d = true
			}
		}()
	}
	if d {
		eb.Close()
	}
}
