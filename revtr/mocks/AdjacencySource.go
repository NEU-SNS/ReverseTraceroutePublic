package mocks

import "github.com/NEU-SNS/ReverseTraceroute/revtr/types"
import "github.com/stretchr/testify/mock"

type AdjacencySource struct {
	mock.Mock
}

// GetAdjacenciesByIP1 provides a mock function with given fields: _a0
func (_m *AdjacencySource) GetAdjacenciesByIP1(_a0 uint32) ([]types.Adjacency, error) {
	ret := _m.Called(_a0)

	var r0 []types.Adjacency
	if rf, ok := ret.Get(0).(func(uint32) []types.Adjacency); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Adjacency)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAdjacenciesByIP2 provides a mock function with given fields: _a0
func (_m *AdjacencySource) GetAdjacenciesByIP2(_a0 uint32) ([]types.Adjacency, error) {
	ret := _m.Called(_a0)

	var r0 []types.Adjacency
	if rf, ok := ret.Get(0).(func(uint32) []types.Adjacency); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.Adjacency)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAdjacencyToDestByAddrAndDest24 provides a mock function with given fields: _a0, _a1
func (_m *AdjacencySource) GetAdjacencyToDestByAddrAndDest24(_a0 uint32, _a1 uint32) ([]types.AdjacencyToDest, error) {
	ret := _m.Called(_a0, _a1)

	var r0 []types.AdjacencyToDest
	if rf, ok := ret.Get(0).(func(uint32, uint32) []types.AdjacencyToDest); ok {
		r0 = rf(_a0, _a1)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]types.AdjacencyToDest)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(uint32, uint32) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
