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

package server_test

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/atlas/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/pb"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/repo"
	"github.com/NEU-SNS/ReverseTraceroute/atlas/server"
	cmocks "github.com/NEU-SNS/ReverseTraceroute/controller/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/util"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/test"
	vpmocks "github.com/NEU-SNS/ReverseTraceroute/vpservice/mocks"
	vppb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var now = time.Now()
var date string = fmt.Sprintf("'%s'", now.Format("2006-01-02 15:04:05"))

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

func TestGetPathsWithToken(t *testing.T) {
	trsm := &mocks.TRStore{}
	trsm.On("FindIntersectingTraceroute",
		mock.AnythingOfType("types.IntersectionQuery")).Return(nil, repo.ErrNoIntFound)
	trsm.On("GetAtlasSources",
		mock.AnythingOfType("uint32"), mock.AnythingOfType("time.Duration")).Return([]uint32{}, nil)
	clm := &cmocks.Client{}
	vpsm := &vpmocks.VPSource{}
	vpsm.On("GetVPs").Return(&vppb.VPReturn{}, nil)
	var opts []server.Option
	opts = append(opts, server.WithClient(clm),
		server.WithTRS(trsm), server.WithVPS(vpsm),
		server.WithCache(&mockCache{}))

	serv := server.NewServer(opts...)
	load := []*pb.IntersectionRequest{
		&pb.IntersectionRequest{
			Address: 9,
			Dest:    10,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 0,
			Dest:    1,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 11,
			Dest:    15,
			Src:     2,
		},
		&pb.IntersectionRequest{
			Address: 19,
			Dest:    20,
			Src:     2,
		},
	}
	var responses []*pb.IntersectionResponse
	for _, l := range load {
		res, err := serv.GetIntersectingPath(l)
		if err != nil {
			t.Fatal("Failed to load requests ", err)
		}
		if res.Type != pb.IResponseType_TOKEN {
			t.Fatalf("GetIntersectingPath(%v) expected token response got(%v)", l, res)
		}
		responses = append(responses, res)
	}
	// This is ugly, but wait so that other goroutines can run
	<-time.After(time.Second * 2)
	for _, resp := range responses {
		r, err := serv.GetPathsWithToken(&pb.TokenRequest{
			Token: resp.Token,
		})
		if err != nil {
			t.Fatalf("Failed to get path with token(%v) got err %v", resp.Token, err)
		}
		if r.Token != resp.Token {
			t.Fatalf("Expected Token[%v], Got Token[%v]", resp.Token, r.Token)
		}
		if r.Type != pb.IResponseType_NONE_FOUND {
			t.Fatalf("Expected[%v], Got[%v]", pb.IResponseType_NONE_FOUND, r.Type)
		}
	}
}

func TestGetPathsWithTokenInvalidToken(t *testing.T) {
	trsm := &mocks.TRStore{}
	trsm.On("FindIntersectingTraceroute",
		mock.AnythingOfType("types.IntersectionQuery")).Return(nil, repo.ErrNoIntFound)
	clm := &cmocks.Client{}
	vpsm := &vpmocks.VPSource{}
	var opts []server.Option
	opts = append(opts, server.WithClient(clm),
		server.WithTRS(trsm), server.WithVPS(vpsm),
		server.WithCache(&mockCache{}))
	serv := server.NewServer(opts...)
	tokenReq := &pb.TokenRequest{
		Token: 99999,
	}
	r, err := serv.GetPathsWithToken(tokenReq)
	if err != nil {
		t.Fatalf("Expected nil error received resp[%v], err[%v]", r, err)
	}
	if r.Type != pb.IResponseType_ERROR || r.Token != tokenReq.Token {
		t.Fatalf("Unexpected response resp[%v], err[%v]", r, err)
	}
}

func TestGetIntersectingPathStaleness(t *testing.T) {
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	id := test.InsertTestTraceroute(1, 2, "mlab", date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	// Just refresh the cache
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})
	
	// Check staleness
	st, err := atlasClient.GetIntersectingPath(context.Background())
	ir := pb.IntersectionRequest {
		Address: 3093903169,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 15,
		Platforms: []string{"mlab"},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_ERROR)
		// This means no intersection was found, that is expected
		break
	}
}

