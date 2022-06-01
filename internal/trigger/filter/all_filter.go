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

package filter

import (
	"context"

	"github.com/linkall-labs/vanus/observability/log"

	ce "github.com/cloudevents/sdk-go/v2"
)

type allFilter []Filter

func NewAllFilter(filters ...Filter) Filter {
	if len(filters) == 0 {
		return nil
	}
	return append(allFilter{}, filters...)
}

func (filter allFilter) Filter(event ce.Event) Result {
	log.Debug(context.Background(), "all filter ", map[string]interface{}{"filter": filter, "event": event})
	for _, f := range filter {
		res := f.Filter(event)
		if res == FailFilter {
			return FailFilter
		}
	}
	return PassFilter
}

var _ Filter = (*allFilter)(nil)
