// Code generated by MockGen. DO NOT EDIT.
// Source: client.go

// Package api is a generated GoMock package.
package api

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	cloudevents "github.com/vanus-labs/vanus/proto/pkg/cloudevents"
)

// MockClient is a mock of Client interface.
type MockClient struct {
	ctrl     *gomock.Controller
	recorder *MockClientMockRecorder
}

// MockClientMockRecorder is the mock recorder for MockClient.
type MockClientMockRecorder struct {
	mock *MockClient
}

// NewMockClient creates a new mock instance.
func NewMockClient(ctrl *gomock.Controller) *MockClient {
	mock := &MockClient{ctrl: ctrl}
	mock.recorder = &MockClientMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClient) EXPECT() *MockClientMockRecorder {
	return m.recorder
}

// Disconnect mocks base method.
func (m *MockClient) Disconnect(ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Disconnect", ctx)
}

// Disconnect indicates an expected call of Disconnect.
func (mr *MockClientMockRecorder) Disconnect(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Disconnect", reflect.TypeOf((*MockClient)(nil).Disconnect), ctx)
}

// Eventbus mocks base method.
func (m *MockClient) Eventbus(ctx context.Context, opts ...EventbusOption) (Eventbus, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Eventbus", varargs...)
	ret0, _ := ret[0].(Eventbus)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Eventbus indicates an expected call of Eventbus.
func (mr *MockClientMockRecorder) Eventbus(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Eventbus", reflect.TypeOf((*MockClient)(nil).Eventbus), varargs...)
}

// MockEventbus is a mock of Eventbus interface.
type MockEventbus struct {
	ctrl     *gomock.Controller
	recorder *MockEventbusMockRecorder
}

// MockEventbusMockRecorder is the mock recorder for MockEventbus.
type MockEventbusMockRecorder struct {
	mock *MockEventbus
}

// NewMockEventbus creates a new mock instance.
func NewMockEventbus(ctrl *gomock.Controller) *MockEventbus {
	mock := &MockEventbus{ctrl: ctrl}
	mock.recorder = &MockEventbusMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventbus) EXPECT() *MockEventbusMockRecorder {
	return m.recorder
}

// Close mocks base method.
func (m *MockEventbus) Close(ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Close", ctx)
}

// Close indicates an expected call of Close.
func (mr *MockEventbusMockRecorder) Close(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Close", reflect.TypeOf((*MockEventbus)(nil).Close), ctx)
}

// GetLog mocks base method.
func (m *MockEventbus) GetLog(ctx context.Context, logID uint64, opts ...LogOption) (Eventlog, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, logID}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "GetLog", varargs...)
	ret0, _ := ret[0].(Eventlog)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetLog indicates an expected call of GetLog.
func (mr *MockEventbusMockRecorder) GetLog(ctx, logID interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, logID}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetLog", reflect.TypeOf((*MockEventbus)(nil).GetLog), varargs...)
}

// ListLog mocks base method.
func (m *MockEventbus) ListLog(ctx context.Context, opts ...LogOption) ([]Eventlog, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "ListLog", varargs...)
	ret0, _ := ret[0].([]Eventlog)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ListLog indicates an expected call of ListLog.
func (mr *MockEventbusMockRecorder) ListLog(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ListLog", reflect.TypeOf((*MockEventbus)(nil).ListLog), varargs...)
}

// Reader mocks base method.
func (m *MockEventbus) Reader(opts ...ReadOption) BusReader {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Reader", varargs...)
	ret0, _ := ret[0].(BusReader)
	return ret0
}

// Reader indicates an expected call of Reader.
func (mr *MockEventbusMockRecorder) Reader(opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Reader", reflect.TypeOf((*MockEventbus)(nil).Reader), opts...)
}

// Writer mocks base method.
func (m *MockEventbus) Writer(opts ...WriteOption) BusWriter {
	m.ctrl.T.Helper()
	varargs := []interface{}{}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Writer", varargs...)
	ret0, _ := ret[0].(BusWriter)
	return ret0
}

// Writer indicates an expected call of Writer.
func (mr *MockEventbusMockRecorder) Writer(opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Writer", reflect.TypeOf((*MockEventbus)(nil).Writer), opts...)
}

// MockBusWriter is a mock of BusWriter interface.
type MockBusWriter struct {
	ctrl     *gomock.Controller
	recorder *MockBusWriterMockRecorder
}

// MockBusWriterMockRecorder is the mock recorder for MockBusWriter.
type MockBusWriterMockRecorder struct {
	mock *MockBusWriter
}