func TestGetIntersectingPathPlatform(t *testing.T) {
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()

	fmt.Printf("%s\n", date)
	fmt.Printf("%s\n", now)

	id := test.InsertTestTraceroute(1, 2, "mlab", date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	// Just refresh the cache
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})

	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check platform 
	ir := pb.IntersectionRequest {
		Address: 3093903169,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 15,
		Platforms: []string{"dummy"}, // No platforms
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_TOKEN)
		// This means no intersection was found, that is expected
		break
	}
}

func TestGetIntersectingPathTrHop(t *testing.T) {

	// Check ignore source
	// ir = pb.IntersectionRequest {
	// 	Address: 3093903169,
	// 	Dest: 71826316,
	// 	UseAliases: true,
	// 	IgnoreSource: true,
	// 	IgnoreSourceAs: false,
	// 	Src: 1,
	// 	Staleness: 43800 * 12,  // a year
	// }
	// err = st.Send(&ir)
	// if err != nil {
	// 	panic(err)
	// }

	// for {
	// 	itr, err := st.Recv()
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	if err != nil {
	// 		log.Error(err)
	// 		panic(err)
	// 	}
	// 	require.EqualValues(t, itr.Type, pb.IResponseType_TOKEN)
	// 	// This means no intersection was found, that is expected
	// 	break
	// }
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	id := test.InsertTestTraceroute(1, 2, platform,  date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})
	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3093903169,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 1440,  // 1 day
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		
		
		str    := now.Format(environment.DateFormat)
		trTime, err := time.Parse(time.RFC3339, str)
		trTimestamp := trTime.Unix()
		require.EqualValues(t, itr.Timestamp, trTimestamp)
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		require.EqualValues(t, itr.TracerouteId, id)
		require.EqualValues(t, itr.DistanceToSource, 4) // Distance to source
		// Intersection was found, now look at what is in the path and check that it returns expected path
		expectedHops := [] uint32 {3093903169, 3093905493, 3093902558, 71683070, 2}
		for i, hop := range(itr.Path.Hops)  {
			require.EqualValues(t, hop.Ip, expectedHops[i])
			require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_EXACT)
		}
		break
	}
}

func TestGetIntersectingPathTrHopAlias(t *testing.T) {
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	alias := uint32(10)
	test.DeleteTestAlias(db, 0)
	test.InsertTestAlias(db, 3, alias)
	defer test.DeleteTestAlias(db, 0)
	id := test.InsertTestTraceroute(alias, 2, platform, date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})
	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800 *  12,  // 1 month * 12
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		str    := now.Format(environment.DateFormat)
		trTime, err := time.Parse(time.RFC3339, str)
		trTimestamp := trTime.Unix()
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		require.EqualValues(t, itr.Timestamp, trTimestamp)
		require.EqualValues(t, itr.TracerouteId, id)
		require.EqualValues(t, itr.Path.Address, alias)
		require.EqualValues(t, itr.DistanceToSource, 5) // Distance to source
		// Intersection was found, now look at what is in the path and check that it returns expected path
		expectedHops := [] uint32 {alias, 3093903169, 3093905493, 3093902558, 71683070, 2}
		for i, hop := range(itr.Path.Hops)  {
			require.EqualValues(t, hop.Ip, expectedHops[i])
			require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_EXACT)
		}
		break
	}
}


func TestGetIntersectingPathPlatformRRHop(t *testing.T) {

	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "test"
	id := test.InsertTestTraceroute(1, 2, platform, date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})
	st, err := atlasClient.GetIntersectingPath(context.Background())
	// Check good intersection rr hop
	ir := pb.IntersectionRequest {
		Address: 71677608,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800 * 12,  // 1 month * 12
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		require.EqualValues(t, itr.TracerouteId, id)
		str    := now.Format(environment.DateFormat)
		trTime, err := time.Parse(time.RFC3339, str)
		trTimestamp := trTime.Unix()
		require.EqualValues(t, itr.Timestamp, trTimestamp)
		require.EqualValues(t, itr.DistanceToSource, 3) // Distance to source
		// Intersection was found, now look at what is in the path and check that it returns expected path
		expectedHops := [] uint32 {71677608, 1,3093903169, 3093905493, 3093902558, 71683070, 2}
		for i, hop := range(itr.Path.Hops)  {
			require.EqualValues(t, hop.Ip, expectedHops[i])
			if 0 <= i  && i <= 5 { // 5 because tr hop 0-4
				require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_BETWEEN)
			} else  {
				require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_EXACT)
			}
			
		}
		break
	}

}

