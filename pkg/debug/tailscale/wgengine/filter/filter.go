// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filter

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang/groupcache/lru"
	"tailscale.com/ratelimit"
	"tailscale.com/wgengine/packet"
)

type Filter struct {
	matches Matches

	udpMu  sync.Mutex
	udplru *lru.Cache
}

type Response int

const (
	Drop Response = iota
	Accept
	noVerdict // Returned from subfilters to continue processing.
)

func (r Response) String() string {
	switch r {
	case Drop:
		return "Drop"
	case Accept:
		return "Accept"
	case noVerdict:
		return "noVerdict"
	default:
		return "???"
	}
}

type RunFlags int

const (
	LogDrops RunFlags = 1 << iota
	LogAccepts
	HexdumpDrops
	HexdumpAccepts
)

type tuple struct {
	SrcIP   IP
	DstIP   IP
	SrcPort uint16
	DstPort uint16
}

const LRU_MAX = 512 // max entries in UDP LRU cache

var MatchAllowAll = Matches{
	Match{[]IPPortRange{IPPortRangeAny}, []IP{IPAny}},
}

func NewAllowAll() *Filter {
	return New(MatchAllowAll)
}

func NewAllowNone() *Filter {
	return New(nil)
}

func New(matches Matches) *Filter {
	f := &Filter{
		matches: matches,
		udplru:  lru.New(LRU_MAX),
	}
	return f
}

func maybeHexdump(flag RunFlags, b []byte) string {
	if flag != 0 {
		return packet.Hexdump(b) + "\n"
	} else {
		return ""
	}
}

// TODO(apenwarr): use a bigger bucket for specifically TCP SYN accept logging?
//   Logging is a quick way to record every newly opened TCP connection, but
//   we have to be cautious about flooding the logs vs letting people use
//   flood protection to hide their traffic. We could use a rate limiter in
//   the actual *filter* for SYN accepts, perhaps.
var acceptBucket = ratelimit.Bucket{
	Burst:        3,
	FillInterval: 10 * time.Second,
}
var dropBucket = ratelimit.Bucket{
	Burst:        10,
	FillInterval: 5 * time.Second,
}

func logRateLimit(runflags RunFlags, b []byte, q *packet.QDecode, r Response, why string) {
	if r == Drop && (runflags&LogDrops) != 0 && dropBucket.TryGet() > 0 {
		var qs string
		if q == nil {
			qs = fmt.Sprintf("(%d bytes)", len(b))
		} else {
			qs = q.String()
		}
		log.Printf("Drop: %v %v %s\n%s", qs, len(b), why, maybeHexdump(runflags&HexdumpDrops, b))
	} else if r == Accept && (runflags&LogAccepts) != 0 && acceptBucket.TryGet() > 0 {
		log.Printf("Accept: %v %v %s\n%s", q, len(b), why, maybeHexdump(runflags&HexdumpAccepts, b))
	}
}

func (f *Filter) RunIn(b []byte, q *packet.QDecode, rf RunFlags) Response {
	r := pre(b, q, rf)
	if r == Accept || r == Drop {
		// already logged
		return r
	}

	r, why := f.runIn(q)
	logRateLimit(rf, b, q, r, why)
	return r
}

func (f *Filter) RunOut(b []byte, q *packet.QDecode, rf RunFlags) Response {
	r := pre(b, q, rf)
	if r == Drop || r == Accept {
		// already logged
		return r
	}
	r, why := f.runOut(q)
	logRateLimit(rf, b, q, r, why)
	return r
}

func (f *Filter) runIn(q *packet.QDecode) (r Response, why string) {
	switch q.IPProto {
	case packet.ICMP:
		// If any port is open to an IP, allow ICMP to it.
		if matchIPWithoutPorts(f.matches, q) {
			return Accept, "icmp ok"
		}
	case packet.TCP:
		// For TCP, we want to allow *outgoing* connections,
		// which means we want to allow return packets on those
		// connections. To make this restriction work, we need to
		// allow non-SYN packets (continuation of an existing session)
		// to arrive. This should be okay since a new incoming session
		// can't be initiated without first sending a SYN.
		// It happens to also be much faster.
		// TODO(apenwarr): Skip the rest of decoding in this path?
		if q.IPProto == packet.TCP && !q.IsTCPSyn() {
			return Accept, "tcp non-syn"
		}
		if matchIPPorts(f.matches, q) {
			return Accept, "tcp ok"
		}
	case packet.UDP:
		t := tuple{q.SrcIP, q.DstIP, q.SrcPort, q.DstPort}

		f.udpMu.Lock()
		_, ok := f.udplru.Get(t)
		f.udpMu.Unlock()

		if ok {
			return Accept, "udp cached"
		}
		if matchIPPorts(f.matches, q) {
			return Accept, "udp ok"
		}
	default:
		return Drop, "Unknown proto"
	}
	return Drop, "no rules matched"
}

func (f *Filter) runOut(q *packet.QDecode) (r Response, why string) {
	if q.IPProto == packet.UDP {
		t := tuple{q.DstIP, q.SrcIP, q.DstPort, q.SrcPort}

		f.udpMu.Lock()
		f.udplru.Add(t, t)
		f.udpMu.Unlock()
	}
	return Accept, "ok out"
}

func pre(b []byte, q *packet.QDecode, rf RunFlags) Response {
	if len(b) == 0 {
		// wireguard keepalive packet, always permit.
		return Accept
	}
	if len(b) < 20 {
		logRateLimit(rf, b, nil, Drop, "too short")
		return Drop
	}
	q.Decode(b)

	if q.IPProto == packet.Junk {
		// Junk packets are dangerous; always drop them.
		logRateLimit(rf, b, q, Drop, "junk!")
		return Drop
	} else if q.IPProto == packet.Fragment {
		// Fragments after the first always need to be passed through.
		// Very small fragments are considered Junk by QDecode.
		logRateLimit(rf, b, q, Accept, "fragment")
		return Accept
	}

	return noVerdict
}
