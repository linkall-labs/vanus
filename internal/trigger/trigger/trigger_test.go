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

package trigger

import (
	// standard libraries
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	// third-party libraries
	ce "github.com/cloudevents/sdk-go/v2"

	// first-party libraries
	eb "github.com/linkall-labs/eventbus-go"
	"github.com/linkall-labs/eventbus-go/pkg/discovery"
	"github.com/linkall-labs/eventbus-go/pkg/discovery/record"
	"github.com/linkall-labs/eventbus-go/pkg/inmemory"

	// this project
	"github.com/linkall-labs/vanus/internal/primitive"
	pInfo "github.com/linkall-labs/vanus/internal/primitive/info"
	"github.com/linkall-labs/vanus/internal/trigger/info"
	"github.com/linkall-labs/vanus/internal/trigger/offset"
)

func Test_e2e(t *testing.T) {
	tg := NewTrigger(nil, &primitive.Subscription{
		ID:      1,
		Filters: []*primitive.SubscriptionFilter{{Exact: map[string]string{"type": "none"}}},
		Sink:    "http://localhost:18080",
	}, offset.NewOffsetManager().RegisterSubscription(1))
	emit := 0
	pre := 0
	go func() {
		for {
			time.Sleep(time.Second)
			cur := emit
			t.Logf("%v TPS: %d", time.Now(), cur-pre)
			pre = cur
		}
	}()

	ebVRN := "vanus+local:///eventbus/1"
	elVRN := "vanus+inmemory:///eventlog/1?eventbus=1&keepalive=true"
	br := &record.EventBus{
		VRN: ebVRN,
		Logs: []*record.EventLog{
			{
				VRN:  elVRN,
				Mode: record.PremWrite | record.PremRead,
			},
		},
	}
	inmemory.UseInMemoryLog("vanus+inmemory")
	ns := inmemory.UseNameService("vanus+local")
	// register metadata of eventbus
	vrn, err := discovery.ParseVRN(ebVRN)
	if err != nil {
		panic(err.Error())
	}
	ns.Register(vrn, br)

	go func() {
		w, err := eb.OpenBusWriter(ebVRN)
		if err != nil {
			t.Fatal(err)
		}

		// FIXME
		time.Sleep(10 * time.Second)

		for i := 0; i < 10000000000; i++ {
			// Create an Event.
			event := ce.NewEvent()
			event.SetID(fmt.Sprintf("%d", i))
			event.SetSource("example/uri")
			event.SetType("none")
			event.SetData(ce.ApplicationJSON, map[string]string{"hello": "world", "type": "none"})

			_, err = w.Append(context.Background(), &event)
			if err != nil {
				t.Log(err)
			}
		}

		w.Close()
	}()
	go func() {
		ls, err := eb.LookupReadableLogs(context.Background(), ebVRN)
		if err != nil {
			t.Fatal(err)
		}

		r, err := eb.OpenLogReader(ls[0].VRN)
		if err != nil {
			t.Fatal(err)
		}

		_, err = r.Seek(context.Background(), 0, io.SeekStart)
		if err != nil {
			t.Fatal(err)
		}
		var offset uint64
		for {
			events, err := r.Read(context.Background(), 5)
			if err != nil {
				t.Fatal(err)
			}

			if len(events) == 0 {
				time.Sleep(time.Second)
				continue
			}

			for _, e := range events {
				tg.EventArrived(context.Background(), info.EventRecord{
					EventOffset: info.EventOffset{
						Event: e,
						OffsetInfo: pInfo.OffsetInfo{
							Offset:     offset,
							EventLogID: 1,
						},
					},
				})
				emit++
			}
		}

		r.Close()
	}()
	receive := 0
	receivePre := 0
	go http.ListenAndServe(":18080", http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		receive++
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			fmt.Println(err)
		}
		var _ = string(body)
	}))
	go func() {
		for {
			time.Sleep(time.Second)
			cur := receive
			t.Logf("%v RECEIVE TPS: %d", time.Now(), cur-receivePre)
			receivePre = cur
		}
	}()
	tg.Start()

	time.Sleep(time.Hour)
	tg.Stop()
}