// NewMockBusWriter creates a new mock instance.
func NewMockBusWriter(ctrl *gomock.Controller) *MockBusWriter {
	mock := &MockBusWriter{ctrl: ctrl}
	mock.recorder = &MockBusWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBusWriter) EXPECT() *MockBusWriterMockRecorder {
	return m.recorder
}

// Append mocks base method.
func (m *MockBusWriter) Append(ctx context.Context, events *cloudevents.CloudEventBatch, opts ...WriteOption) ([]string, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, events}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Append", varargs...)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Append indicates an expected call of Append.
func (mr *MockBusWriterMockRecorder) Append(ctx, events interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, events}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Append", reflect.TypeOf((*MockBusWriter)(nil).Append), varargs...)
}

// MockBusReader is a mock of BusReader interface.
type MockBusReader struct {
	ctrl     *gomock.Controller
	recorder *MockBusReaderMockRecorder
}

// MockBusReaderMockRecorder is the mock recorder for MockBusReader.
type MockBusReaderMockRecorder struct {
	mock *MockBusReader
}

// NewMockBusReader creates a new mock instance.
func NewMockBusReader(ctrl *gomock.Controller) *MockBusReader {
	mock := &MockBusReader{ctrl: ctrl}
	mock.recorder = &MockBusReaderMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBusReader) EXPECT() *MockBusReaderMockRecorder {
	return m.recorder
}

// Read mocks base method.
func (m *MockBusReader) Read(ctx context.Context, opts ...ReadOption) (*cloudevents.CloudEventBatch, int64, uint64, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Read", varargs...)
	ret0, _ := ret[0].(*cloudevents.CloudEventBatch)
	ret1, _ := ret[1].(int64)
	ret2, _ := ret[2].(uint64)
	ret3, _ := ret[3].(error)
	return ret0, ret1, ret2, ret3
}

// Read indicates an expected call of Read.
func (mr *MockBusReaderMockRecorder) Read(ctx interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Read", reflect.TypeOf((*MockBusReader)(nil).Read), varargs...)
}

// MockEventlog is a mock of Eventlog interface.
type MockEventlog struct {
	ctrl     *gomock.Controller
	recorder *MockEventlogMockRecorder
}

// MockEventlogMockRecorder is the mock recorder for MockEventlog.
type MockEventlogMockRecorder struct {
	mock *MockEventlog
}

// NewMockEventlog creates a new mock instance.
func NewMockEventlog(ctrl *gomock.Controller) *MockEventlog {
	mock := &MockEventlog{ctrl: ctrl}
	mock.recorder = &MockEventlogMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEventlog) EXPECT() *MockEventlogMockRecorder {
	return m.recorder
}

// EarliestOffset mocks base method.
func (m *MockEventlog) EarliestOffset(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EarliestOffset", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// EarliestOffset indicates an expected call of EarliestOffset.
func (mr *MockEventlogMockRecorder) EarliestOffset(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EarliestOffset", reflect.TypeOf((*MockEventlog)(nil).EarliestOffset), ctx)
}

// ID mocks base method.
func (m *MockEventlog) ID() uint64 {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ID")
	ret0, _ := ret[0].(uint64)
	return ret0
}

// ID indicates an expected call of ID.
func (mr *MockEventlogMockRecorder) ID() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ID", reflect.TypeOf((*MockEventlog)(nil).ID))
}

// LatestOffset mocks base method.
func (m *MockEventlog) LatestOffset(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LatestOffset", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LatestOffset indicates an expected call of LatestOffset.
func (mr *MockEventlogMockRecorder) LatestOffset(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LatestOffset", reflect.TypeOf((*MockEventlog)(nil).LatestOffset), ctx)
}

// Length mocks base method.
func (m *MockEventlog) Length(ctx context.Context) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Length", ctx)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Length indicates an expected call of Length.
func (mr *MockEventlogMockRecorder) Length(ctx interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Length", reflect.TypeOf((*MockEventlog)(nil).Length), ctx)
}

// QueryOffsetByTime mocks base method.
func (m *MockEventlog) QueryOffsetByTime(ctx context.Context, timestamp int64) (int64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueryOffsetByTime", ctx, timestamp)
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueryOffsetByTime indicates an expected call of QueryOffsetByTime.
func (mr *MockEventlogMockRecorder) QueryOffsetByTime(ctx, timestamp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryOffsetByTime", reflect.TypeOf((*MockEventlog)(nil).QueryOffsetByTime), ctx, timestamp)
}
