package mocks

import (
	"github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/stretchr/testify/mock"

	grpc "google.golang.org/grpc"
)

import context "golang.org/x/net/context"

type ControllerClient struct {
	mock.Mock
}

// Ping provides a mock function with given fields: ctx, in, opts
func (_m *ControllerClient) Ping(ctx context.Context, in *datamodel.PingArg, opts ...grpc.CallOption) (controllerapi.Controller_PingClient, error) {
	ret := _m.Called(ctx, in, opts)

	var r0 controllerapi.Controller_PingClient
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.PingArg, ...grpc.CallOption) controllerapi.Controller_PingClient); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_PingClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.PingArg, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Traceroute provides a mock function with given fields: ctx, in, opts
func (_m *ControllerClient) Traceroute(ctx context.Context, in *datamodel.TracerouteArg, opts ...grpc.CallOption) (controllerapi.Controller_TracerouteClient, error) {
	ret := _m.Called(ctx, in, opts)

	var r0 controllerapi.Controller_TracerouteClient
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.TracerouteArg, ...grpc.CallOption) controllerapi.Controller_TracerouteClient); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_TracerouteClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.TracerouteArg, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetVPs provides a mock function with given fields: ctx, in, opts
func (_m *ControllerClient) GetVPs(ctx context.Context, in *datamodel.VPRequest, opts ...grpc.CallOption) (*datamodel.VPReturn, error) {
	ret := _m.Called(ctx, in, opts)

	var r0 *datamodel.VPReturn
	if rf, ok := ret.Get(0).(func(context.Context, *datamodel.VPRequest, ...grpc.CallOption) *datamodel.VPReturn); ok {
		r0 = rf(ctx, in, opts...)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*datamodel.VPReturn)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, *datamodel.VPRequest, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, in, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// ReceiveSpoofedProbes provides a mock function with given fields: ctx, opts
func (_m *ControllerClient) ReceiveSpoofedProbes(ctx context.Context, opts ...grpc.CallOption) (controllerapi.Controller_ReceiveSpoofedProbesClient, error) {
	ret := _m.Called(ctx, opts)

	var r0 controllerapi.Controller_ReceiveSpoofedProbesClient
	if rf, ok := ret.Get(0).(func(context.Context, ...grpc.CallOption) controllerapi.Controller_ReceiveSpoofedProbesClient); ok {
		r0 = rf(ctx, opts...)
	} else {
		r0 = ret.Get(0).(controllerapi.Controller_ReceiveSpoofedProbesClient)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, ...grpc.CallOption) error); ok {
		r1 = rf(ctx, opts...)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
