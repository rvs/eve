// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build depends_on_currently_unreleased

package ipn

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tailscale/wireguard-go/tun/tuntest"
	"tailscale.com/control/controlclient"
	"tailscale.com/tailcfg"
	"tailscale.com/testy"
	"tailscale.com/wgengine"
	"tailscale.com/wgengine/magicsock"
	"tailscale.io/control" // not yet released
)

func TestIPN(t *testing.T) {
	testy.FixLogs(t)
	defer testy.UnfixLogs(t)

	// Turn off STUN for the test to make it hermitic.
	// TODO(crawshaw): add a test that runs against a local STUN server.
	magicsock.DisableSTUNForTesting = true
	defer func() { magicsock.DisableSTUNForTesting = false }()

	// TODO(apenwarr): Make resource checks actually pass.
	// They don't right now, because (at least) wgengine doesn't fully
	// shut down.
	//	rc := testy.NewResourceCheck()
	//	defer rc.Assert(t)

	var ctl *control.Server

	ctlHandler := func(w http.ResponseWriter, r *http.Request) {
		ctl.ServeHTTP(w, r)
	}
	https := httptest.NewServer(http.HandlerFunc(ctlHandler))
	serverURL := https.URL
	defer https.Close()
	defer https.CloseClientConnections()

	tmpdir, err := ioutil.TempDir("", "ipntest")
	if err != nil {
		t.Fatalf("create tempdir: %v\n", err)
	}
	ctl, err = control.New(tmpdir, tmpdir, serverURL, true)
	if err != nil {
		t.Fatalf("create control server: %v\n", ctl)
	}

	n1 := newNode(t, "n1", https)
	defer n1.Backend.Shutdown()
	n1.Backend.StartLoginInteractive()

	n2 := newNode(t, "n2", https)
	defer n2.Backend.Shutdown()
	n2.Backend.StartLoginInteractive()

	t.Run("login", func(t *testing.T) {
		var s1, s2 State
		for {
			t.Logf("\n\nn1.state=%v n2.state=%v\n\n", s1, s2)

			// TODO(crawshaw): switch from || to &&. To do this we need to
			// transmit some data so that the handshake completes on both
			// sides. (Because handshakes are 1RTT, it is the data
			// transmission that completes the handshake.)
			if s1 == Running || s2 == Running {
				// TODO(apenwarr): ensure state sequence.
				// Right now we'll just exit as soon as
				// state==Running, even if the backend is lying or
				// something. Not a great test.
				break
			}

			select {
			case n := <-n1.NotifyCh:
				t.Logf("n1n: %v\n", n)
				if n.State != nil {
					s1 = *n.State
					if s1 == NeedsMachineAuth {
						authNode(t, ctl, n1.Backend)
					}
				}
			case n := <-n2.NotifyCh:
				t.Logf("n2n: %v\n", n)
				if n.State != nil {
					s2 = *n.State
					if s2 == NeedsMachineAuth {
						authNode(t, ctl, n2.Backend)
					}
				}
			case <-time.After(3 * time.Second):
				t.Fatalf("\n\n\nFATAL: timed out waiting for notifications.\n\n\n")
			}
		}
	})

	n1addr := n1.Backend.NetMap().Addresses[0].IP
	n2addr := n2.Backend.NetMap().Addresses[0].IP
	t.Run("ping n2", func(t *testing.T) {
		t.Skip("TODO(crawshaw): skipping ping test, it is flaky")
		msg := tuntest.Ping(n2addr.IP(), n1addr.IP())
		n1.ChannelTUN.Outbound <- msg
		select {
		case msgRecv := <-n2.ChannelTUN.Inbound:
			if !bytes.Equal(msg, msgRecv) {
				t.Error("bad ping")
			}
		case <-time.After(1 * time.Second):
			t.Error("no ping seen")
		}
	})
	t.Run("ping n1", func(t *testing.T) {
		t.Skip("TODO(crawshaw): skipping ping test, it is flaky")
		msg := tuntest.Ping(n1addr.IP(), n2addr.IP())
		n2.ChannelTUN.Outbound <- msg
		select {
		case msgRecv := <-n1.ChannelTUN.Inbound:
			if !bytes.Equal(msg, msgRecv) {
				t.Error("bad ping")
			}
		case <-time.After(1 * time.Second):
			t.Error("no ping seen")
		}
	})

drain:
	for {
		select {
		case <-n1.NotifyCh:
		case <-n2.NotifyCh:
		default:
			break drain
		}
	}

	n1.Backend.Logout()

	t.Run("logout", func(t *testing.T) {
		select {
		case n := <-n1.NotifyCh:
			if n.State != nil {
				if *n.State != NeedsLogin {
					t.Errorf("n.State=%v, want %v", n.State, NeedsLogin)
					return
				}
			}
		case <-time.After(3 * time.Second):
			t.Fatalf("timeout waiting for logout notification")
		}
	})
}

type testNode struct {
	Backend    *LocalBackend
	ChannelTUN *tuntest.ChannelTUN
	NotifyCh   <-chan Notify
}

// Create a new IPN node.
func newNode(t *testing.T, prefix string, https *httptest.Server) testNode {
	t.Helper()
	logfe := func(fmt string, args ...interface{}) {
		t.Logf(prefix+".e: "+fmt, args...)
	}
	logf := func(fmt string, args ...interface{}) {
		t.Logf(prefix+": "+fmt, args...)
	}

	tun := tuntest.NewChannelTUN()
	e1, err := wgengine.NewUserspaceEngineAdvanced(logfe, tun.TUN(), wgengine.NewFakeRouter, 0)
	if err != nil {
		t.Fatalf("NewFakeEngine: %v\n", err)
	}
	n, err := NewLocalBackend(logf, prefix, &MemoryStore{}, e1)
	if err != nil {
		t.Fatalf("NewLocalBackend: %v\n", err)
	}
	nch := make(chan Notify, 1000)
	c := controlclient.Persist{
		Provider:  "google",
		LoginName: "test1@tailscale.com",
	}
	prefs := NewPrefs()
	prefs.ControlURL = https.URL
	prefs.Persist = &c
	n.Start(Options{
		FrontendLogID: prefix + "-f",
		Prefs:         prefs,
		Notify: func(n Notify) {
			// Automatically visit auth URLs
			if n.BrowseToURL != nil {
				t.Logf("\n\n\nURL! %vv\n", *n.BrowseToURL)
				hc := https.Client()
				_, err := hc.Get(*n.BrowseToURL)
				if err != nil {
					t.Logf("BrowseToURL: %v\n", err)
				}
			}
			nch <- n
		},
	})

	return testNode{
		Backend:    n,
		ChannelTUN: tun,
		NotifyCh:   nch,
	}
}

// Tell the control server to authorize the given node.
func authNode(t *testing.T, ctl *control.Server, n *LocalBackend) {
	mk := n.prefs.Persist.PrivateMachineKey.Public()
	nk := n.prefs.Persist.PrivateNodeKey.Public()
	ctl.AuthorizeMachine(tailcfg.MachineKey(mk), tailcfg.NodeKey(nk))
}
