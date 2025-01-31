// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/cloudzero/cloudzero-insights-controller/app/types (interfaces: Store)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	types "github.com/cloudzero/cloudzero-insights-controller/app/types"
	gomock "github.com/golang/mock/gomock"
)

// MockStore is a mock of Store interface.
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore.
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance.
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// All mocks base method.
func (m *MockStore) All(arg0 context.Context, arg1 *string) (types.MetricRange, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "All", arg0, arg1)
	ret0, _ := ret[0].(types.MetricRange)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// All indicates an expected call of All.
func (mr *MockStoreMockRecorder) All(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "All", reflect.TypeOf((*MockStore)(nil).All), arg0, arg1)
}

// Delete mocks base method.
func (m *MockStore) Delete(arg0 context.Context, arg1 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockStoreMockRecorder) Delete(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockStore)(nil).Delete), arg0, arg1)
}

// Get mocks base method.
func (m *MockStore) Get(arg0 context.Context, arg1 string) (*types.Metric, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(*types.Metric)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockStoreMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockStore)(nil).Get), arg0, arg1)
}

// Put mocks base method.
func (m *MockStore) Put(arg0 context.Context, arg1 ...types.Metric) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Put", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// Put indicates an expected call of Put.
func (mr *MockStoreMockRecorder) Put(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Put", reflect.TypeOf((*MockStore)(nil).Put), arg0, arg1)
}

// Flush mocks base method.
func (m *MockStore) Flush() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Flush")
	ret0, _ := ret[0].(error)
	return ret0
}

// Flush indicates an expected call of Flush.
func (mr *MockStoreMockRecorder) Flush() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Flush", reflect.TypeOf((*MockStore)(nil).Flush))
}

// Pending mocks base method.
func (m *MockStore) Pending() int {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Pending")
	ret0, _ := ret[0].(int)
	return ret0
}

// Pending indicates an expected call of Pending.
func (mr *MockStoreMockRecorder) Pending() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Pending", reflect.TypeOf((*MockStore)(nil).Pending))
}
