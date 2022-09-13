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

//Package plcontroller is the library for creating a planet-lab controller
package plcontroller

import (
	"fmt"

	plc "github.com/NEU-SNS/ReverseTraceroute/controller/pb"
	dm "github.com/NEU-SNS/ReverseTraceroute/datamodel"
	env "github.com/NEU-SNS/ReverseTraceroute/environment"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/log"
	con "golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// ControllerSender Is a Sender which sends to the central controller received spoofs
// In order to being able to receive on both debug controller and not break the spoofing capabilities
// in production, send spoofed packets on both debug and production when used in debug
type ControllerSender struct {
	RootCA string
	conn   *grpc.ClientConn
	debugConn *grpc.ClientConn
}

// Send satisfies the Sender interface for a ControllerSender
func (cs *ControllerSender) Send(sps []*dm.Probe, addr uint32) error {

	// Lazy initialization
	if cs.conn == nil {
		// WARNING, hack here. Please read
		// We have only one plcontroller that can run (because nodes only connect to scamper port 4381, which 
		// can be run by one process), so if docker plcontroller is in debug mode on walter, send both the spoofed
		// packets to the debug and the production controller.
		// This allows production healthcheck measurements to be sent to production controller 
		// (while debug will just not be sent).
		
		// Connect to production controller
		ip, _ := util.Int32ToIPString(addr)
		saddr := fmt.Sprintf("%s:%d", ip, env.ControllerPortProduction)
		creds, err := credentials.NewClientTLSFromFile(cs.RootCA, "controller.revtr.ccs.neu.edu")
		if err != nil {
			return err
		}
		cc, err := grpc.Dial(saddr, grpc.WithTransportCredentials(creds))
		if err != nil {
			return err
		}
		cs.conn = cc
	}
	if cs.debugConn == nil {
		// Connect to debug controller
		// ip, _ := util.Int32ToIPString(addr)
		saddr := fmt.Sprintf("localhost:%d", env.ControllerPortDebug)
		// creds, err := credentials.NewClientTLSFromFile(cs.RootCA, "controller.revtr.ccs.neu.edu")
		cc, err := grpc.Dial(saddr, grpc.WithInsecure())
		if err != nil {
			cs.debugConn = nil
			return err
		}
		cs.debugConn = cc
	}

	err := cs.forwardSpoofed(sps, cs.conn)
	if err != nil {
		return err
	}

	err = cs.forwardSpoofed(sps, cs.debugConn)
	if err != nil {
		// We can not return the error as it might be that the debug controller (which is run when we debug)
		// is not present, so just log the error
		log.Error(err)
	}
	return nil
}

func (cs *ControllerSender) forwardSpoofed(sps []*dm.Probe, conn *grpc.ClientConn) error {
	cl := plc.NewControllerClient(conn)
	ctx, cancel := con.WithCancel(con.Background())
	defer cancel()
	stream, err := cl.ReceiveSpoofedProbes(ctx)
	if err != nil {
		return err
	}
	for _, sp := range sps {
		if err := stream.Send(sp); err != nil {
			return err
		}
	}
	_, err = stream.CloseAndRecv()
	return err
}

// Close closes the connection the sender uses
func (cs *ControllerSender) Close() error {
	if cs.conn != nil {
		return cs.conn.Close()
	}
	return nil
}
