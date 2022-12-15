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

package action

import (
	"testing"

	"github.com/linkall-labs/vanus/internal/primitive/transform/context"

	ce "github.com/cloudevents/sdk-go/v2"
	. "github.com/smartystreets/goconvey/convey"
)

func newEvent() *ce.Event {
	e := ce.NewEvent()
	e.SetID("testID")
	e.SetType("testType")
	e.SetSource("testSource")
	return &e
}

func TestDateFormatAction(t *testing.T) {
	Convey("test format date", t, func() {
		Convey("test default time zone", func() {
			e := newEvent()
			e.SetExtension("test", "2022-11-15T15:41:25Z")
			a, err := NewAction([]interface{}{newDateFormatAction().Name(), "$.test", "Y-m-d H:i:s"})
			So(err, ShouldBeNil)
			err = a.Execute(&context.EventContext{
				Event: e,
			})
			So(err, ShouldBeNil)
			So(e.Extensions()["test"], ShouldEqual, "2022-11-15 15:41:25")
		})
		Convey("test with time zone", func() {
			e := newEvent()
			e.SetExtension("test", "2022-11-15T15:41:25Z")
			a, err := NewAction([]interface{}{newDateFormatAction().Name(), "$.test", "Y-m-d H:i:s", "EST"})
			So(err, ShouldBeNil)
			err = a.Execute(&context.EventContext{
				Event: e,
			})
			So(err, ShouldBeNil)
			So(e.Extensions()["test"], ShouldEqual, "2022-11-15 10:41:25")
		})
	})
}

func TestUnixTimeFormatAction(t *testing.T) {
	Convey("test format unix time", t, func() {
		Convey("test with default time zone", func() {
			a, err := NewAction([]interface{}{newUnixTimeFormatAction().Name(), "$.data.time", "Y-m-d H:i:s"})
			So(err, ShouldBeNil)
			ceCtx := &context.EventContext{
				Event: newEvent(),
				Data:  map[string]interface{}{"time": float64(1668498285)},
			}
			err = a.Execute(ceCtx)
			So(err, ShouldBeNil)
			So(ceCtx.Data.(map[string]interface{})["time"], ShouldEqual, "2022-11-15 07:44:45")
		})
		Convey("test with time zone", func() {
			a, err := NewAction([]interface{}{newUnixTimeFormatAction().Name(), "$.data.time", "Y-m-d H:i:s", "EST"})
			So(err, ShouldBeNil)
			ceCtx := &context.EventContext{
				Event: newEvent(),
				Data:  map[string]interface{}{"time": float64(1668498285)},
			}
			err = a.Execute(ceCtx)
			So(err, ShouldBeNil)
			So(ceCtx.Data.(map[string]interface{})["time"], ShouldEqual, "2022-11-15 02:44:45")
		})
	})
}
