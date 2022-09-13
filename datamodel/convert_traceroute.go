// +build !386

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
	"encoding/binary"
	"net"
	"time"

	"github.com/NEU-SNS/ReverseTraceroute/util"
	"github.com/NEU-SNS/ReverseTraceroute/warts"
)

// ConvertTraceroute converts a warts Traceroute to a datamodel Traceroute
func ConvertTraceroute(in warts.Traceroute) Traceroute {
	t := Traceroute{}
	t.Platform = "mlab"
	t.Type = "trace"
	t.UserId = in.Flags.UserID
	t.Src = uint32(in.Flags.Src.Address)
	t.Dst = uint32(in.Flags.Dst.Address)
	t.Method = in.Flags.TraceType.String()
	t.Sport = uint32(in.Flags.SourcePort)
	t.Dport = uint32(in.Flags.DestPort)
	t.StopReason = in.Flags.StopReason.String()
	t.StopData = uint32(in.Flags.StopData)
	tt := TracerouteTime{}
	tt.Sec = in.Flags.StartTime.Sec
	tt.Usec = int64(in.Flags.StartTime.Usec)
	tt.Ftime = time.Unix(tt.Sec, tt.Usec*1000).Format(`"` + traceTime + `"`)
	t.Start = &tt
	t.HopCount = uint32(in.HopCount)
	t.Attempts = uint32(in.Flags.Attempts)
	t.Hoplimit = uint32(in.Flags.HopLimit)
	t.Firsthop = uint32(in.Flags.StartTTL)
	t.Wait = uint32(in.Flags.TimeoutS)
	t.WaitProbe = uint32(in.Flags.MinWaitCenti)
	t.Tos = uint32(in.Flags.IPToS)
	t.ProbeSize = uint32(in.Flags.ProbeSize)
	hops := convertHops(in)
	t.Hops = hops
	t.GapLimit = uint32(in.Flags.GapLimit)
	return t
}

func convertHops(in warts.Traceroute) []*TracerouteHop {
	hops := in.Hops
	retHops := make([]*TracerouteHop, in.HopCount)
	for i, hop := range hops {
		h := &TracerouteHop{}
		h.Addr = uint32(hop.Address.Address)
		h.ProbeTtl = uint32(hop.ProbeTTL)
		h.ProbeId = uint32(hop.ProbeID)
		h.ProbeSize = uint32(hop.ProbeSize)
		h.Rtt = &RTT{}
		h.Rtt.Sec = hop.RTT.Sec
		h.Rtt.Usec = int64(hop.RTT.Usec)
		h.ReplyTtl = uint32(hop.ReplyTTL)
		h.ReplyTos = uint32(hop.ToS)
		h.ReplySize = uint32(hop.ReplySize)
		h.ReplyIpid = uint32(hop.IPID)
		h.IcmpType = uint32((hop.ICMPTypeCode & 0xFF00) >> 8)
		h.IcmpCode = uint32(hop.ICMPTypeCode & 0x00FF)
		h.IcmpQTtl = uint32(hop.QuotedTTL)
		h.IcmpQIpl = uint32(hop.QuotedIPLength)
		h.IcmpQTos = uint32(hop.QuotesToS)
		h.IcmpExt = convertICMPExt(hop.ICMPExt)
		retHops[i] = h
	}
	return retHops
}

func convertICMPExt(in warts.ICMPExtensionList) *ICMPExtensions {
	icmpExtensions := ICMPExtensions {
		Length: uint32(in.Length),
		IcmpExtensionList: []*ICMPExtension{},
	}
	icmpExtensions.Length = uint32(in.Length)
	for _, icmpExtension := range(in.Extensions) {
		icmpExtensions.IcmpExtensionList = append(icmpExtensions.IcmpExtensionList, &ICMPExtension{
			Length: uint32(icmpExtension.Length),
			ClassNumber: uint32(icmpExtension.ClassNumber),
			TypeNumber: uint32(icmpExtension.TypeNumber),
			Data: icmpExtension.Data,
		})
	} 

	return &icmpExtensions

}


