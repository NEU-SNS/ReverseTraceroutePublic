package mocks

import "github.com/stretchr/testify/mock"

import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"

type AtlasServer struct {
	mock.Mock
}

// GetIntersectingPath provides a mock function with given fields: _a0
func (_m *AtlasServer) GetIntersectingPath(_a0 *pb.IntersectionRequest) (*pb.IntersectionResponse, error) {
	ret := _m.Called(_a0)

	var r0 *pb.IntersectionResponse
	if rf, ok := ret.Get(0).(func(*pb.IntersectionRequest) *pb.IntersectionResponse); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.IntersectionResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*pb.IntersectionRequest) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetPathsWithToken provides a mock function with given fields: _a0
func (_m *AtlasServer) GetPathsWithToken(_a0 *pb.TokenRequest) (*pb.TokenResponse, error) {
	ret := _m.Called(_a0)

	var r0 *pb.TokenResponse
	if rf, ok := ret.Get(0).(func(*pb.TokenRequest) *pb.TokenResponse); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.TokenResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*pb.TokenRequest) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
