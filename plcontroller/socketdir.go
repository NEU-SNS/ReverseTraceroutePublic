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
	"context"
	"fmt"
	"net"
	"path"
	"strings"

	"github.com/NEU-SNS/ReverseTraceroute/log"
	"github.com/NEU-SNS/ReverseTraceroute/scamper"
	"github.com/NEU-SNS/ReverseTraceroute/util"
	vpspb "github.com/NEU-SNS/ReverseTraceroute/vpservice/pb"
	"github.com/NEU-SNS/ReverseTraceroute/watcher"
)

const (
	mailSubjectReverseTracerouteDeployment = "Reverse Traceroute deployment of your source"
)

func (c *PlController) handlEvents() {
	c.clearAllVps()
	for {
		event, err := c.w.GetEvent(c.shutdown)
		if err == watcher.ErrWatcherClosed {
			return
		}
		if err != nil {
			log.Error(err)
			continue
		}
		switch event.Type() {
		case watcher.Create:

			// Hack to debug what's happening on one socket
			// if !strings.Contains(event.Name(), "195.113.161.14") {
			// 	continue
			// }

			log.Infof("Create socket: %s", event.Name())
			// time.Sleep(1 *time.Second)
			con, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: event.Name(), Net: "unix"})
			if err != nil {
				log.Error(err)
				continue
			}
			s, err := scamper.NewSocket(
				event.Name(),
				con)
			if err != nil {
				log.Error(err)
				continue
			}
			ip, err := util.IPStringToInt32(s.IP())
			if err != nil {
				log.Errorf("Failed to convert socket IP: %v", err)
				s.Stop()
				continue
			}
			// Get the owner of the VP
			owner, err := c.db.GetVPOwner(ip)
			
			if err != nil {
				log.Errorf("Failed to insert new vantage point %d, %v", ip, err)
				s.Stop()
				continue
			}

			c.client.AddSocket(s)

			ipArg := ip
			// go func(ipArg uint32) {
					
				ipS, _ := util.Int32ToIPString(ipArg)
				if owner.Name == "mlab" {
					// M-Lab nodes are trusted
					
				} else {
					continue
					// Check if the owner of the VP has provided any information
					if owner.Name == "" || owner.Email == "" {
						// The IP address is not known so do not insert the VP into DB
						log.Infof("Trying to add an unknown source %s", ipS)
						s.Stop()
						c.client.RemoveSocket(ipS)
						return 
					}
					err = c.db.InsertVP(ipArg, c.ip)
					if err != nil {
						log.Errorf("Failed to insert new vantage point %d, %v", ipArg, err)
						s.Stop()
						c.client.RemoveSocket(ipS)
						return 
					}
					// Test vantage points capabilities if it's not an M-Lab VP.
					resp, err := c.vpsclient.TestNewVPCapabilities(context.Background(), &vpspb.TestNewVPCapabilitiesRequest{
						Addr: ipArg,
						Hostname: owner.Hostname,
						Site: owner.Site,
					})
	
					if err != nil {
						log.Errorf("Failed to test new vantage point capabilities %d, %v", ipArg, err)
						s.Stop()
						c.client.RemoveSocket(ipS)
						return 
					}
	
					if !resp.CanReverseTraceroutes {
						// Tell the owner of the source that the vantage point can not receive revtr
						// Send an email to the owner to tell the state of his vp
						err = SendEmail([] string {owner.Email}, mailSubjectReverseTracerouteDeployment,
							fmt.Sprintf("Your vantage point %s can not receive reverse traceroutes, RR probes were dropped on the path", ipS))	
						if owner.Email != "kevinmylinh.vermeulen@gmail.com" {
							// Hack to debug and keep the socket open
							s.Stop()
							c.client.RemoveSocket(ipS)
						}
						return 
					} else {
						err = SendEmail([] string {owner.Email}, mailSubjectReverseTracerouteDeployment, 
							fmt.Sprintf("Your vantage point %s can receive reverse traceroutes. Please wait few minutes while we set up your vantage point.", ipS))	
					}
					
					// Launch the atlas 
					// _, err = c.atclient.RunTracerouteAtlasToSource(context.Background(), &atpb.RunTracerouteAtlasToSourceRequest{
					// 	Source: ip,
					// 	WithRipe: true,
					// 	RipeAccount: "",
					// 	RipeKey: "",
					// })

					if err != nil {
						// Send an email to the owner to tell the state of his vp
						err = SendEmail([] string {owner.Email}, mailSubjectReverseTracerouteDeployment, fmt.Sprintf("There was a problem for running traceroute atlas to your source %s.", ipS))
					} else {
						// Send an email to the owner to tell the state of his vp
						err = SendEmail([] string {owner.Email}, mailSubjectReverseTracerouteDeployment, fmt.Sprintf("Your vantage point %s is ready ", ipS))
					}
	
					
				}
				
				err = c.db.UpdateController(ipArg, c.ip, c.ip)
				if err != nil {
					log.Errorf("Failed to update controller  %v", err)
					s.Stop()
					c.client.RemoveSocket(ipS)
					return 
				}
			
				vpsConnected.Add(1)
			// } (ip)

			
			 

		case watcher.Remove:
			log.Infof("Remove socket: %s", event.Name())
			ipPort := strings.Split(path.Base(event.Name()), ":")
			ip := ipPort[0]
			port := ipPort[1] 
			nip, err := util.IPStringToInt32(ip)
			if err != nil {
				log.Errorf("Failed to convert socket IP: %v", err)
				continue
			}
			
			sock, err := c.client.GetSocket(ip)
			if err != scamper.ErrorSocketNotFound && sock.Port() == port {
				c.client.RemoveSocket(ip)	
				sock.Stop()
			}

			
			
			err = c.db.UpdateController(nip, 0, c.ip)
			if err != nil {
				log.Errorf("Failed to update controller  %v", err)
			}
			
			vpsConnected.Sub(1)
		}
	}
}

func (c *PlController) clearAllVps() {
	err := c.db.ClearAllVPs()
	if err != nil {
		log.Error(err)
	}
}

//This is only for use when a server is going down
func (c *PlController) removeAllVps() {
	log.Debug("Removing all VPS")
	if c.client == nil {
		return
	}
	vps := c.client.GetAllSockets()
	log.Debug(vps)
	for sock := range vps {
		log.Debugf("Removing %v", sock.IP())
		ip, err := util.IPStringToInt32(sock.IP())
		if err != nil {
			log.Error(err)
			continue
		}
		err = c.db.UpdateController(ip, 0, c.ip)
		if err != nil {
			log.Error(err)
		}
	}
}
