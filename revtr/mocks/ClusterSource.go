package mocks

import "github.com/stretchr/testify/mock"

type ClusterSource struct {
	mock.Mock
}

// GetClusterIDByIP provides a mock function with given fields: _a0
func (_m *ClusterSource) GetClusterIDByIP(_a0 uint32) (int, error) {
	ret := _m.Called(_a0)

	var r0 int
	if rf, ok := ret.Get(0).(func(uint32) int); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Get(0).(int)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetIPsForClusterID provides a mock function with given fields: _a0
func (_m *ClusterSource) GetIPsForClusterID(_a0 int) ([]uint32, error) {
	ret := _m.Called(_a0)

	var r0 []uint32
	if rf, ok := ret.Get(0).(func(int) []uint32); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]uint32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
