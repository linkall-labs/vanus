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

package filter_test

import (
	"testing"

	"github.com/linkall-labs/vanus/internal/trigger/filter"

	ce "github.com/cloudevents/sdk-go/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSuffixFilter(t *testing.T) {
	event := ce.NewEvent()
	event.SetID("testID")
	event.SetSource("testSource")

	Convey("suffix filter pass", t, func() {
		f := filter.NewSuffixFilter(map[string]string{
			"id":     "ID",
			"source": "Source",
		})
		result := f.Filter(event)
		So(result, ShouldEqual, filter.PassFilter)
	})

	Convey("suffix filter pass", t, func() {
		f := filter.NewPrefixFilter(map[string]string{
			"id":     "un",
			"source": "test",
		})
		result := f.Filter(event)
		So(result, ShouldEqual, filter.FailFilter)
	})

}