func TestGetIntersectingPathPlatformRRHopNotAllowed(t *testing.T) {

	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	defer db.Close()
	id := test.InsertTestTraceroute(1, 2, "mlab", date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})

	st, err := atlasClient.GetIntersectingPath(context.Background())
	// Check good intersection rr hop
	ir := pb.IntersectionRequest {
		Address: 71677608,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: false,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{"mlab"},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_TOKEN)
		break
	}

}

func TestGetIntersectingPathPlatformTrHopPlatformOrder(t *testing.T) {

	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	defer db.Close()
	test.DeleteTestTracerouteByLabel(db, "test")
	test.DeleteTestTracerouteByLabel(db, "test2")
	id2 :=  test.InsertTestTraceroute(1, 2, "test2", date, db)
	id := test.InsertTestTraceroute(1, 2, "test", date, db)
	id3 := test.InsertTestTraceroute(1, 2, "test", date, db)
	defer test.DeleteTestTraceroute(db, id)
	defer test.DeleteTestTraceroute(db, id2)
	defer test.DeleteTestTraceroute(db, id3)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})

	st, err := atlasClient.GetIntersectingPath(context.Background())
	// Check good intersection rr hop
	ir := pb.IntersectionRequest {
		Address: 3093903169,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: false,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{"test", "test2"},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		str    := now.Format(environment.DateFormat)
		trTime, err := time.Parse(time.RFC3339, str)
		trTimestamp := trTime.Unix()
		require.EqualValues(t, itr.Timestamp, trTimestamp)
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		require.EqualValues(t, itr.Platform, "test") // Order by platform
		require.EqualValues(t, itr.TracerouteId, id) // order by ID
		
		// Intersection was found, now look at what is in the path and check that it returns expected path
		expectedHops := [] uint32 {3093903169, 3093905493, 3093902558, 71683070, 2}
		for i, hop := range(itr.Path.Hops)  {
			require.EqualValues(t, hop.Ip, expectedHops[i])
			require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_EXACT)
		}
		break
	}

}

func TestGetIntersectingPathRRHopOrderByPlatform(t *testing.T) {

	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	defer db.Close()
	test.DeleteTestTracerouteByLabel(db, "test")
	test.DeleteTestTracerouteByLabel(db, "test2")
	id2 :=  test.InsertTestTraceroute(1, 2, "test2", date, db)
	id := test.InsertTestTraceroute(1, 2, "test", date, db)
	id3 := test.InsertTestTraceroute(1, 2, "test", date, db)
	defer test.DeleteTestTraceroute(db, id)
	defer test.DeleteTestTraceroute(db, id2)
	defer test.DeleteTestTraceroute(db, id3)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})

	st, err := atlasClient.GetIntersectingPath(context.Background())
	// Check good intersection rr hop
	ir := pb.IntersectionRequest {
		Address: 71677608,
		Dest: 2,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{"test", "test2"},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		require.EqualValues(t, itr.TracerouteId, id)
		str    := now.Format(environment.DateFormat)
		trTime, err := time.Parse(time.RFC3339, str)
		trTimestamp := trTime.Unix()
		require.EqualValues(t, itr.Timestamp, trTimestamp)
		require.EqualValues(t, itr.Platform, "test")
		require.EqualValues(t, itr.TracerouteId, id)
		// Intersection was found, now look at what is in the path and check that it returns expected path
		expectedHops := [] uint32 {71677608, 1,3093903169, 3093905493, 3093902558, 71683070, 2}
		for i, hop := range(itr.Path.Hops)  {
			require.EqualValues(t, hop.Ip, expectedHops[i])
			if 0 <= i  && i <= 5 { // 5 because tr hop 0-4
				require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_BETWEEN)
			} else  {
				require.EqualValues(t, hop.IntersectHopType, pb.IntersectHopType_EXACT)
			}
			
		}
		break
	}

}

func TestMarkTracerouteStale(t *testing.T) {
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	id := test.InsertTestTraceroute(1, 2, "mlab", date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	
	// Mark traceroute as stale
	_, err = atlasClient.MarkTracerouteStale(context.Background(), &pb.MarkTracerouteStaleRequest{OldTracerouteId: id})
	if err != nil {
		panic(err)
	}
	// Check that it is stale 
	staleCheckQuery := fmt.Sprintf("SELECT stale from atlas_traceroutes where id=%d", id)
	rows, err := db.Query(staleCheckQuery)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		stale := 0
		err := rows.Scan(&stale)
		if err != nil {
			panic(err)
		}
		require.EqualValues(t, stale, 1)
	}	

}

