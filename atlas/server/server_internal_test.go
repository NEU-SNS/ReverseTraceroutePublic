/*
 Copyright (c) 2015, Northeastern University
 All rights reserved.

 Redistribution and use in source and binary forms, with or without
 modification, are permitted provided that the following conditions are met:
     * Redistributions of source code must retain the above copyright
       notice, this list of conditions and the following disclaimer.
     * Redistributions in binary form must reproduce the above copyright
       notice, this list of conditions and the following disclaimer in the
       documentation and/or other materials provided with the distribution.
     * Neither the name of the Northeastern University nor the
       names of its contributors may be used to endorse or promote products
       derived from this software without specific prior written permission.

 THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
 ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
 WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
 DISCLAIMED. IN NO EVENT SHALL Northeastern University BE LIABLE FOR ANY
 DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
 (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
 LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
 ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
 SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
*/

package server

import (
	"sort"
	"testing"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
)

func uint32SliceEqual(l, r []uint32) bool {
	if len(l) != len(r) {
		return false
	}
	sort.Sort(UInt32Slice(l))
	sort.Sort(UInt32Slice(r))
	for i, li := range l {
		if li != r[i] {
			return false
		}
	}
	return true
}

func TestRunningTrace_TryAdd(t *testing.T) {
	var tests = []struct {
		desc  string
		dst   uint32
		setup [][]uint32
		test  []uint32
		res   []uint32
	}{
		{
			desc:  "Add Overlapping Addresses",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{2, 4, 5},
			res:   []uint32{4, 5},
		},
		{
			desc:  "Add Overlapping Addresses out of order",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 5, 7, 99, 3, 18}}, test: []uint32{2, 5, 7, 18, 4, 15, 105},
			res: []uint32{2, 4, 15, 105},
		},
	}
	for _, test := range tests {
		rt := newRunningTraces()
		for _, ts := range test.setup {
			rt.TryAdd(test.dst, ts)
		}
		res := rt.TryAdd(test.dst, test.test)
		if !uint32SliceEqual(res, test.res) {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, res, test.res)
		}
	}
}

func TestRunningTrace_Remove(t *testing.T) {
	var tests = []struct {
		desc  string
		dst   uint32
		setup [][]uint32
		test  []uint32
		res   []uint32
	}{
		{
			desc:  "Remove addresses",
			dst:   0,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{1, 2, 3},
			res:   nil,
		},
		{
			desc:  "Remove partial addresses",
			dst:   600,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{1, 3},
			res:   []uint32{2},
		},
		{
			desc:  "Remove not present addresses",
			dst:   999,
			setup: [][]uint32{[]uint32{1, 2, 3}},
			test:  []uint32{4, 5},
			res:   []uint32{1, 2, 3},
		},
	}
	for _, test := range tests {
		rt := newRunningTraces()
		for _, ts := range test.setup {
			added := rt.TryAdd(test.dst, ts)
			if !uint32SliceEqual(added, ts) {
				t.Fatalf("%s: failed to add sources got: %v, expected: %v", test.desc, added, test.res)
			}
		}
		rt.Remove(test.dst, test.test)
		check, _ := rt.Check(test.dst)
		if !uint32SliceEqual(check, test.res) {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, check, test.res)
		}
	}
}

type mockCache struct {
	cache map[interface{}]interface{}
}

func (mc *mockCache) Get(key interface{}) (interface{}, bool) {
	if mc.cache == nil {
		mc.cache = make(map[interface{}]interface{})
	}
	val, ok := mc.cache[key]
	return val, ok
}

func (mc *mockCache) Add(key interface{}, val interface{}) bool {
	if mc.cache == nil {
		mc.cache = make(map[interface{}]interface{})
	}
	mc.cache[key] = val
	return false
}

func (mc *mockCache) Remove(key interface{}) {
	if mc.cache == nil {
		mc.cache = make(map[interface{}]interface{})
	}
	delete(mc.cache, key)
}

func TestTokenCache_Add(t *testing.T) {
	var tests = []struct {
		desc string
		add  *pb.IntersectionRequest
	}{
		{
			desc: "Add IR",
			add: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
		},
	}
	for _, test := range tests {
		tc := newTokenCache(&mockCache{})
		id, _ := tc.Add(test.add)
		ir, _ := tc.Get(id)
		if ir.Address != test.add.Address {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.add)
		}
	}
}

func TestTokenCache_Get(t *testing.T) {
	var tests = []struct {
		desc   string
		add    *pb.IntersectionRequest
		expect *pb.IntersectionRequest
	}{
		{
			desc: "Get IR",
			add: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
			expect: &pb.IntersectionRequest{
				Address:      1,
				Dest:         2,
				Staleness:    30,
				UseAliases:   true,
				IgnoreSource: true,
				Src:          566,
			},
		},
		{
			desc:   "Get IR Nil",
			add:    nil,
			expect: nil,
		},
	}
	for _, test := range tests {
		tc := newTokenCache(&mockCache{})
		id, _ := tc.Add(test.add)
		ir, _ := tc.Get(id)
		if ir == nil {
			if ir != test.expect {
				t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.expect)
			}
			continue
		}
		if ir.Address != test.expect.Address {
			t.Fatalf("%s: got: %v, expected: %v", test.desc, ir, test.expect)
		}
	}
}
