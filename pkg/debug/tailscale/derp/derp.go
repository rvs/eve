// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package derp implements DERP, the Detour Encrypted Routing Protocol.
//
// DERP routes packets to clients using curve25519 keys as addresses.
//
// DERP is used by Tailscale nodes to proxy encrypted WireGuard
// packets through the Tailscale cloud servers when a direct path
// cannot be found or opened. DERP is a last resort. Both sides
// between very aggressive NATs, firewalls, no IPv6, etc? Well, DERP.
package derp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"time"
)

// MaxPacketSize is the maximum size of a packet sent over DERP.
// (This only includes the data bytes visible to magicsock, not
// including its on-wire framing overhead)
const MaxPacketSize = 64 << 10

// magic is the DERP magic number, sent in the frameServerKey frame
// upon initial connection.
const magic = "DERP🔑" // 8 bytes: 0x44 45 52 50 f0 9f 94 91

const (
	nonceLen   = 24
	keyLen     = 32
	maxInfoLen = 1 << 20
	keepAlive  = 60 * time.Second
)

// protocolVersion is bumped whenever there's a wire-incompatible change.
//   * version 1 (zero on wire): consistent box headers, in use by employee dev nodes a bit
//   * version 2: received packets have src addrs in frameRecvPacket at beginning
const protocolVersion = 2

const (
	protocolSrcAddrs = 2 // protocol version at which client expects src addresses
)

// frameType is the one byte frame type at the beginning of the frame
// header.  The second field is a big-endian uint32 describing the
// length of the remaining frame (not including the initial 5 bytes).
type frameType byte

/*
Protocol flow:

Login:
* client connects
* server sends frameServerKey
* client sends frameClientInfo
* server sends frameServerInfo

Steady state:
* server occasionally sends frameKeepAlive
* client sends frameSendPacket
* server then sends frameRecvPacket to recipient
*/
const (
	frameServerKey     = frameType(0x01) // 8B magic + 32B public key + (0+ bytes future use)
	frameClientInfo    = frameType(0x02) // 32B pub key + 24B nonce + naclbox(json)
	frameServerInfo    = frameType(0x03) // 24B nonce + naclbox(json)
	frameSendPacket    = frameType(0x04) // 32B dest pub key + packet bytes
	frameRecvPacket    = frameType(0x05) // v0/1: packet bytes, v2: 32B src pub key + packet bytes
	frameKeepAlive     = frameType(0x06) // no payload, no-op (to be replaced with ping/pong)
	frameNotePreferred = frameType(0x07) // 1 byte payload: 0x01 or 0x00 for whether this is client's home node
)

var bin = binary.BigEndian

func writeUint32(bw *bufio.Writer, v uint32) error {
	var b [4]byte
	bin.PutUint32(b[:], v)
	_, err := bw.Write(b[:])
	return err
}

func readUint32(br *bufio.Reader) (uint32, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(br, b); err != nil {
		return 0, err
	}
	return bin.Uint32(b), nil
}

func readFrameTypeHeader(br *bufio.Reader, wantType frameType) (frameLen uint32, err error) {
	gotType, frameLen, err := readFrameHeader(br)
	if err == nil && wantType != gotType {
		err = fmt.Errorf("bad frame type 0x%X, want 0x%X", gotType, wantType)
	}
	return frameLen, err
}

func readFrameHeader(br *bufio.Reader) (t frameType, frameLen uint32, err error) {
	tb, err := br.ReadByte()
	if err != nil {
		return 0, 0, err
	}
	frameLen, err = readUint32(br)
	if err != nil {
		return 0, 0, err
	}
	return frameType(tb), frameLen, nil
}

// readFrame reads a frame header and then reads its payload into
// b[:frameLen].
//
// If the frame header length is greater than maxSize, readFrame returns
// an error after reading the frame header.
//
// If the frame is less than maxSize but greater than len(b), len(b)
// bytes are read, err will be io.ErrShortBuffer, and frameLen and t
// will both be set. That is, callers need to explicitly handle when
// they get more data than expected.
func readFrame(br *bufio.Reader, maxSize uint32, b []byte) (t frameType, frameLen uint32, err error) {
	t, frameLen, err = readFrameHeader(br)
	if err != nil {
		return 0, 0, err
	}
	if frameLen > maxSize {
		return 0, 0, fmt.Errorf("frame header size %d exceeds reader limit of %d", frameLen, maxSize)
	}
	n, err := io.ReadFull(br, b[:frameLen])
	if err != nil {
		return 0, 0, err
	}
	remain := frameLen - uint32(n)
	if remain > 0 {
		if _, err := io.CopyN(ioutil.Discard, br, int64(remain)); err != nil {
			return 0, 0, err
		}
		err = io.ErrShortBuffer
	}
	return t, frameLen, err
}

func writeFrameHeader(bw *bufio.Writer, t frameType, frameLen uint32) error {
	if err := bw.WriteByte(byte(t)); err != nil {
		return err
	}
	return writeUint32(bw, frameLen)
}

// writeFrame writes a complete frame & flushes it.
func writeFrame(bw *bufio.Writer, t frameType, b []byte) error {
	if len(b) > 10<<20 {
		return errors.New("unreasonably large frame write")
	}
	if err := writeFrameHeader(bw, t, uint32(len(b))); err != nil {
		return err
	}
	if _, err := bw.Write(b); err != nil {
		return err
	}
	return bw.Flush()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
