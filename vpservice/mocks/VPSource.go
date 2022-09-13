package mocks

import "github.com/stretchr/testify/mock"

import "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"

type VPSource struct {
	mock.Mock
}

// GetVPs provides a mock function with given fields:
func (_m *VPSource) GetVPs() (*pb.VPReturn, error) {
	ret := _m.Called()

	var r0 *pb.VPReturn
	if rf, ok := ret.Get(0).(func() *pb.VPReturn); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.VPReturn)
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

// GetOneVPPerSite provides a mock function with given fields:
func (_m *VPSource) GetOneVPPerSite() (*pb.VPReturn, error) {
	ret := _m.Called()

	var r0 *pb.VPReturn
	if rf, ok := ret.Get(0).(func() *pb.VPReturn); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*pb.VPReturn)
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

// GetRRSpoofers provides a mock function with given fields: addr, max
func (_m *VPSource) GetRRSpoofers(addr uint32, max uint32) ([]*pb.VantagePoint, error) {
	ret := _m.Called(addr, max)

	var r0 []*pb.VantagePoint
	if rf, ok := ret.Get(0).(func(uint32, uint32) []*pb.VantagePoint); ok {
		r0 = rf(addr, max)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.VantagePoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, uint32) error); ok {
		r1 = rf(addr, max)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetTSSpoofers provides a mock function with given fields: max
func (_m *VPSource) GetTSSpoofers(max uint32) ([]*pb.VantagePoint, error) {
	ret := _m.Called(max)

	var r0 []*pb.VantagePoint
	if rf, ok := ret.Get(0).(func(uint32) []*pb.VantagePoint); ok {
		r0 = rf(max)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*pb.VantagePoint)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(max)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
