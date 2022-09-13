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

	"github.com/NEU-SNS/ReverseTraceroute/warts"
	"gonum.org/v1/gonum/stat"
)

// ConvertPing converts a warts ping to a DM ping
func ConvertPing(in warts.Ping) Ping {
	p := Ping{}
	p.Src = uint32(in.Flags.Src.Address)
	p.Dst = uint32(in.Flags.Dst.Address)
	p.Type = in.Type
	p.Method = in.Flags.PingMethod.String()
	dmt := &Time{}
	dmt.Sec = in.Flags.StartTime.Sec
	dmt.Usec = int64(in.Flags.StartTime.Usec)
	p.Start = dmt
	p.PingSent = uint32(in.Flags.ProbeCount)
	p.ProbeSize = uint32(in.Flags.ProbeSize)
	p.UserId = in.Flags.UserID
	p.Ttl = uint32(in.Flags.ProbeTTL)
	p.Wait = uint32(in.Flags.ProbeWaitS)
	p.Timeout = uint32(in.Flags.ProbeTimeout)
	p.Flags = in.Flags.PF.Strings()
	replies := make([]*PingResponse, in.ReplyCount)
	for i, resp := range in.PingReplies {
		rep := &PingResponse{}
		rep.From = uint32(resp.Addr.Address)
		rep.Seq = uint32(resp.ProbeID)
		rep.ReplySize = uint32(resp.ReplySize)
		rep.ReplyTtl = uint32(resp.ReplyTTL)
		rep.ReplyProto = resp.ReplyProto.String()
		txt := &Time{}
		txt.Sec = resp.Tx.Sec
		txt.Usec = int64(resp.Tx.Usec)
		rep.Tx = txt
		rxt := &Time{}
		rxt.Sec = txt.Sec + resp.RTT.Sec
		rxt.Usec = txt.Usec + int64(resp.RTT.Usec)
		rep.Rx = rxt
		rep.Rtt = uint32(resp.RTT.Sec*1000000 + int64(resp.RTT.Usec))
		rep.ProbeIpid = uint32(resp.ProbeIPID)
		rep.ReplyIpid = uint32(resp.ReplyIPID)
		rep.IcmpType = uint32((resp.ICMP & 0xFF00) >> 8)
		rep.IcmpCode = uint32(resp.ICMP & 0x00FF)
		if in.IsICMPTimestamp() {
			p.Flags = append(p.Flags, "icmptimestamp")
			rep.IcmpTs = new(ICMPTs)
			rep.IcmpTs.OriginateTimestamp = resp.TSReply.OTimestamp
			rep.IcmpTs.ReceiveTimestamp = resp.TSReply.RTimestamp
			rep.IcmpTs.TransmitTimestamp = resp.TSReply.TTimestamp
		}
		if in.IsTsOnly() {
			rep.Tsonly = make([]uint32, 0)
			for _, ts := range resp.V4TS.TimeStamps {
				rep.Tsonly = append(rep.Tsonly, uint32(ts))
			}
		} else if in.IsTsAndAddr() {
			p.Flags = append(p.Flags, "tsandaddr")
			rep.Tsandaddr = make([]*TsAndAddr, 0)
			expectedTsAndAddrs := in.Flags.TS
			for i, ts := range resp.V4TS.TimeStamps {
				tsa := &TsAndAddr{}
				tsa.Ip = uint32(resp.V4TS.Addrs[i].Address)
				tsa.Ts = uint32(ts)
				rep.Tsandaddr = append(rep.Tsandaddr, tsa)
			}
			for i := len(resp.V4TS.TimeStamps); i < len(expectedTsAndAddrs); i++{
				rep.Tsandaddr = append(rep.Tsandaddr, &TsAndAddr{Ip: uint32(expectedTsAndAddrs[i].Address)} )
			}
		}
		for _, addr := range resp.V4RR.Addrs {
			rep.RR = append(rep.RR, uint32(addr.Address))
		}
		replies[i] = rep
	}
	p.Responses = replies
	stat := in.GetStats()
	pstats := &PingStats{}
	pstats.Loss = float32(stat.Loss)
	pstats.Max = stat.Max
	pstats.Min = stat.Min
	pstats.Avg = stat.Avg
	pstats.Stddev = stat.StdDev
	pstats.Replies = int32(stat.Replies)
	p.Statistics = pstats

	return p
}

