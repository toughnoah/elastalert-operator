// Code generated by MockGen. DO NOT EDIT.
// Source: podtemplate.go

// Package mock_podspec is a generated GoMock package.
package mock_podspec

import (
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
)

// MockUtil is a mock of Util interface.
type MockUtil struct {
	ctrl     *gomock.Controller
	recorder *MockUtilMockRecorder
}

// MockUtilMockRecorder is the mock recorder for MockUtil.
type MockUtilMockRecorder struct {
	mock *MockUtil
}

// NewMockUtil creates a new mock instance.
func NewMockUtil(ctrl *gomock.Controller) *MockUtil {
	mock := &MockUtil{ctrl: ctrl}
	mock.recorder = &MockUtilMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockUtil) EXPECT() *MockUtilMockRecorder {
	return m.recorder
}

// GetUtcTime mocks base method.
func (m *MockUtil) GetUtcTime() time.Time {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUtcTime")
	ret0, _ := ret[0].(time.Time)
	return ret0
}

// GetUtcTime indicates an expected call of GetUtcTime.
func (mr *MockUtilMockRecorder) GetUtcTime() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUtcTime", reflect.TypeOf((*MockUtil)(nil).GetUtcTime))
}

// GetUtcTimeString mocks base method.
func (m *MockUtil) GetUtcTimeString() string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUtcTimeString")
	ret0, _ := ret[0].(string)
	return ret0
}

// GetUtcTimeString indicates an expected call of GetUtcTimeString.
func (mr *MockUtilMockRecorder) GetUtcTimeString() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUtcTimeString", reflect.TypeOf((*MockUtil)(nil).GetUtcTimeString))
}