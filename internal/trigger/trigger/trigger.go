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

//go:generate mockgen -source=trigger.go  -destination=mock_trigger.go -package=trigger
package trigger

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/linkall-labs/vanus/internal/primitive"
	pInfo "github.com/linkall-labs/vanus/internal/primitive/info"
	"github.com/linkall-labs/vanus/internal/primitive/vanus"
	"github.com/linkall-labs/vanus/internal/trigger/client"
	"github.com/linkall-labs/vanus/internal/trigger/filter"
	"github.com/linkall-labs/vanus/internal/trigger/info"
	"github.com/linkall-labs/vanus/internal/trigger/offset"
	"github.com/linkall-labs/vanus/internal/trigger/reader"
	"github.com/linkall-labs/vanus/internal/trigger/transform"
	"github.com/linkall-labs/vanus/internal/util"
	"github.com/linkall-labs/vanus/observability/log"
	"github.com/linkall-labs/vanus/observability/metrics"

	ce "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/ratelimit"
)

type State string

const (
	TriggerCreated   = "created"
	TriggerPending   = "pending"
	TriggerRunning   = "running"
	TriggerSleep     = "sleep"
	TriggerPaused    = "paused"
	TriggerStopped   = "stopped"
	TriggerDestroyed = "destroyed"
)

type Trigger interface {
	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Change(ctx context.Context, subscription *primitive.Subscription) error
	GetOffsets(ctx context.Context) pInfo.ListOffsetInfo
	ResetOffsetToTimestamp(ctx context.Context, timestamp int64) (pInfo.ListOffsetInfo, error)
}

type trigger struct {
	subscriptionIDStr string

	subscription  *primitive.Subscription
	offsetManager *offset.SubscriptionOffset
	reader        reader.Reader
	eventCh       chan info.EventRecord
	sendCh        chan info.EventRecord
	client        client.EventClient
	filter        filter.Filter
	transformer   *transform.Transformer
	rateLimiter   ratelimit.Limiter
	config        Config

	state State
	stop  context.CancelFunc
	lock  sync.RWMutex
	wg    util.Group
}

func NewTrigger(subscription *primitive.Subscription, opts ...Option) Trigger {
	t := &trigger{
		stop:              func() {},
		config:            defaultConfig(),
		state:             TriggerCreated,
		filter:            filter.GetFilter(subscription.Filters),
		offsetManager:     offset.NewSubscriptionOffset(subscription.ID),
		subscription:      subscription,
		subscriptionIDStr: subscription.ID.String(),
		transformer:       transform.NewTransformer(subscription.Transformer),
	}
	t.applyOptions(opts...)
	if t.rateLimiter == nil {
		t.rateLimiter = ratelimit.NewUnlimited()
	}
	return t
}

func (t *trigger) applyOptions(opts ...Option) {
	for _, fn := range opts {
		fn(t)
	}
}

func (t *trigger) getClient() client.EventClient {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.client
}

func newEventClient(sink primitive.URI,
	protocol primitive.Protocol,
	credential primitive.SinkCredential) client.EventClient {
	switch protocol {
	case primitive.AwsLambdaProtocol:
		_credential, _ := credential.(*primitive.CloudSinkCredential)
		return client.NewAwsLambdaClient(_credential.AccessKeyID, _credential.SecretAccessKey, string(sink))
	default:
		return client.NewHTTPClient(string(sink))
	}
}

func (t *trigger) changeTarget(sink primitive.URI,
	protocol primitive.Protocol,
	credential primitive.SinkCredential) error {
	eventCli := newEventClient(sink, protocol, credential)
	t.lock.Lock()
	defer t.lock.Unlock()
	t.client = eventCli
	t.subscription.Sink = sink
	t.subscription.Protocol = protocol
	t.subscription.SinkCredential = credential
	return nil
}

func (t *trigger) getFilter() filter.Filter {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.filter
}

