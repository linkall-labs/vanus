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

package client

import (
	"context"

	ce "github.com/cloudevents/sdk-go/v2"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

type http struct {
	client ce.Client
}

func NewHTTPClient(url string) EventClient {
	c, _ := ce.NewClientHTTP(ce.WithTarget(url))
	return &http{
		client: c,
	}
}

func (c *http) Send(ctx context.Context, event ce.Event) Result {
	if err := c.client.Send(ctx, event); !ce.IsACK(err) {
		r := Result{Err: err}
		if v, ok := err.(*cehttp.Result); ok {
			r.StatusCode = v.StatusCode
		} else {
			r.StatusCode = ErrHTTP
		}
		return r
	}
	return Success
}
