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

package plcontroller

import (
	"testing"

	"golang.org/x/net/context"

	"github.com/NEU-SNS/ReverseTraceroute/datamodel"
	"github.com/NEU-SNS/ReverseTraceroute/plcontroller/mocks"
	smock "github.com/NEU-SNS/ReverseTraceroute/spoofmap/mocks"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	mmock "github.com/stretchr/testify/mock"
)

var (
	addr          = "localhost"
	scamperport   = "45454"
	port          = 44000
	sockdir       = "/tmp/scamper_sockets"
	binpath       = "/usr/local/bin/sc_remoted"
	converterpath = "/usr/local/bin/sc_warts2json"
	pprofaddr     = "localhost:45454"
	certfile      = "../doc/certs/test.crt"
	keyfile       = "../doc/certs/test.key"
)

var conf = Config{
	Local: LocalConfig{
		Addr:      &addr,
		PProfAddr: &pprofaddr,
		CertFile:  &certfile,
		KeyFile:   &keyfile,
		Port:      &port,
	},
	Scamper: ScamperConfig{
		Port:          &scamperport,
		SockDir:       &sockdir,
		BinPath:       &binpath,
		ConverterPath: &converterpath,
	},
}

func TestRecSpoof(t *testing.T) {
	defer util.LeakCheck(t)()
	cl := &mocks.Client{}
	sm := &smock.SpoofMap{}
	var plc = &PlController{}
	plc.client = cl
	plc.spoofs = sm
	spoof := &datamodel.Spoof{
		Ip:  1111111,
		Dst: 222222222,
		Id:  10,
		Sip: 10101010,
	}
	dummy := &datamodel.PingMeasurement{
		Src:     spoof.Sip,
		Dst:     spoof.Dst,
		Count:   "1",
		Timeout: 2,
		Ttl:     "1",
		Sport:   "61681",
		Dport:   "62195",
	}

	cl.On("DoMeasurement", mmock.AnythingOfType("string"), dummy).Return(nil, uint32(0), nil)
	cl.On("RemoveMeasurement", mmock.AnythingOfType("string"),
		mmock.AnythingOfType("uint32")).Return(nil)
	sm.On("Register", *spoof).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := plc.recSpoof(ctx, spoof)
	if err != nil {
		t.Fatalf("plc.recSpoof(%v), got[%v], expected[<nil>]", spoof, err)
	}
	cl.AssertNumberOfCalls(t, "DoMeasurement", 1)
	sm.AssertNumberOfCalls(t, "Register", 1)
	sm.AssertExpectations(t)
}