func (t *trigger) changeFilter(filters []*primitive.SubscriptionFilter) {
	f := filter.GetFilter(filters)
	t.lock.Lock()
	defer t.lock.Unlock()
	t.filter = f
	t.subscription.Filters = filters
}

func (t *trigger) getTransformer() *transform.Transformer {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return t.transformer
}

func (t *trigger) changeTransformer(transformer *primitive.Transformer) {
	trans := transform.NewTransformer(transformer)
	t.lock.Lock()
	defer t.lock.Unlock()
	t.transformer = trans
	t.subscription.Transformer = transformer
}

func (t *trigger) changeConfig(config primitive.SubscriptionConfig) {
	t.lock.Lock()
	defer t.lock.Unlock()
	if config.RateLimit != t.config.RateLimit {
		t.applyOptions(WithRateLimit(config.RateLimit))
	}
	t.subscription.Config = config
}

// eventArrived for test.
func (t *trigger) eventArrived(ctx context.Context, event info.EventRecord) error {
	select {
	case t.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *trigger) retrySendEvent(ctx context.Context, e *ce.Event) error {
	retryTimes := 0
	doFunc := func() error {
		timeout, cancel := context.WithTimeout(ctx, t.config.SendTimeOut)
		defer cancel()
		t.rateLimiter.Take()
		return t.getClient().Send(timeout, *e)
	}
	var err error
	transformer := t.getTransformer()
	if transformer != nil {
		err = transformer.Execute(e)
		if err != nil {
			return err
		}
	}
	for retryTimes < t.config.MaxRetryTimes {
		retryTimes++
		startTime := time.Now()
		if err = doFunc(); err != nil {
			metrics.TriggerPushEventCounter.WithLabelValues(t.subscriptionIDStr, metrics.LabelValuePushEventFail).Inc()
			log.Info(ctx, "process event error", map[string]interface{}{
				"error": err, "retryTimes": retryTimes,
			})
			time.Sleep(t.config.RetryInterval)
		} else {
			metrics.TriggerPushEventCounter.WithLabelValues(t.subscriptionIDStr, metrics.LabelValuePushEventSuccess).Inc()
			metrics.TriggerPushEventRtCounter.WithLabelValues(t.subscriptionIDStr).Observe(time.Since(startTime).Seconds())
			log.Debug(ctx, "send ce event success", map[string]interface{}{
				"event": e,
			})
			return nil
		}
	}
	return err
}

func (t *trigger) runEventProcess(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-t.eventCh:
			if !ok {
				return
			}
			t.offsetManager.EventReceive(event.OffsetInfo)
			if res := filter.Run(t.getFilter(), *event.Event); res == filter.FailFilter {
				t.offsetManager.EventCommit(event.OffsetInfo)
				continue
			}
			t.sendCh <- event
			metrics.TriggerFilterMatchCounter.WithLabelValues(t.subscriptionIDStr).Inc()
		}
	}
}

func (t *trigger) runEventSend(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-t.sendCh:
			if !ok {
				return
			}
			err := t.retrySendEvent(ctx, event.Event)
			if err != nil {
				log.Error(ctx, "send event to sink failed", map[string]interface{}{
					log.KeyError: err,
					"event":      event,
				})
			}
			t.offsetManager.EventCommit(event.OffsetInfo)
		}
	}
}

func (t *trigger) getReaderConfig() reader.Config {
	controllers := t.config.Controllers
	sub := t.subscription
	ebVrn := fmt.Sprintf("vanus://%s/eventbus/%s?controllers=%s",
		controllers[0],
		sub.EventBus,
		strings.Join(controllers, ","))
	var offsetTimestamp int64
	if sub.Config.OffsetTimestamp != nil {
		offsetTimestamp = int64(*sub.Config.OffsetTimestamp)
	}
	return reader.Config{
		EventBusName:    sub.EventBus,
		EventBusVRN:     ebVrn,
		SubscriptionID:  sub.ID,
		OffsetType:      sub.Config.OffsetType,
		OffsetTimestamp: offsetTimestamp,
		Offset:          getOffset(t.offsetManager, sub),
	}
}