type RIPEMeasurementTracerouteResult struct {
	Src  string `json:"from"`      
	Dst  string `json:"dst_addr"`
	Proto string `json:"proto"`             
	ProbeSize int `json:"size"`
	ParisID int `json:"paris_id"`
	Tos int `json:"tos"`
	Start uint32 `json:"timestamp"`
	End uint32 `json:"endtime"`

	RepliesByHop []RIPETracerouteReplies `json:"result"`    

	

	// [{
	// 	"msm_id": 28100368,
	// "prb_id": 6278,
	// "timestamp": 1604965202,
	// "msm_name": "Traceroute",
	// "from": "132.227.123.3",
	// "type": "traceroute",
	// "group_id": 28100368,
	// "stored_timestamp": 1604965220
	// 	"fw": 5020,
	// 	"mver": "2.2.1",
	// 	"lts": 36,
	// 	"endtime": 1604965215,
	// 	"dst_name": "8.8.8.8",
	// 	"dst_addr": "8.8.8.8",
	// 	"src_addr": "132.227.123.3",
	// 	"proto": "ICMP",
	// 	"af": 4,
	// 	"size": 48,
	// 	"paris_id": 1,
	// 	"result": [{
	// 		"hop": 1,
	// 		"result": [{
	// 			"x": "*"
	// 		}, {
	// 			"x": "*"
	// 		}, {
	// 			"x": "*"
	// 		}]
	// 	}, {
	// 		"hop": 2,
	// 		"result": [{
	// 			"from": "134.157.167.125",
	// 			"ttl": 254,
	// 			"size": 28,
	// 			"rtt": 0.492
	// 		}, {
	// 			"from": "134.157.167.125",
	// 			"ttl": 254,
	// 			"size": 28,
	// 			"rtt": 0.465
	// 		}, {
	// 			"from": "134.157.167.125",
	// 			"ttl": 254,
	// 			"size": 28,
	// 			"rtt": 0.769
	// 		}]
	// 	}, {
	// 		"hop": 3,
	// 		"result": [{
	// 			"from": "134.157.254.124",
	// 			"ttl": 253,
	// 			"size": 28,
	// 			"rtt": 0.655
	// 		}, {
	// 			"from": "134.157.254.124",
	// 			"ttl": 253,
	// 			"size": 28,
	// 			"rtt": 0.608
	// 		}, {
	// 			"from": "134.157.254.124",
	// 			"ttl": 253,
	// 			"size": 28,
	// 			"rtt": 0.738
	// 		}]
	// 	}, {
	
}

type RIPETracerouteReplies struct {
	Hop int `json:"hop,omitempty"`
	Result []RIPETracerouteReply `json:"result,omitempty"`
}

type RIPETracerouteReply struct {
	From string `json:"from,omitempty"`
	Ttl int `json:"ttl,omitempty"`
	Size int `json:"size,omitempty"`
	Rtt float32 `json:"rtt,omitempty"`
	Itos int `json:"itos,omitempty"`
	Err* string `json:"err,omitempty"` // ICMP error type (unreachable, ...)
	ICMPExt *RIPETracerouteReplyICMPExt `json:"icmpext,omitempty"` 
	// In case of timeout, above fields are omitted
	Timeout *string `json:"x,omitempty"`
	// In case of error, above fields are omitted
	Error *string `json:"error,omitempty"`
}

type RIPETracerouteReplyICMPExt struct {
	Version int `json:"version,omitempty"`
	RFC4884 int `json:"rfc4884,omitempty"`
	Obj     []RIPETracerouteReplyICMPExtObj `json:"obj,omitempty"`
	// "icmpext": {
		// 				"version": 2,
		// 				"rfc4884": 0,
		// 				"obj": [{
		// 					"class": 1,
		// 					"type": 1,
		// 					"mpls": [{
		// 						"label": 6072,
		// 						"exp": 0,
		// 						"s": 1,
		// 						"ttl": 1
		// 					}]
		// 				}]
		// 			}
}

type RIPETracerouteReplyICMPExtObj struct {
	Class int `json:"class,omitempty"`
	Type  int `json:"type,omitempty"`
	MPLS  []RIPETracerouteReplyICMPExtObjMPLS `json:"mpls,omitempty"`
}

type RIPETracerouteReplyICMPExtObjMPLS struct {
	Label int `json:"label,omitempty"`
	Exp   int `json:"exp,omitempty"`
	S     int `json:"s,omitempty"`
	Ttl   int `json:"ttl,omitempty"`
}

