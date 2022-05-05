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

package template

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestExecuteJsonString(t *testing.T) {
	p := NewParser()
	p.Parse(`{"key":"${str}"}`)
	Convey("no data", t, func() {
		m := make(map[string]Data)
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":""}`)
	})
	Convey("null", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNullData()
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":""}`)
	})
	Convey("no exist", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNoExistData()
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":""}`)
	})
	Convey("string", t, func() {
		m := make(map[string]Data)
		m["str"] = NewTextData([]byte("str"))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":"str"}`)
	})
	Convey("other num", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte("123"))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":"123"}`)
	})
}

func TestExecuteJsonValue(t *testing.T) {
	p := NewParser()
	p.Parse(`{"key":${str}}`)
	Convey("no data", t, func() {
		m := make(map[string]Data)
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":null}`)
	})
	Convey("null", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNullData()
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":null}`)
	})
	Convey("no exist", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNoExistData()
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":null}`)
	})
	Convey("string", t, func() {
		m := make(map[string]Data)
		m["str"] = NewTextData([]byte("str"))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":"str"}`)
	})
	Convey("other num", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte("123"))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":123}`)
	})
	Convey("other bool", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte("true"))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":true}`)
	})

	Convey("other obj", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte(`{"k":"v"}`))
		v := p.Execute(m)
		So(v, ShouldEqual, `{"key":{"k":"v"}}`)
	})
}

func TestExecuteText(t *testing.T) {
	p := NewParser()
	p.Parse(`abc ${str}`)
	Convey("no data", t, func() {
		m := make(map[string]Data)
		v := p.Execute(m)
		So(v, ShouldEqual, `abc `)
	})
	Convey("null", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNullData()
		v := p.Execute(m)
		So(v, ShouldEqual, `abc `)
	})
	Convey("no exist", t, func() {
		m := make(map[string]Data)
		m["str"] = NewNoExistData()
		v := p.Execute(m)
		So(v, ShouldEqual, `abc `)
	})
	Convey("string", t, func() {
		m := make(map[string]Data)
		m["str"] = NewTextData([]byte("str"))
		v := p.Execute(m)
		So(v, ShouldEqual, `abc str`)
	})
	Convey("other num", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte("123"))
		v := p.Execute(m)
		So(v, ShouldEqual, `abc 123`)
	})
	Convey("other bool", t, func() {
		m := make(map[string]Data)
		m["str"] = NewOtherData([]byte("true"))
		v := p.Execute(m)
		So(v, ShouldEqual, `abc true`)
	})

}