type RIPEMeasurementPingResult struct {
	Src  string `json:"src_addr"`      
	Dst  string `json:"dst_addr"`
	Proto string `json:"proto"`             
	ProbeSize int `json:"size"`
	Start int `json:"timestamp"`
	ProbeCount int `json:"sent"`
	Ttl int `json:"ttl"` // TTL of the first reply received
	Replies []RIPEPingReplies `json:"result"`    
	Min float32   `json:"min"`
	Max float32   `json:"max"`
	Avg float32   `json:"average"`
	Received int `json:"rcvd"`
	// [{
	// 	"fw": 5020,
	// 	"mver": "2.2.1",
	// 	"lts": 21,
	// 	"dst_name": "8.8.8.8",
	// 	"af": 4,
	// 	"dst_addr": "8.8.8.8",
	// 	"src_addr": "132.227.123.3",
	// 	"proto": "ICMP",
	// 	"ttl": 116,
	// 	"size": 48,
	// 	"result": [{
	// 		"rtt": 1.460254
	// 	}, {
	// 		"rtt": 1.402824
	// 	}, {
	// 		"rtt": 1.396073
	// 	}, {
	// 		"rtt": 1.438949
	// 	}, {
	// 		"rtt": 1.398119
	// 	}],
	// 	"dup": 0,
	// 	"rcvd": 5,
	// 	"sent": 5,
	// 	"min": 1.396073,
	// 	"max": 1.460254,
	// 	"avg": 1.4192438,
	// 	"msm_id": 28093538,
	// 	"prb_id": 6278,
	// 	"timestamp": 1604938252,
	// 	"msm_name": "Ping",
	// 	"from": "132.227.123.3",
	// 	"type": "ping",
	// 	"group_id": 28093538,
	// 	"step": null,
	// 	"stored_timestamp": 1604938757
	// }]
}
type RIPEPingReplies struct {
	Src *string `json:"src_Addr,omitempty"`
	Rtt float32 `json:"rtt,omitempty"`
	Ttl *int `json:"ttl,omitempty"`
	// In case of timeout, above fields are omitted
	Timeout *string `json:"x,omitempty"`
	// In case of error, above fields are omitted
	Error *string `json:"error,omitempty"`
}

func ConvertRIPEPing(ripePing RIPEMeasurementPingResult) (Ping){

	// Comments are not available fields in RIPE results
	p := Ping{}
	p.Src = uint32(binary.BigEndian.Uint32(net.ParseIP(ripePing.Src).To4()))
	p.Dst = uint32(binary.BigEndian.Uint32(net.ParseIP(ripePing.Dst).To4()))
	// p.Type = in.Type
	// p.Method = in.Flags.PingMethod.String()
	dmt := &Time{}
	dmt.Sec = int64(ripePing.Start)
	dmt.Usec = int64(0)
	p.Start = dmt
	p.PingSent = uint32(ripePing.ProbeCount)
	p.ProbeSize = uint32(ripePing.ProbeSize)
	// Retrieve this field later
	p.UserId = 0
	// Unavailable
	p.Ttl = 0
	// Unavailable
	// p.Wait = uint32(in.Flags.ProbeWaitS)
	// Unvailable, guess is 5 seconds from RIPE doc?
	p.Timeout = uint32(5)
	// p.Flags = in.Flags.PF.Strings()

	rtts := []float64{}
	replies := []*PingResponse{}
	for _, resp := range ripePing.Replies {
		if resp.Timeout != nil || resp.Error != nil{
			continue	
		}
		rep := &PingResponse{}
		if resp.Src != nil{
			rep.From = 	uint32(binary.BigEndian.Uint32(net.ParseIP(*resp.Src).To4()))
		} else {
			rep.From = uint32(p.Src)
		}
		if resp.Ttl != nil{
			rep.ReplyTtl = 	uint32(*resp.Ttl)
		} else {
			rep.ReplyTtl = 	uint32(ripePing.Ttl)
		}
		// Unvailable
		// rep.Seq = uint32(resp.ProbeID)
		// rep.ReplySize = uint32(resp.ReplySize)
		rep.ReplyProto = ripePing.Proto
		txt := &Time{}
		txt.Sec = 0
		txt.Usec = 0
		rep.Tx = txt
		// Unavailable
		rxt := &Time{}
		rxt.Sec = 0
		rxt.Usec = 0
		rep.Rx = rxt
		rep.Rtt = uint32(resp.Rtt*1000)
		rtts = append(rtts, float64(resp.Rtt))
		// Unavailable
		// rep.ProbeIpid = uint32(resp.ProbeIPID)
		// rep.ReplyIpid = uint32(resp.ReplyIPID)
		// Unavailable 
		// rep.IcmpType = uint32((resp.ICMP & 0xFF00) >> 8)
		// rep.IcmpCode = uint32(resp.ICMP & 0x00FF)

		// No IP options in RIPE probes
		// if in.IsTsOnly() {
		// 	rep.Tsonly = make([]uint32, 0)
		// 	for _, ts := range resp.V4TS.TimeStamps {
		// 		rep.Tsonly = append(rep.Tsonly, uint32(ts))
		// 	}
		// } else if in.IsTsAndAddr() {
		// 	p.Flags = append(p.Flags, "tsandaddr")
		// 	rep.Tsandaddr = make([]*TsAndAddr, 0)
		// 	for i, ts := range resp.V4TS.TimeStamps {
		// 		tsa := &TsAndAddr{}
		// 		tsa.Ip = uint32(resp.V4TS.Addrs[i].Address)
		// 		tsa.Ts = uint32(ts)
		// 		rep.Tsandaddr = append(rep.Tsandaddr, tsa)
		// 	}
		// }
		// for _, addr := range resp.V4RR.Addrs {
		// 	rep.RR = append(rep.RR, uint32(addr.Address))
		// }
		replies = append(replies, rep)
	}
	p.Responses = replies
	pstats := &PingStats{}
	pstats.Loss = 1 - float32(ripePing.Received) / float32(ripePing.ProbeCount)
	pstats.Max = ripePing.Max
	pstats.Min = ripePing.Min
	pstats.Avg = ripePing.Avg
	pstats.Stddev = float32(stat.StdDev(rtts, nil))
	pstats.Replies = int32(ripePing.Received)
	p.Statistics = pstats
	return p
}