// ConvertRIPETraceroute converts a RIPE Traceroute to a datamodel Traceroute
func ConvertRIPETraceroute(ripeTraceroute RIPEMeasurementTracerouteResult) Traceroute {
	t := Traceroute{}
	t.Platform = "ripe"
	t.Type = "trace"
	// t.UserId = in.Flags.UserID
	src, _ := util.IPStringToInt32(ripeTraceroute.Src)
	if src == 0 {
		print()
	}
	t.Src = uint32(binary.BigEndian.Uint32(net.ParseIP(ripeTraceroute.Src).To4()))
	t.Dst = uint32(binary.BigEndian.Uint32(net.ParseIP(ripeTraceroute.Dst).To4()))
	// t.Method = in.Flags.TraceType.String()
	t.Sport = uint32(ripeTraceroute.ParisID)
	t.Dport = uint32(ripeTraceroute.ParisID)
	// Unavailable
	// t.StopReason = in.Flags.StopReason.String()
	// t.StopData = uint32(in.Flags.StopData)
	tt := TracerouteTime{}
	tt.Sec = int64(ripeTraceroute.Start)
	tt.Usec = int64(0)
	tt.Ftime = time.Unix(tt.Sec, tt.Usec*1000).Format(`"` + traceTime + `"`)
	t.Start = &tt
	
	// Guess based on RIPE Atlas API documentation
	t.Attempts = 3 // Default in RIPE Atlas
	t.Hoplimit = 30 // Default in RIPE Atlas
	t.Firsthop = 1 // Default in RIPE Atlas

	// t.Wait = uint32(in.Flags.TimeoutS)
	// t.WaitProbe = uint32(in.Flags.MinWaitCenti)
	t.Tos = uint32(ripeTraceroute.Tos)
	t.ProbeSize = uint32(ripeTraceroute.ProbeSize)
	hops, maxTTL:= convertRIPEHops(t.Src, ripeTraceroute)
	t.Hops = hops
	// t.GapLimit = uint32(in.Flags.GapLimit)

	// Set this later
	t.HopCount = uint32(maxTTL)

	return t
}

func convertRIPEHops(src uint32, in RIPEMeasurementTracerouteResult) ([]*TracerouteHop, int) {
	repliesByHops := in.RepliesByHop
	retHops := []*TracerouteHop{}
	// Set the source as TTLL 0 to be able to intersect it in the Atlas
	retHops = append(retHops, &TracerouteHop{
		Addr: src,
	})
	maxTTL := 0
	for _, repliesHop := range repliesByHops {
		probeTTL := repliesHop.Hop
		if probeTTL > maxTTL{
			maxTTL = probeTTL
		}
		replies := repliesHop.Result
		for _, reply := range(replies){
			h := &TracerouteHop{}
			if  reply.Timeout != nil{
				continue
			}
			if reply.Error != nil {
				continue
			}
			if reply.From == "" {
				continue
			}
			// if len(net.ParseIP(reply.From).To4()) == 0 {
			// 	continue
			// }
			h.Addr = uint32(uint32(binary.BigEndian.Uint32(net.ParseIP(reply.From).To4())))
			h.ProbeTtl = uint32(probeTTL)
			// h.ProbeId = uint32(hop.ProbeID)
			h.ProbeSize = uint32(in.ProbeSize)
			h.Rtt = &RTT{}
			h.Rtt.Sec = int64(reply.Rtt / 1000)
			// Get the "partie entière" of the RTT in s
			toSubstract := int64(reply.Rtt / 1000) 
			h.Rtt.Usec =  int64(reply.Rtt * 1000) - (toSubstract * 1000000)
			h.ReplyTtl = uint32(reply.Ttl)
			h.ReplyTos = uint32(reply.Itos)
			h.ReplySize = uint32(reply.Size)
			// h.ReplyIpid = uint32(hop.IPID)
			// h.IcmpType = uint32((hop.ICMPTypeCode & 0xFF00) >> 8)
			// h.IcmpCode = uint32(hop.ICMPTypeCode & 0x00FF)
			// h.IcmpQTtl = uint32(hop.QuotedTTL)
			// h.IcmpQIpl = uint32(hop.QuotedIPLength)
			// h.IcmpQTos = uint32(hop.QuotesToS)
			if reply.ICMPExt != nil {
				h.IcmpExt = convertRIPEICMPExt(reply.ICMPExt)
			}
			retHops = append(retHops, h)
		}
		
	}
	return retHops, maxTTL
}


func convertRIPEICMPExt(in *RIPETracerouteReplyICMPExt) *ICMPExtensions {
	icmpExtensions := ICMPExtensions {
		IcmpExtensionList: []*ICMPExtension{},
	}
	// icmpExtensions.Length = uint32(in.Length)
	for _, icmpExtension := range(in.Obj) {
		icmpExtensions.IcmpExtensionList = append(icmpExtensions.IcmpExtensionList, &ICMPExtension{
			// Length: uint32(icmpExtension.Length),
			ClassNumber: uint32(icmpExtension.Class),
			TypeNumber: uint32(icmpExtension.Type),
			// TODO do something with labels
			// Data: icmpExtension.Data,
		})
	} 

	return &icmpExtensions

}