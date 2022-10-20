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

package pipeline

import (
	"github.com/linkall-labs/vanus/internal/primitive"
	"github.com/linkall-labs/vanus/internal/trigger/context"
	"github.com/linkall-labs/vanus/observability/log"
)

type Pipeline struct {
	actions []*action
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		actions: make([]*action, 0),
	}
}

func (p *Pipeline) Parse(actions []*primitive.Action) {
	p.actions = make([]*action, 0, len(actions))
	for i := range actions {
		_action, err := newAction(actions[i].Command)
		if err != nil {
			log.Warning(nil, "new action error", map[string]interface{}{
				log.KeyError: err,
				"command":    actions[i].Command,
			})
			continue
		}
		p.actions = append(p.actions, _action)
	}
}

func (p *Pipeline) Run(ceCtx *context.EventContext) error {
	for _, a := range p.actions {
		err := a.Execute(ceCtx)
		if err != nil {
			log.Warning(nil, "action execute error", map[string]interface{}{
				log.KeyError: err,
				"command":    a.fn.Name(),
			})
		}
	}
	return nil
}
