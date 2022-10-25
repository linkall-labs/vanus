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

import (
	// standard libraries
	"context"

	"github.com/linkall-labs/vanus/observability/tracing"
	"go.opentelemetry.io/otel/trace"

	// third-party libraries
	"github.com/linkall-labs/vanus/pkg/controller"
	"google.golang.org/grpc/credentials/insecure"

	// first-party libraries
	"github.com/linkall-labs/vanus/client/pkg/record"
	ctrlpb "github.com/linkall-labs/vanus/proto/pkg/controller"
	metapb "github.com/linkall-labs/vanus/proto/pkg/meta"
)

func NewNameService(endpoints []string) *NameService {
	return &NameService{
		client: controller.NewEventbusClient(endpoints, insecure.NewCredentials()),
		tracer: tracing.NewTracer("internal.discovery.eventbus", trace.SpanKindClient),
	}
}

type NameService struct {
	client ctrlpb.EventBusControllerClient
	tracer *tracing.Tracer
}

func (ns *NameService) LookupWritableLogs(ctx context.Context, eventbus string) ([]*record.Eventlog, error) {
	ctx, span := ns.tracer.Start(ctx, "LookupWritableLogs")
	defer span.End()

	req := &metapb.EventBus{
		Name: eventbus,
	}

	resp, err := ns.client.GetEventBus(ctx, req)

	if err != nil {
		return nil, err
	}
	return toLogs(resp.GetLogs()), nil
}

func (ns *NameService) LookupReadableLogs(ctx context.Context, eventbus string) ([]*record.Eventlog, error) {
	ctx, span := ns.tracer.Start(ctx, "LookupReadableLogs")
	defer span.End()

	req := &metapb.EventBus{
		Name: eventbus,
	}

	resp, err := ns.client.GetEventBus(ctx, req)
	if err != nil {
		return nil, err
	}

	return toLogs(resp.GetLogs()), nil
}

func toLogs(logpbs []*metapb.EventLog) []*record.Eventlog {
	if len(logpbs) <= 0 {
		return make([]*record.Eventlog, 0)
	}
	logs := make([]*record.Eventlog, 0, len(logpbs))
	for _, logpb := range logpbs {
		logs = append(logs, toLog(logpb))
	}
	return logs
}

func toLog(logpb *metapb.EventLog) *record.Eventlog {
	log := &record.Eventlog{
		ID:   logpb.GetEventLogId(),
		Mode: record.PremWrite | record.PremRead,
	}
	return log
}
