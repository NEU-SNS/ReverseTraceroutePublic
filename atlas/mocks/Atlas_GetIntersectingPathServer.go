package mocks

import "github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
import "github.com/stretchr/testify/mock"

type Atlas_GetIntersectingPathServer struct {
	mock.Mock
}

// Send provides a mock function with given fields: _a0
func (_m *Atlas_GetIntersectingPathServer) Send(_a0 *pb.IntersectionResponse) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(*pb.IntersectionResponse) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Recv provides a mock function with given fields:
func (_m *Atlas_GetIntersectingPathServer) Recv() (*pb.IntersectionRequest, error) {
	ret := _m.Called()

	var r0 *pb.IntersectionRequest
	if rf, ok := ret.Get(0).(func() *pb.IntersectionRequest); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.IntersectionRequest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
