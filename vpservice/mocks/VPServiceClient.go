package mocks

import "github.com/stretchr/testify/mock"

import datamodel "github.com/NEU-SNS/ReverseTraceroute/datamodel"
import context "golang.org/x/net/context"
import grpc "google.golang.org/grpc"

type VPServiceClient struct {
	mock.Mock
}

// GetVPs provides a mock function with given fields: ctx, in, opts
func (_m *VPServiceClient) GetVPs(ctx context.Context, in *datamodel.VPRequest, opts ...grpc.CallOption) (*datamodel.VPReturn, error) {
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
