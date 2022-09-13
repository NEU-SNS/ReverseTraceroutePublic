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

package datamodel

import (
	"fmt"
	"strconv"

	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/golang/protobuf/proto"
)

// CUnmarshal is the unmarshal for data coming from the cache
func (p *Ping) CUnmarshal(data []byte) error {
	return proto.Unmarshal(data, p)
}

// CMarshal marshals a ping for the cache
func (p *Ping) CMarshal() []byte {
	ret, err := proto.Marshal(p)
	if err != nil {
		return nil
	}
	return ret
}

// Key gets the key for a PM
func (pm *PingMeasurement) Key(site string) string {
	
	if pm.RR {
		if pm.Spoof{
			sa, _ := util.IPStringToInt32(pm.SAddr) // sa is the address spoofed 
			return fmt.Sprintf("%s_%s_%d_%d", "XRRP", site, sa, pm.Dst)
		} else {
			return fmt.Sprintf("%s_%d_%d_%d", "XRRP", 0, pm.Src, pm.Dst)
		}
			
	}
	
	// No cache for timestamp as we do not know what was put in the TS packet

	// } else if pm.TimeStamp != "" {
	// 	if pm.SpooferAddr != 0 {
	// 		sa, _ := util.IPStringToInt32(pm.SAddr) // sa is the address spoofed 
	// 		site, ok := sitePerVP[pm.Src]
	// 		if !ok {
	// 			site = strconv.FormatUint(uint64(pm.Src), 10)
	// 		}
	// 		return fmt.Sprintf("%s_%s_%d_%d", "XTSP", site, pm.Dst, sa)	
	// 	} else  {
	// 		return fmt.Sprintf("%s_%d_%d_%d", "XTSP", 0, pm.Dst, pm.Src)
	// 	}
	
	
	// Not sure we want to cache regular pings
	return ""
	// return fmt.Sprintf("%s_%d_%d", "XXXP", pm.Dst, pm.Src)
}

func stringIn(ar []string, in string) bool {
	for _, a := range ar {
		if a == in {
			return true
		}
	}
	return false
}

// Key gets the key for a Ping
func (p *Ping) KeyPerSite(sitePerVP map[uint32]string) string {
	site := ""
	if p.SpoofedFrom == 0 {
		// Not spoofed so site is 0
		site= "0"

	} else {
		siteValue, ok := sitePerVP[p.SpoofedFrom]
		site = siteValue
		if !ok {
			site = strconv.FormatUint(uint64(p.SpoofedFrom), 10)
		}
	}
	// if stringIn(p.Flags, "tsandaddr") {
	// 	return fmt.Sprintf("%s_%s_%d_%d", "XTSP", site, p.Dst, p.Src)
	// }
	if stringIn(p.Flags, "v4rr") {
		return fmt.Sprintf("%s_%s_%d_%d", "XRRP", site, p.Src, p.Dst)	
	}
	return fmt.Sprintf("%s_%d_%d_%d", "XXXP", p.SpoofedFrom, p.Dst, p.Src)
}