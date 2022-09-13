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
	"strings"
	"testing"

	survey "github.com/NEU-SNS/ReverseTraceroute/survey"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/stretchr/testify/assert"
)

const (
	rootCA = "/home/kevin/go/src/github.com/NEU-SNS/ReverseTraceroute/cmd/vpservice/root.crt"
)

func TestNewVPCapabilitiesTest(t *testing.T) {
	client, err := survey.CreateVPServiceClient(rootCA)
	if err != nil {
		panic(err)
	}

	source := uint32(3561608025)
	// sourceS, _ := util.Int32ToIPString(source)
	resp, err := client.TestNewVPCapabilities(context.Background(), &pb.TestNewVPCapabilitiesRequest{
		Addr: source,
	})
	assert.EqualValues(t, resp.CanReverseTraceroutes, true)
}

func TestNewVPCapabilitiesTestFail(t *testing.T) {
	client, err := survey.CreateVPServiceClient(rootCA)
	if err != nil {
		panic(err)
	}
	source := "8.8.8.8"
	sourceI, err := util.IPStringToInt32(source)
	// sourceS, _ := util.Int32ToIPString(source)
	resp, err := client.TestNewVPCapabilities(context.Background(), &pb.TestNewVPCapabilitiesRequest{
		Addr: sourceI,
	})
	if err != nil {
		panic(err)
	}
	assert.EqualValues(t, resp.CanReverseTraceroutes, false)
}


func TestNewVPCapabilitiesTestExternalSource(t *testing.T) {
	client, err := survey.CreateVPServiceClient(rootCA)
	if err != nil {
		panic(err)
	}
	source := "195.113.161.14" // ple2.cesnet.cz
	sourceI, err := util.IPStringToInt32(source)
	// sourceS, _ := util.Int32ToIPString(source)
	resp, err := client.TestNewVPCapabilities(context.Background(), &pb.TestNewVPCapabilitiesRequest{
		Addr: sourceI,
	})
	if err != nil {
		panic(err)
	}
	assert.EqualValues(t, resp.CanReverseTraceroutes, false)
}

func TestUnquarantine(t *testing.T) {

	client, err := survey.CreateVPServiceClient(rootCA)
	if err != nil {
		panic(err)
	}
	// Check which sources can RR 
	
	vpsForTesting := []*pb.VantagePoint{}
	vpsToTest := []*pb.VantagePoint{}
	VPResponse, err := client.GetVPs(context.Background(), &pb.VPRequest{})
	if err != nil {
		panic(err)
	}
	for _, vp := range(VPResponse.Vps[:]) {
		if strings.Contains(vp.Hostname, "sandbox") {
			// vpsForTesting = append(vpsForTesting, vp)
			// vpsToTest = append(vpsToTest, vp)
		} else if strings.Contains(vp.Hostname, "staging") {
			// vpsToTest = append(vpsToTest, vp)
			vpsForTesting = append(vpsForTesting, vp)
		} else if strings.Contains(vp.Hostname, "oti") {
			vpsToTest = append(vpsToTest, vp)
		}
	}

	

	// source := "195.113.161.14" // ple2.cesnet.cz
	// sourceI, err := util.IPStringToInt32(source)
	// testVPs = append(testVPs, &pb.VantagePoint{
	// 	Ip: sourceI,
	// 	Hostname: "ple2.cesnet.cz",
	// })
	
	vpsForTesting = vpsForTesting[:30]

	// Try unquarantine nodes
	client.TryUnquarantine(context.Background(), &pb.TryUnquarantineRequest{
		VantagePoints: vpsToTest[:],
		VantagePointsForTesting: vpsForTesting,
		IsSelfForTesting: false,
		IsTestOnlyActive: false,
	})
	

}

func TestUnquarantineAll(t *testing.T) {
	client, err := survey.CreateVPServiceClient(rootCA)
	if err != nil {
		panic(err)
	}
	resp, err := client.TryUnquarantine(context.Background(), &pb.TryUnquarantineRequest{
		IsTryAllVantagePoints: true,
		IsTestOnlyActive: false,
	})
	if err != nil {
		panic(err)
	}
	assert.True(t, resp != nil)
}