func TestMarkTracerouteStaleSource(t *testing.T) {
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	source := uint32(2)
	id := test.InsertTestTraceroute(1, source, "mlab", date, db) 
	defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	atlasClient.InsertTraceroutes(context.Background(), &pb.InsertTraceroutesRequest{})

	
	// Mark traceroute as stale
	_, err = atlasClient.MarkTracerouteStaleSource(context.Background(), 
	&pb.MarkTracerouteStaleSourceRequest{Source: source})
	if err != nil {
		panic(err)
	}
	// Check that it is stale 
	staleCheckQuery := fmt.Sprintf("SELECT stale from atlas_traceroutes where id=%d", id)
	rows, err := db.Query(staleCheckQuery)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		stale := 0
		err := rows.Scan(&stale)
		if err != nil {
			panic(err)
		}
		require.EqualValues(t, stale, 1)
	}	

}

func TestConnectionScalability(t *testing.T) {
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	defer db.Close()

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	sources := util.IPsFromFile("../../rankingservice/algorithms/evaluation/resources/sources.csv") 
	var wg sync.WaitGroup
	for i := 0 ; i < 10000; i++ {
		if i % 100 == 0 {
			log.Infof("Done querying %d of itr", i)
		}
		wg.Add(1)
		time.Sleep(100 * time.Nanosecond)
		// time.Sleep(30 * time.Millisecond)
		go func(i int) {
			defer wg.Done()
			st, err := atlasClient.GetIntersectingPath(context.Background())
			// Check good intersection rr hop
			sourceIndex := rand.Intn(len(sources))
			source, _ := util.IPStringToInt32(sources[sourceIndex])
			ir := pb.IntersectionRequest {
				Address: 71677608,
				Dest: source,
				UseAliases: true,
				UseAtlasRr: true,
				IgnoreSource: true,
				IgnoreSourceAs: false,
				Src: 1,
				Staleness: 43800,  // 1 month
				Platforms: []string{"ripe"},
			}
			err = st.Send(&ir)
			if err != nil {
				panic(err)
			}

			for {
				_, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error(err)
					panic(err)
				}
				if i % 100 == 0 {
					log.Infof("Processed ir %d", i)
				}
				
				break
			}
			err = st.CloseSend()
			if err != nil {
				panic(err)
			}
		} (i)
		
	}
	wg.Wait()	
}



func TestGetIntersectingPathMLabRefresh(t *testing.T) {
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "mlab"
	// id := test.InsertTestTraceroute(1, 68067148, platform, db) 
	// defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3179144268,
		Dest: 68067161,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	err = st.CloseSend()

	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
	}
}

func TestGetIntersectingPathRIPERefresh(t *testing.T) {
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "ripe"

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3586023470,
		Dest: 68067161,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		StalenessBeforeRefresh: 1, 
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}
	err = st.CloseSend()
	for {
		itr, err := st.Recv()
		if err == io.EOF {
			panic(err)
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		break
	}
}


func TestGetIntersectingPathMLabDoubleRefresh(t *testing.T) {
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "mlab"
	// id := test.InsertTestTraceroute(1, 68067148, platform, db) 
	// defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3179144268,
		Dest: 68067161,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{platform},
	}
	var wg sync.WaitGroup
	for i := 0 ; i < 12; i++ {
		wg.Add(1)
		go func (wait int ) {
			defer wg.Done()
			st, err := atlasClient.GetIntersectingPath(context.Background())
			time.Sleep(time.Duration(wait) * time.Second)
			err = st.Send(&ir)
			if err != nil {
				panic(err)
			}
			err = st.CloseSend()
			for {
				itr, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error(err)
					panic(err)
				}
				require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
				log.Infof("%d request Intersected traceroute %d", wait, itr.TracerouteId)
			}
			} (i)
		
	}
	wg.Wait()
	
}

