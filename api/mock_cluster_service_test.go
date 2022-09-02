// Code generated by mockery v2.14.0. DO NOT EDIT.

package api

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	metastore "github.com/spinup-host/spinup/internal/metastore"
)

// mockClusterService is an autogenerated mock type for the clusterService type
type mockClusterService struct {
	mock.Mock
}

// CreateService provides a mock function with given fields: ctx, info
func (_m *mockClusterService) CreateService(ctx context.Context, info *metastore.ClusterInfo) error {
	ret := _m.Called(ctx, info)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, *metastore.ClusterInfo) error); ok {
		r0 = rf(ctx, info)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// GetClusterByID provides a mock function with given fields: ctx, clusterID
func (_m *mockClusterService) GetClusterByID(ctx context.Context, clusterID string) (metastore.ClusterInfo, error) {
	ret := _m.Called(ctx, clusterID)

	var r0 metastore.ClusterInfo
	if rf, ok := ret.Get(0).(func(context.Context, string) metastore.ClusterInfo); ok {
		r0 = rf(ctx, clusterID)
	} else {
		r0 = ret.Get(0).(metastore.ClusterInfo)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, clusterID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ListClusters provides a mock function with given fields: ctx
func (_m *mockClusterService) ListClusters(ctx context.Context) ([]metastore.ClusterInfo, error) {
	ret := _m.Called(ctx)

	var r0 []metastore.ClusterInfo
	if rf, ok := ret.Get(0).(func(context.Context) []metastore.ClusterInfo); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]metastore.ClusterInfo)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

type mockConstructorTestingTnewMockClusterService interface {
	mock.TestingT
	Cleanup(func())
}

// newMockClusterService creates a new instance of mockClusterService. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func newMockClusterService(t mockConstructorTestingTnewMockClusterService) *mockClusterService {
	mock := &mockClusterService{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}