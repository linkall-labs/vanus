// Code generated by MockGen. DO NOT EDIT.
// Source: worker.go

// Package worker is a generated GoMock package.
package worker

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	primitive "github.com/linkall-labs/vanus/internal/primitive"
	info "github.com/linkall-labs/vanus/internal/primitive/info"
)

// MockSubscriptionWorker is a mock of SubscriptionWorker interface.
type MockSubscriptionWorker struct {
	ctrl     *gomock.Controller
	recorder *MockSubscriptionWorkerMockRecorder
}

// MockSubscriptionWorkerMockRecorder is the mock recorder for MockSubscriptionWorker.
type MockSubscriptionWorkerMockRecorder struct {
	mock *MockSubscriptionWorker
}

// NewMockSubscriptionWorker creates a new mock instance.
func NewMockSubscriptionWorker(ctrl *gomock.Controller) *MockSubscriptionWorker {
	mock := &MockSubscriptionWorker{ctrl: ctrl}
	mock.recorder = &MockSubscriptionWorkerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscriptionWorker) EXPECT() *MockSubscriptionWorkerMockRecorder {
	return m.recorder
}

// Change mocks base method.
func (m *MockSubscriptionWorker) Change(ctx context.Context, subscription *primitive.Subscription) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Change", ctx, subscription)
	ret0, _ := ret[0].(error)
	return ret0
}

// Change indicates an expected call of Change.
func (mr *MockSubscriptionWorkerMockRecorder) Change(ctx, subscription interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Change", reflect.TypeOf((*MockSubscriptionWorker)(nil).Change), ctx, subscription)
}

// GetStopTime mocks base method.
func (m *MockSubscriptionWorker) GetStopTime() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetStopTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetStopTime indicates an expected call of GetStopTime.
func (mr *MockSubscriptionWorkerMockRecorder) GetStopTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetStopTime", reflect.TypeOf((*MockSubscriptionWorker)(nil).GetStopTime))
}

// IsStart mocks base method.
func (m *MockSubscriptionWorker) IsStart() bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "IsStart")
	ret0, _ := ret[0].(bool)
	return ret0
}

// IsStart indicates an expected call of IsStart.
func (mr *MockSubscriptionWorkerMockRecorder) IsStart() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "IsStart", reflect.TypeOf((*MockSubscriptionWorker)(nil).IsStart))
}

// ResetOffsetToTimestamp mocks base method.
func (m *MockSubscriptionWorker) ResetOffsetToTimestamp(ctx context.Context, timestamp uint64) (info.ListOffsetInfo, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ResetOffsetToTimestamp", ctx, timestamp)
	ret0, _ := ret[0].(info.ListOffsetInfo)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ResetOffsetToTimestamp indicates an expected call of ResetOffsetToTimestamp.
func (mr *MockSubscriptionWorkerMockRecorder) ResetOffsetToTimestamp(ctx, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ResetOffsetToTimestamp", reflect.TypeOf((*MockSubscriptionWorker)(nil).ResetOffsetToTimestamp), ctx, timestamp)
}

// Run mocks base method.
func (m *MockSubscriptionWorker) Run(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockSubscriptionWorkerMockRecorder) Run(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockSubscriptionWorker)(nil).Run), ctx)
}

// Stop mocks base method.
func (m *MockSubscriptionWorker) Stop(ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop", ctx)
}

// Stop indicates an expected call of Stop.
func (mr *MockSubscriptionWorkerMockRecorder) Stop(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockSubscriptionWorker)(nil).Stop), ctx)
}