func TestGetIntersectingPathMLabRefreshbackgroundEvaluation(t *testing.T) {
	
	// Run the atlas before running this test.
	db := test.GetMySQLDB("../../cmd/atlas/atlas.config", "traceroute_atlas")
	defer db.Close()
	platform := "mlab"
	// id := test.InsertTestTraceroute(1, 68067148, platform, db) 
	// defer test.DeleteTestTraceroute(db, id)

	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	st, err := atlasClient.GetIntersectingPath(context.Background())

	// Check good intersection traceroute hop
	ir := pb.IntersectionRequest {
		Address: 3179144268,
		Dest: 68067161,
		UseAliases: true,
		UseAtlasRr: true,
		IgnoreSource: false,
		IgnoreSourceAs: false,
		Src: 1,
		Staleness: 43800,  // 1 month
		Platforms: []string{platform},
	}
	err = st.Send(&ir)
	if err != nil {
		panic(err)
	}

	err = st.CloseSend()
	source := uint32(0)
	for {
		itr, err := st.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			panic(err)
		}
		require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
		source = itr.Src
	}
	time.Sleep(15 * time.Second)
	defer test.DeleteTestTracerouteByLabel(db, "staleness_evaluation_mlab")
	// Select the traceroute 
	query := fmt.Sprintf(" SELECT date FROM atlas_traceroutes WHERE dest=? AND src=? and platform='staleness_evaluation_mlab' ORDER BY date desc LIMIT 1")
	res, err := db.Query(query, 68067161, source)
	defer res.Close()
	if err != nil {
		panic(err)
	}
	for res.Next() {
		var timestamp time.Time
		err := res.Scan(&timestamp)
		if err != nil {
			panic(err)
		}
		now := time.Now()
		require.True(t, now.Sub(timestamp) < 30 * time.Second)
	}
}


func TestRunTracerouteAtlas(t *testing.T) {

	source := "195.113.161.14" // ple2.cesnet.cz
	sourceI, err := util.IPStringToInt32(source)
	
	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	_, err = atlasClient.RunTracerouteAtlasToSource(context.Background(), &pb.RunTracerouteAtlasToSourceRequest{
		Source: sourceI,
		WithRipe: true,
	})

	assert.Equal(t, err, nil)
}


func TestTracerouteAtlasLoadQueries(t *testing.T) {
	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}

	destinations := util.IPsFromFile("/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/algorithms/evaluation/resources/bgp_survey_destinations.csv") 
	sources := util.IPsFromFile("/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/rankingservice/algorithms/evaluation/resources/sources.csv") 


	nRevtrLoad := 100000
	wg := sync.WaitGroup{}
	now := time.Now()
	for i := 0; i < nRevtrLoad ; i++ {
		// fmt.Printf("Sent %d requests\n", i)
		wg.Add(1)
		// time.Sleep(1 *time.Millisecond)
		go func() {
			defer wg.Done()
			st, err := atlasClient.GetIntersectingPath(context.Background())
			
			sourceIndex := rand.Intn(len(sources))
			destIndex := rand.Intn(len(destinations))

			source, _ := util.IPStringToInt32(sources[sourceIndex])
			hop, _ := util.IPStringToInt32(destinations[destIndex])
			

			// Check good intersection traceroute hop
			ir := pb.IntersectionRequest {
				Address: source,
				Dest: hop,
				UseAliases: true,
				UseAtlasRr: true,
				IgnoreSource: false,
				IgnoreSourceAs: false,
				Src: 1,
				Staleness: 43800,  // 1 month
				Platforms: []string{"ripe", "mlab"},
			}
			err = st.Send(&ir)
			if err != nil {
				panic(err)
			}

			err = st.CloseSend()
			// source := uint32(0)
			for {
				_, err := st.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Error(err)
					panic(err)
				}
				require.EqualValues(t, err, nil)
				// require.EqualValues(t, itr.Type, pb.IResponseType_PATH)
				// source = itr.Src
			}
			// time.Sleep(15 * time.Second)
	

		} ()
	}
	wg.Wait()
	elapsed := time.Now().Sub(now)

	log.Infof("Took %s seconds to run %d queries", elapsed, nRevtrLoad)

}


func TestTracerouteAtlasLoadHopsInMemory(t *testing.T) {
	atlasClient, err := survey.CreateAtlasClient("../../cmd/atlas/root.crt")
	if err != nil {
		panic(err)
	}
	
	now := time.Now()
	res, err := atlasClient.GetAvailableHopAtlasPerSource(context.Background(), &pb.GetAvailableHopAtlasPerSourceRequest{})
	elapsed := time.Now().Sub(now)
	log.Infof("Took %s seconds to load atlas", elapsed)
	for source, hops := range res.HopsPerSource{
		log.Infof("Source %d, %d hops", source, len(hops.Hops))
	}

}