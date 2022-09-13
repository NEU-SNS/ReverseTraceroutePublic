package mocks

import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
import "github.com/stretchr/testify/mock"

import context "golang.org/x/net/context"
import grpc "google.golang.org/grpc"

type AtlasClient struct {
	mock.Mock
}

// GetIntersectingPath provides a mock function with given fields: ctx, opts
func (_m *AtlasClient) GetIntersectingPath(ctx context.Context, opts ...grpc.CallOption) (pb.Atlas_GetIntersectingPathClient, error) {
	ret := _m.Called(ctx, opts)

	var r0 pb.Atlas_GetIntersectingPathClient
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) pb.Atlas_GetIntersectingPathClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		r0 = ret.Get(0).(pb.Atlas_GetIntersectingPathClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPathsWithToken provides a mock function with given fields: ctx, opts
func (_m *AtlasClient) GetPathsWithToken(ctx context.Context, opts ...grpc.CallOption) (pb.Atlas_GetPathsWithTokenClient, error) {
	ret := _m.Called(ctx, opts)

	var r0 pb.Atlas_GetPathsWithTokenClient
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) pb.Atlas_GetPathsWithTokenClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		r0 = ret.Get(0).(pb.Atlas_GetPathsWithTokenClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
