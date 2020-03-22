// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package derp

import (
	"bufio"
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"tailscale.com/net/nettest"
	"tailscale.com/types/key"
)

func newPrivateKey(t *testing.T) (k key.Private) {
	t.Helper()
	if _, err := crand.Read(k[:]); err != nil {
		t.Fatal(err)
	}
	return
}

func TestSendRecv(t *testing.T) {
	serverPrivateKey := newPrivateKey(t)
	s := NewServer(serverPrivateKey, t.Logf)
	defer s.Close()

	const numClients = 3
	var clientPrivateKeys []key.Private
	var clientKeys []key.Public
	for i := 0; i < numClients; i++ {
		priv := newPrivateKey(t)
		clientPrivateKeys = append(clientPrivateKeys, priv)
		clientKeys = append(clientKeys, priv.Public())
	}

	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	var clients []*Client
	var connsOut []Conn
	var recvChs []chan []byte
	errCh := make(chan error, 3)

	for i := 0; i < numClients; i++ {
		t.Logf("Connecting client %d ...", i)
		cout, err := net.Dial("tcp", ln.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		defer cout.Close()
		connsOut = append(connsOut, cout)

		cin, err := ln.Accept()
		if err != nil {
			t.Fatal(err)
		}
		defer cin.Close()
		brwServer := bufio.NewReadWriter(bufio.NewReader(cin), bufio.NewWriter(cin))
		go s.Accept(cin, brwServer, fmt.Sprintf("test-client-%d", i))

		key := clientPrivateKeys[i]
		brw := bufio.NewReadWriter(bufio.NewReader(cout), bufio.NewWriter(cout))
		c, err := NewClient(key, cout, brw, t.Logf)
		if err != nil {
			t.Fatalf("client %d: %v", i, err)
		}
		clients = append(clients, c)
		recvChs = append(recvChs, make(chan []byte))
		t.Logf("Connected client %d.", i)
	}

	t.Logf("Starting read loops")
	for i := 0; i < numClients; i++ {
		go func(i int) {
			for {
				b := make([]byte, 1<<16)
				m, err := clients[i].Recv(b)
				if err != nil {
					errCh <- err
					return
				}
				switch m := m.(type) {
				default:
					t.Errorf("unexpected message type %T", m)
					continue
				case ReceivedPacket:
					if m.Source.IsZero() {
						t.Errorf("zero Source address in ReceivedPacket")
					}
					recvChs[i] <- m.Data
				}
			}
		}(i)
	}

	recv := func(i int, want string) {
		t.Helper()
		select {
		case b := <-recvChs[i]:
			if got := string(b); got != want {
				t.Errorf("client1.Recv=%q, want %q", got, want)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("client%d.Recv, got nothing, want %q", i, want)
		}
	}
	recvNothing := func(i int) {
		t.Helper()
		select {
		case b := <-recvChs[0]:
			t.Errorf("client%d.Recv=%q, want nothing", i, string(b))
		default:
		}
	}

	wantActive := func(total, home int64) {
		t.Helper()
		dl := time.Now().Add(5 * time.Second)
		var gotTotal, gotHome int64
		for time.Now().Before(dl) {
			gotTotal, gotHome = s.curClients.Value(), s.curHomeClients.Value()
			if gotTotal == total && gotHome == home {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Errorf("total/home=%v/%v; want %v/%v", gotTotal, gotHome, total, home)
	}

	msg1 := []byte("hello 0->1\n")
	if err := clients[0].Send(clientKeys[1], msg1); err != nil {
		t.Fatal(err)
	}
	recv(1, string(msg1))
	recvNothing(0)
	recvNothing(2)

	msg2 := []byte("hello 1->2\n")
	if err := clients[1].Send(clientKeys[2], msg2); err != nil {
		t.Fatal(err)
	}
	recv(2, string(msg2))
	recvNothing(0)
	recvNothing(1)

	wantActive(3, 0)
	clients[0].NotePreferred(true)
	wantActive(3, 1)
	clients[0].NotePreferred(true)
	wantActive(3, 1)
	clients[0].NotePreferred(false)
	wantActive(3, 0)
	clients[0].NotePreferred(false)
	wantActive(3, 0)
	clients[1].NotePreferred(true)
	wantActive(3, 1)
	connsOut[1].Close()
	wantActive(2, 0)
	clients[2].NotePreferred(true)
	wantActive(2, 1)
	clients[2].NotePreferred(false)
	wantActive(2, 0)
	connsOut[2].Close()
	wantActive(1, 0)

	t.Logf("passed")
	s.Close()
}

func TestSendFreeze(t *testing.T) {
	serverPrivateKey := newPrivateKey(t)
	s := NewServer(serverPrivateKey, t.Logf)
	defer s.Close()
	s.WriteTimeout = 100 * time.Millisecond

	// We send two streams of messages:
	//
	//	alice --> bob
	//	alice --> cathy
	//
	// Then cathy stops processing messsages.
	// That should not interfere with alice talking to bob.

	newClient := func(name string, k key.Private) (c *Client, clientConn nettest.Conn) {
		t.Helper()
		c1, c2 := nettest.NewConn(name, 1024)
		go s.Accept(c1, bufio.NewReadWriter(bufio.NewReader(c1), bufio.NewWriter(c1)), name)

		brw := bufio.NewReadWriter(bufio.NewReader(c2), bufio.NewWriter(c2))
		c, err := NewClient(k, c2, brw, t.Logf)
		if err != nil {
			t.Fatal(err)
		}
		return c, c2
	}

	aliceKey := newPrivateKey(t)
	aliceClient, aliceConn := newClient("alice", aliceKey)

	bobKey := newPrivateKey(t)
	bobClient, bobConn := newClient("bob", bobKey)

	cathyKey := newPrivateKey(t)
	cathyClient, cathyConn := newClient("cathy", cathyKey)

	var (
		aliceCh = make(chan struct{}, 32)
		bobCh   = make(chan struct{}, 32)
		cathyCh = make(chan struct{}, 32)
	)
	chs := func(name string) chan struct{} {
		switch name {
		case "alice":
			return aliceCh
		case "bob":
			return bobCh
		case "cathy":
			return cathyCh
		default:
			panic("unknown ch: " + name)
		}
	}

	errCh := make(chan error, 4)
	recv := func(name string, client *Client) {
		ch := chs(name)
		for {
			b := make([]byte, 1<<9)
			m, err := client.Recv(b)
			if err != nil {
				errCh <- fmt.Errorf("%s: %w", name, err)
				return
			}
			switch m := m.(type) {
			default:
				errCh <- fmt.Errorf("%s: unexpected message type %T", name, m)
				return
			case ReceivedPacket:
				if m.Source.IsZero() {
					errCh <- fmt.Errorf("%s: zero Source address in ReceivedPacket", name)
					return
				}
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}
	go recv("alice", aliceClient)
	go recv("bob", bobClient)
	go recv("cathy", cathyClient)

	var cancel func()
	go func() {
		t := time.NewTicker(2 * time.Millisecond)
		defer t.Stop()
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		for {
			select {
			case <-t.C:
			case <-ctx.Done():
				errCh <- nil
				return
			}

			msg1 := []byte("hello alice->bob\n")
			if err := aliceClient.Send(bobKey.Public(), msg1); err != nil {
				errCh <- fmt.Errorf("alice send to bob: %w", err)
				return
			}
			msg2 := []byte("hello alice->cathy\n")

			// TODO: an error is expected here.
			// We ignore it, maybe we should log it somehow?
			aliceClient.Send(cathyKey.Public(), msg2)
		}
	}()

	drainAny := func(ch chan struct{}) {
		// We are draining potentially infinite sources,
		// so place some reasonable upper limit.
		//
		// The important thing here is to make sure that
		// if any tokens remain in the channel, they
		// must have been generated after drainAny was
		// called.
		for i := 0; i < cap(ch); i++ {
			select {
			case <-ch:
			default:
				return
			}
		}
	}
	drain := func(t *testing.T, name string) bool {
		t.Helper()
		timer := time.NewTimer(1 * time.Second)
		defer timer.Stop()

		// Ensure ch has at least one element.
		ch := chs(name)
		select {
		case <-ch:
		case <-timer.C:
			t.Errorf("no packet received by %s", name)
			return false
		}
		// Drain remaining.
		drainAny(ch)
		return true
	}
	isEmpty := func(t *testing.T, name string) {
		t.Helper()
		select {
		case <-chs(name):
			t.Errorf("packet received by %s, want none", name)
		default:
		}
	}

	t.Run("initial send", func(t *testing.T) {
		drain(t, "bob")
		drain(t, "cathy")
		isEmpty(t, "alice")
	})

	t.Run("block cathy", func(t *testing.T) {
		// Block cathy. Now the cathyConn buffer will fill up quickly,
		// and the derp server will back up.
		cathyConn.SetReadBlock(true)
		time.Sleep(2 * s.WriteTimeout)

		drain(t, "bob")
		drainAny(chs("cathy"))
		isEmpty(t, "alice")

		// Now wait a little longer, and ensure packets still flow to bob
		if !drain(t, "bob") {
			t.Errorf("connection alice->bob frozen by alice->cathy")
		}
	})

	// Cleanup, make sure we process all errors.
	t.Logf("TEST COMPLETE, cancelling sender")
	cancel()
	t.Logf("closing connections")
	aliceConn.Close()
	bobConn.Close()
	cathyConn.Close()

	for i := 0; i < cap(errCh); i++ {
		err := <-errCh
		if err != nil {
			if errors.Is(err, io.EOF) {
				continue
			}
			t.Error(err)
		}
	}
}