// getOffset from subscription,if subscriptionOffset exist,use subscriptionOffset.
func getOffset(subscriptionOffset *offset.SubscriptionOffset, sub *primitive.Subscription) map[vanus.ID]uint64 {
	// get offset from subscription
	offsetMap := make(map[vanus.ID]uint64)
	for _, o := range sub.Offsets {
		offsetMap[o.EventLogID] = o.Offset
	}
	// get offset from offset manager
	offsets := subscriptionOffset.GetCommit()
	for _, offset := range offsets {
		offsetMap[offset.EventLogID] = offset.Offset
	}
	return offsetMap
}

func (t *trigger) Init(ctx context.Context) error {
	t.client = newEventClient(t.subscription.Sink, t.subscription.Protocol, t.subscription.SinkCredential)
	t.eventCh = make(chan info.EventRecord, t.config.BufferSize)
	t.sendCh = make(chan info.EventRecord, t.config.BufferSize)
	t.reader = reader.NewReader(t.getReaderConfig(), t.eventCh)
	t.offsetManager.Clear()
	return nil
}

func (t *trigger) Start(ctx context.Context) error {
	log.Info(ctx, "trigger start...", map[string]interface{}{
		log.KeySubscriptionID: t.subscription.ID,
	})
	_ = t.reader.Start()
	ctx, cancel := context.WithCancel(context.Background())
	t.stop = cancel
	for i := 0; i < t.config.FilterProcessSize; i++ {
		t.wg.StartWithContext(ctx, t.runEventProcess)
	}
	for i := 0; i < t.config.SendProcessSize; i++ {
		t.wg.StartWithContext(ctx, t.runEventSend)
	}
	t.state = TriggerRunning
	log.Info(ctx, "trigger started", map[string]interface{}{
		log.KeySubscriptionID: t.subscription.ID,
	})
	return nil
}

func (t *trigger) Stop(ctx context.Context) error {
	log.Info(ctx, "trigger stop...", map[string]interface{}{
		log.KeySubscriptionID: t.subscription.ID,
	})
	if t.state == TriggerStopped {
		return nil
	}
	t.stop()
	t.reader.Close()
	t.wg.Wait()
	close(t.eventCh)
	close(t.sendCh)
	t.state = TriggerStopped
	log.Info(ctx, "trigger stopped", map[string]interface{}{
		log.KeySubscriptionID: t.subscription.ID,
	})
	return nil
}

func (t *trigger) Change(ctx context.Context, subscription *primitive.Subscription) error {
	if t.subscription.Sink != subscription.Sink ||
		t.subscription.Protocol != subscription.Protocol ||
		!reflect.DeepEqual(t.subscription.SinkCredential, subscription.SinkCredential) {
		err := t.changeTarget(subscription.Sink, subscription.Protocol, subscription.SinkCredential)
		if err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(t.subscription.Filters, subscription.Filters) {
		t.changeFilter(subscription.Filters)
	}
	if !reflect.DeepEqual(t.subscription.Transformer, subscription.Transformer) {
		t.changeTransformer(subscription.Transformer)
	}
	if !reflect.DeepEqual(t.subscription.Config, subscription.Config) {
		t.changeConfig(subscription.Config)
	}
	return nil
}

func (t *trigger) ResetOffsetToTimestamp(ctx context.Context, timestamp int64) (pInfo.ListOffsetInfo, error) {
	offsets, err := t.reader.GetOffsetByTimestamp(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	t.subscription.Offsets = offsets
	t.offsetManager.Clear()
	return offsets, nil
}

func (t *trigger) GetOffsets(ctx context.Context) pInfo.ListOffsetInfo {
	return t.offsetManager.GetCommit()
}
