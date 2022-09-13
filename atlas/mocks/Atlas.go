package mocks

import "github.com/stretchr/testify/mock"

import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
import "golang.org/x/net/context"

type Atlas struct {
	mock.Mock
}

// GetIntersectingPath provides a mock function with given fields: _a0
func (_m *Atlas) GetIntersectingPath(_a0 context.Context) (pb.Atlas_GetIntersectingPathClient, error) {
	ret := _m.Called(_a0)

	var r0 pb.Atlas_GetIntersectingPathClient
	if rf, ok := ret.Get(0).(func(context.Context) pb.Atlas_GetIntersectingPathClient); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(pb.Atlas_GetIntersectingPathClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPathsWithToken provides a mock function with given fields: _a0
func (_m *Atlas) GetPathsWithToken(_a0 context.Context) (pb.Atlas_GetPathsWithTokenClient, error) {
	ret := _m.Called(_a0)

	var r0 pb.Atlas_GetPathsWithTokenClient
	if rf, ok := ret.Get(0).(func(context.Context) pb.Atlas_GetPathsWithTokenClient); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(pb.Atlas_GetPathsWithTokenClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
