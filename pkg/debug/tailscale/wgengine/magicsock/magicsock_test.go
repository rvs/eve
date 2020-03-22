// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package magicsock

import (
	"bytes"
	crand "crypto/rand"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tailscale/wireguard-go/device"
	"github.com/tailscale/wireguard-go/tun/tuntest"
	"github.com/tailscale/wireguard-go/wgcfg"
	"tailscale.com/derp"
	"tailscale.com/derp/derphttp"
	"tailscale.com/derp/derpmap"
	"tailscale.com/stun/stuntest"
	"tailscale.com/types/key"
	"tailscale.com/types/logger"
)

func TestListen(t *testing.T) {

	epCh := make(chan string, 16)
	epFunc := func(endpoints []string) {
		for _, ep := range endpoints {
			epCh <- ep
		}
	}

	stunAddr, stunCleanupFn := stuntest.Serve(t)
	defer stunCleanupFn()

	port := pickPort(t)
	conn, err := Listen(Options{
		Port:          port,
		DERPs:         derpmap.NewTestWorld(stunAddr),
		EndpointsFunc: epFunc,
		Logf:          t.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	go func() {
		var pkt [64 << 10]byte
		for {
			_, _, _, err := conn.ReceiveIPv4(pkt[:])
			if err != nil {
				return
			}
		}
	}()

	timeout := time.After(10 * time.Second)
	var endpoints []string
	suffix := fmt.Sprintf(":%d", port)
collectEndpoints:
	for {
		select {
		case ep := <-epCh:
			endpoints = append(endpoints, ep)
			if strings.HasSuffix(ep, suffix) {
				break collectEndpoints
			}
		case <-timeout:
			t.Fatalf("timeout with endpoints: %v", endpoints)
		}
	}
}

func pickPort(t *testing.T) uint16 {
	t.Helper()
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	return uint16(conn.LocalAddr().(*net.UDPAddr).Port)
}

func TestDerpIPConstant(t *testing.T) {
	if DerpMagicIP != derpMagicIP.String() {
		t.Errorf("str %q != IP %v", DerpMagicIP, derpMagicIP)
	}
	if len(derpMagicIP) != 4 {
		t.Errorf("derpMagicIP is len %d; want 4", len(derpMagicIP))
	}
}

func TestPickDERPFallback(t *testing.T) {
	c := &Conn{
		derps: derpmap.Prod(),
	}
	a := c.pickDERPFallback()
	if a == 0 {
		t.Fatalf("pickDERPFallback returned 0")
	}

	// Test that it's consistent.
	for i := 0; i < 50; i++ {
		b := c.pickDERPFallback()
		if a != b {
			t.Fatalf("got inconsistent %d vs %d values", a, b)
		}
	}

	// Test that that the pointer value of c is blended in and
	// distribution over nodes works.
	got := map[int]int{}
	for i := 0; i < 50; i++ {
		c = &Conn{derps: derpmap.Prod()}
		got[c.pickDERPFallback()]++
	}
	t.Logf("distribution: %v", got)
	if len(got) < 2 {
		t.Errorf("expected more than 1 node; got %v", got)
	}

	// Test that stickiness works.
	const someNode = 123456
	c.myDerp = someNode
	if got := c.pickDERPFallback(); got != someNode {
		t.Errorf("not sticky: got %v; want %v", got, someNode)
	}
}

func makeConfigs(t *testing.T, ports []uint16) []wgcfg.Config {
	t.Helper()

	var privKeys []wgcfg.PrivateKey
	var addresses [][]wgcfg.CIDR

	for i := range ports {
		privKey, err := wgcfg.NewPrivateKey()
		if err != nil {
			t.Fatal(err)
		}
		privKeys = append(privKeys, privKey)

		addresses = append(addresses, []wgcfg.CIDR{
			parseCIDR(t, fmt.Sprintf("1.0.0.%d/32", i+1)),
		})
	}

	var cfgs []wgcfg.Config
	for i, port := range ports {
		cfg := wgcfg.Config{
			Name:       fmt.Sprintf("peer%d", i+1),
			PrivateKey: privKeys[i],
			Addresses:  addresses[i],
			ListenPort: port,
		}
		for peerNum, port := range ports {
			if peerNum == i {
				continue
			}
			peer := wgcfg.Peer{
				PublicKey:  privKeys[peerNum].Public(),
				AllowedIPs: addresses[peerNum],
				Endpoints: []wgcfg.Endpoint{{
					Host: "127.0.0.1",
					Port: port,
				}},
				PersistentKeepalive: 25,
			}
			cfg.Peers = append(cfg.Peers, peer)
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs
}

func parseCIDR(t *testing.T, addr string) wgcfg.CIDR {
	t.Helper()
	cidr, err := wgcfg.ParseCIDR(addr)
	if err != nil {
		t.Fatal(err)
	}
	return cidr
}

func runDERP(t *testing.T) (s *derp.Server, addr string, cleanupFn func()) {
	var serverPrivateKey key.Private
	if _, err := crand.Read(serverPrivateKey[:]); err != nil {
		t.Fatal(err)
	}

	s = derp.NewServer(serverPrivateKey, t.Logf)

	httpsrv := httptest.NewUnstartedServer(derphttp.Handler(s))
	httpsrv.Config.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	httpsrv.StartTLS()
	t.Logf("DERP server URL: %s", httpsrv.URL)

	addr = strings.TrimPrefix(httpsrv.URL, "https://")
	cleanupFn = func() {
		httpsrv.CloseClientConnections()
		httpsrv.Close()
		s.Close()
	}

	return s, addr, cleanupFn
}

// devLogger returns a wireguard-go device.Logger that writes
// wireguard logs to the test logger.
func devLogger(t *testing.T, prefix string) *device.Logger {
	pfx := []interface{}{prefix}
	logf := func(format string, args ...interface{}) {
		t.Helper()
		t.Logf("%s: "+format, append(pfx, args...)...)
	}
	return &device.Logger{
		Debug: logger.StdLogger(logf),
		Info:  logger.StdLogger(logf),
		Error: logger.StdLogger(logf),
	}
}

// TestDeviceStartStop exercises the startup and shutdown logic of
// wireguard-go, which is intimately intertwined with magicsock's own
// lifecycle. We seem to be good at generating deadlocks here, so if
// this test fails you should suspect a deadlock somewhere in startup
// or shutdown. It may be an infrequent flake, so run with
// -count=10000 to be sure.
func TestDeviceStartStop(t *testing.T) {
	conn, err := Listen(Options{
		EndpointsFunc: func(eps []string) {},
		Logf:          t.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	tun := tuntest.NewChannelTUN()
	dev := device.NewDevice(tun.TUN(), &device.DeviceOptions{
		Logger:         devLogger(t, "dev"),
		CreateEndpoint: conn.CreateEndpoint,
		CreateBind:     conn.CreateBind,
		SkipBindUpdate: true,
	})
	dev.Up()
	dev.Close()
}

func TestTwoDevicePing(t *testing.T) {
	// Wipe default DERP list, add local server.
	// (Do it now, or derpHost will try to connect to derp1.tailscale.com.)
	derpServer, derpAddr, derpCleanupFn := runDERP(t)
	defer derpCleanupFn()

	stunAddr, stunCleanupFn := stuntest.Serve(t)
	defer stunCleanupFn()

	derps := derpmap.NewTestWorldWith(&derpmap.Server{
		ID:        1,
		HostHTTPS: derpAddr,
		STUN4:     stunAddr,
		Geo:       "Testopolis",
	})

	epCh1 := make(chan []string, 16)
	conn1, err := Listen(Options{
		Logf:  logger.WithPrefix(t.Logf, "conn1: "),
		DERPs: derps,
		EndpointsFunc: func(eps []string) {
			epCh1 <- eps
		},
		derpTLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn1.Close()

	epCh2 := make(chan []string, 16)
	conn2, err := Listen(Options{
		Logf:  logger.WithPrefix(t.Logf, "conn2: "),
		DERPs: derps,
		EndpointsFunc: func(eps []string) {
			epCh2 <- eps
		},
		derpTLSConfig: &tls.Config{InsecureSkipVerify: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()

	ports := []uint16{conn1.LocalPort(), conn2.LocalPort()}
	cfgs := makeConfigs(t, ports)

	if err := conn1.SetPrivateKey(cfgs[0].PrivateKey); err != nil {
		t.Fatal(err)
	}
	if err := conn2.SetPrivateKey(cfgs[1].PrivateKey); err != nil {
		t.Fatal(err)
	}

	//uapi1, _ := cfgs[0].ToUAPI()
	//t.Logf("cfg0: %v", uapi1)
	//uapi2, _ := cfgs[1].ToUAPI()
	//t.Logf("cfg1: %v", uapi2)

	tun1 := tuntest.NewChannelTUN()
	dev1 := device.NewDevice(tun1.TUN(), &device.DeviceOptions{
		Logger:         devLogger(t, "dev1"),
		CreateEndpoint: conn1.CreateEndpoint,
		CreateBind:     conn1.CreateBind,
		SkipBindUpdate: true,
	})
	dev1.Up()
	if err := dev1.Reconfig(&cfgs[0]); err != nil {
		t.Fatal(err)
	}
	defer dev1.Close()

	tun2 := tuntest.NewChannelTUN()
	dev2 := device.NewDevice(tun2.TUN(), &device.DeviceOptions{
		Logger:         devLogger(t, "dev2"),
		CreateEndpoint: conn2.CreateEndpoint,
		CreateBind:     conn2.CreateBind,
		SkipBindUpdate: true,
	})
	dev2.Up()
	defer dev2.Close()

	if err := dev2.Reconfig(&cfgs[1]); err != nil {
		t.Fatal(err)
	}

	ping1 := func(t *testing.T) {
		t.Helper()

		msg2to1 := tuntest.Ping(net.ParseIP("1.0.0.1"), net.ParseIP("1.0.0.2"))
		tun2.Outbound <- msg2to1
		select {
		case msgRecv := <-tun1.Inbound:
			if !bytes.Equal(msg2to1, msgRecv) {
				t.Error("ping did not transit correctly")
			}
		case <-time.After(3 * time.Second):
			t.Error("ping did not transit")
		}
	}
	ping2 := func(t *testing.T) {
		t.Helper()

		msg1to2 := tuntest.Ping(net.ParseIP("1.0.0.2"), net.ParseIP("1.0.0.1"))
		tun1.Outbound <- msg1to2
		select {
		case msgRecv := <-tun2.Inbound:
			if !bytes.Equal(msg1to2, msgRecv) {
				t.Error("return ping did not transit correctly")
			}
		case <-time.After(3 * time.Second):
			t.Error("return ping did not transit")
		}
	}

	t.Run("ping 1.0.0.1", func(t *testing.T) { ping1(t) })
	t.Run("ping 1.0.0.2", func(t *testing.T) { ping2(t) })
	t.Run("ping 1.0.0.2 via SendPacket", func(t *testing.T) {
		msg1to2 := tuntest.Ping(net.ParseIP("1.0.0.2"), net.ParseIP("1.0.0.1"))
		if err := dev1.SendPacket(msg1to2); err != nil {
			t.Fatal(err)
		}
		select {
		case msgRecv := <-tun2.Inbound:
			if !bytes.Equal(msg1to2, msgRecv) {
				t.Error("return ping did not transit correctly")
			}
		case <-time.After(3 * time.Second):
			t.Error("return ping did not transit")
		}
	})

	t.Run("no-op dev1 reconfig", func(t *testing.T) {
		if err := dev1.Reconfig(&cfgs[0]); err != nil {
			t.Fatal(err)
		}
		ping1(t)
		ping2(t)
	})

	if os.Getenv("RUN_CURSED_TESTS") == "" {
		t.Skip("test is very broken, don't run in CI until it's reliable.")
	}

	pingSeq := func(t *testing.T, count int, totalTime time.Duration, strict bool) {
		msg := func(i int) []byte {
			b := tuntest.Ping(net.ParseIP("1.0.0.2"), net.ParseIP("1.0.0.1"))
			b[len(b)-1] = byte(i) // set seq num
			return b
		}

		// Space out ping transmissions so that the overall
		// transmission happens in totalTime.
		//
		// We do this because the packet spray logic in magicsock is
		// time-based to allow for reliable NAT traversal. However,
		// for the packet spraying test further down, there needs to
		// be at least 1 sprayed packet that is not the handshake, in
		// case the handshake gets eaten by the race resolution logic.
		//
		// This is an inherent "race by design" in our current
		// magicsock+wireguard-go codebase: sometimes, racing
		// handshakes will result in a sub-optimal path for a few
		// hundred milliseconds, until a subsequent spray corrects the
		// issue. In order for the test to reflect that magicsock
		// works as designed, we have to space out packet transmission
		// here.
		interPacketGap := totalTime / time.Duration(count)
		if interPacketGap < 1*time.Millisecond {
			interPacketGap = 0
		}

		for i := 0; i < count; i++ {
			b := msg(i)
			tun1.Outbound <- b
			time.Sleep(interPacketGap)
		}

		for i := 0; i < count; i++ {
			b := msg(i)
			select {
			case msgRecv := <-tun2.Inbound:
				if !bytes.Equal(b, msgRecv) {
					if strict {
						t.Errorf("return ping %d did not transit correctly: %s", i, cmp.Diff(b, msgRecv))
					}
				}
			case <-time.After(3 * time.Second):
				if strict {
					t.Errorf("return ping %d did not transit", i)
				}
			}
		}

	}

	t.Run("ping 1.0.0.1 x50", func(t *testing.T) {
		pingSeq(t, 50, 0, true)
	})

	// Add DERP relay.
	derpEp := wgcfg.Endpoint{Host: "127.3.3.40", Port: 1}
	ep0 := cfgs[0].Peers[0].Endpoints
	ep0 = append([]wgcfg.Endpoint{derpEp}, ep0...)
	cfgs[0].Peers[0].Endpoints = ep0
	ep1 := cfgs[1].Peers[0].Endpoints
	ep1 = append([]wgcfg.Endpoint{derpEp}, ep1...)
	cfgs[1].Peers[0].Endpoints = ep1
	if err := dev1.Reconfig(&cfgs[0]); err != nil {
		t.Fatal(err)
	}
	if err := dev2.Reconfig(&cfgs[1]); err != nil {
		t.Fatal(err)
	}

	t.Run("add DERP", func(t *testing.T) {
		defer func() {
			t.Logf("DERP vars: %s", derpServer.ExpVar().String())
		}()
		pingSeq(t, 20, 0, true)
	})

	// Disable real route.
	cfgs[0].Peers[0].Endpoints = []wgcfg.Endpoint{derpEp}
	cfgs[1].Peers[0].Endpoints = []wgcfg.Endpoint{derpEp}
	if err := dev1.Reconfig(&cfgs[0]); err != nil {
		t.Fatal(err)
	}
	if err := dev2.Reconfig(&cfgs[1]); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond) // TODO remove

	t.Run("all traffic over DERP", func(t *testing.T) {
		defer func() {
			t.Logf("DERP vars: %s", derpServer.ExpVar().String())
			if t.Failed() || true {
				uapi1, _ := cfgs[0].ToUAPI()
				t.Logf("cfg0: %v", uapi1)
				uapi2, _ := cfgs[1].ToUAPI()
				t.Logf("cfg1: %v", uapi2)
			}
		}()
		pingSeq(t, 20, 0, true)
	})

	dev1.RemoveAllPeers()
	dev2.RemoveAllPeers()

	// Give one peer a non-DERP endpoint. We expect the other to
	// accept it via roamAddr.
	cfgs[0].Peers[0].Endpoints = ep0
	if ep2 := cfgs[1].Peers[0].Endpoints; len(ep2) != 1 {
		t.Errorf("unexpected peer endpoints in dev2: %v", ep2)
	}
	if err := dev2.Reconfig(&cfgs[1]); err != nil {
		t.Fatal(err)
	}
	if err := dev1.Reconfig(&cfgs[0]); err != nil {
		t.Fatal(err)
	}
	// Dear future human debugging a test failure here: this test is
	// flaky, and very infrequently will drop 1-2 of the 50 ping
	// packets. This does not affect normal operation of tailscaled,
	// but makes this test fail.
	//
	// TODO(danderson): finish root-causing and de-flake this test.
	t.Run("one real route is enough thanks to spray", func(t *testing.T) {
		pingSeq(t, 50, 700*time.Millisecond, false)

		ep2 := dev2.Config().Peers[0].Endpoints
		if len(ep2) != 2 {
			t.Error("handshake spray failed to find real route")
		}
	})
}

// TestAddrSet tests AddrSet appendDests and UpdateDst.
func TestAddrSet(t *testing.T) {
	mustUDPAddr := func(s string) *net.UDPAddr {
		t.Helper()
		ua, err := net.ResolveUDPAddr("udp", s)
		if err != nil {
			t.Fatal(err)
		}
		return ua
	}
	udpAddrs := func(ss ...string) (ret []net.UDPAddr) {
		t.Helper()
		for _, s := range ss {
			ret = append(ret, *mustUDPAddr(s))
		}
		return ret
	}
	joinUDPs := func(in []*net.UDPAddr) string {
		var sb strings.Builder
		for i, ua := range in {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(ua.String())
		}
		return sb.String()
	}
	var (
		regPacket   = []byte("some regular packet")
		sprayPacket = []byte("0000")
	)
	binary.LittleEndian.PutUint32(sprayPacket[:4], device.MessageInitiationType)
	if !shouldSprayPacket(sprayPacket) {
		t.Fatal("sprayPacket should be classified as a spray packet for testing")
	}

	// A step is either a b+want appendDests tests, or an
	// UpdateDst call, depending on which fields are set.
	type step struct {
		// advance is the time to advance the fake clock
		// before the step.
		advance time.Duration

		// updateDst, if set, does an UpdateDst call and
		// b+want are ignored.
		updateDst *net.UDPAddr

		b    []byte
		want string // comma-separated
	}
	tests := []struct {
		name  string
		as    *AddrSet
		steps []step
	}{
		{
			name: "reg_packet_no_curaddr",
			as: &AddrSet{
				addrs:    udpAddrs("127.3.3.40:1", "123.45.67.89:123", "10.0.0.1:123"),
				curAddr:  -1, // unknown
				roamAddr: nil,
			},
			steps: []step{
				{b: regPacket, want: "127.3.3.40:1"},
			},
		},
		{
			name: "reg_packet_have_curaddr",
			as: &AddrSet{
				addrs:    udpAddrs("127.3.3.40:1", "123.45.67.89:123", "10.0.0.1:123"),
				curAddr:  1, // global IP
				roamAddr: nil,
			},
			steps: []step{
				{b: regPacket, want: "123.45.67.89:123"},
			},
		},
		{
			name: "reg_packet_have_roamaddr",
			as: &AddrSet{
				addrs:    udpAddrs("127.3.3.40:1", "123.45.67.89:123", "10.0.0.1:123"),
				curAddr:  2, // should be ignored
				roamAddr: mustUDPAddr("5.6.7.8:123"),
			},
			steps: []step{
				{b: regPacket, want: "5.6.7.8:123"},
				{updateDst: mustUDPAddr("10.0.0.1:123")}, // no more roaming
				{b: regPacket, want: "10.0.0.1:123"},
			},
		},
		{
			name: "start_roaming",
			as: &AddrSet{
				addrs:   udpAddrs("127.3.3.40:1", "123.45.67.89:123", "10.0.0.1:123"),
				curAddr: 2,
			},
			steps: []step{
				{b: regPacket, want: "10.0.0.1:123"},
				{updateDst: mustUDPAddr("4.5.6.7:123")},
				{b: regPacket, want: "4.5.6.7:123"},
				{updateDst: mustUDPAddr("5.6.7.8:123")},
				{b: regPacket, want: "5.6.7.8:123"},
				{updateDst: mustUDPAddr("123.45.67.89:123")}, // end roaming
				{b: regPacket, want: "123.45.67.89:123"},
			},
		},
		{
			name: "spray_packet",
			as: &AddrSet{
				addrs:    udpAddrs("127.3.3.40:1", "123.45.67.89:123", "10.0.0.1:123"),
				curAddr:  2, // should be ignored
				roamAddr: mustUDPAddr("5.6.7.8:123"),
			},
			steps: []step{
				{b: sprayPacket, want: "127.3.3.40:1,123.45.67.89:123,10.0.0.1:123,5.6.7.8:123"},
				{advance: 300 * time.Millisecond, b: regPacket, want: "127.3.3.40:1,123.45.67.89:123,10.0.0.1:123,5.6.7.8:123"},
				{advance: 300 * time.Millisecond, b: regPacket, want: "127.3.3.40:1,123.45.67.89:123,10.0.0.1:123,5.6.7.8:123"},
				{advance: 3, b: regPacket, want: "5.6.7.8:123"},
				{advance: 2 * time.Millisecond, updateDst: mustUDPAddr("10.0.0.1:123")},
				{advance: 3, b: regPacket, want: "10.0.0.1:123"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faket := time.Unix(0, 0)
			tt.as.Logf = t.Logf
			tt.as.clock = func() time.Time { return faket }
			for i, st := range tt.steps {
				faket = faket.Add(st.advance)

				if st.updateDst != nil {
					if err := tt.as.UpdateDst(st.updateDst); err != nil {
						t.Fatal(err)
					}
					continue
				}
				got, _ := tt.as.appendDests(nil, st.b)
				if gotStr := joinUDPs(got); gotStr != st.want {
					t.Errorf("step %d: got %v; want %v", i, gotStr, st.want)
				}
			}
		})
	}
}
