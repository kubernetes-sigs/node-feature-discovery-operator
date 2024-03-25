// Code generated by MockGen. DO NOT EDIT.
// Source: configmap.go
//
// Generated by this command:
//
//	mockgen -source=configmap.go -package=configmap -destination=mock_configmap.go ConfigMapAPI
//
// Package configmap is a generated GoMock package.
package configmap

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	v10 "sigs.k8s.io/node-feature-discovery-operator/api/v1"
)

// MockConfigMapAPI is a mock of ConfigMapAPI interface.
type MockConfigMapAPI struct {
	ctrl     *gomock.Controller
	recorder *MockConfigMapAPIMockRecorder
}

// MockConfigMapAPIMockRecorder is the mock recorder for MockConfigMapAPI.
type MockConfigMapAPIMockRecorder struct {
	mock *MockConfigMapAPI
}

// NewMockConfigMapAPI creates a new mock instance.
func NewMockConfigMapAPI(ctrl *gomock.Controller) *MockConfigMapAPI {
	mock := &MockConfigMapAPI{ctrl: ctrl}
	mock.recorder = &MockConfigMapAPIMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockConfigMapAPI) EXPECT() *MockConfigMapAPIMockRecorder {
	return m.recorder
}

// SetWorkerConfigMapAsDesired mocks base method.
func (m *MockConfigMapAPI) SetWorkerConfigMapAsDesired(ctx context.Context, nfdInstance *v10.NodeFeatureDiscovery, workerCM *v1.ConfigMap) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetWorkerConfigMapAsDesired", ctx, nfdInstance, workerCM)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetWorkerConfigMapAsDesired indicates an expected call of SetWorkerConfigMapAsDesired.
func (mr *MockConfigMapAPIMockRecorder) SetWorkerConfigMapAsDesired(ctx, nfdInstance, workerCM any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetWorkerConfigMapAsDesired", reflect.TypeOf((*MockConfigMapAPI)(nil).SetWorkerConfigMapAsDesired), ctx, nfdInstance, workerCM)
}