// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package logpolicy manages the creation or reuse of logtail loggers,
// caching collection instance state on disk for use on future runs of
// programs on the same machine.
package logpolicy

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/klauspost/compress/zstd"
	"golang.org/x/crypto/ssh/terminal"
	"tailscale.com/atomicfile"
	"tailscale.com/logtail"
	"tailscale.com/logtail/filch"
	"tailscale.com/version"
)

// Config represents an instance of logs in a collection.
type Config struct {
	Collection string
	PrivateID  logtail.PrivateID
	PublicID   logtail.PublicID
}

// Policy is a logger and its public ID.
type Policy struct {
	// Logtail is the logger.
	Logtail logtail.Logger
	// PublicID is the logger's instance identifier.
	PublicID logtail.PublicID
}

// ToBytes returns the JSON representation of c.
func (c *Config) ToBytes() []byte {
	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		log.Fatalf("logpolicy.Config marshal: %v\n", err)
	}
	return data
}

// Save writes the JSON representation of c to stateFile.
func (c *Config) save(stateFile string) error {
	c.PublicID = c.PrivateID.Public()
	if err := os.MkdirAll(filepath.Dir(stateFile), 0750); err != nil {
		return err
	}
	data := c.ToBytes()
	if err := atomicfile.WriteFile(stateFile, data, 0600); err != nil {
		return err
	}
	return nil
}

// ConfigFromBytes parses a a Config from its JSON encoding.
func ConfigFromBytes(jsonEnc []byte) (*Config, error) {
	c := &Config{}
	if err := json.Unmarshal(jsonEnc, c); err != nil {
		return nil, err
	}
	return c, nil
}

// stderrWriter is an io.Writer that always writes to the latest
// os.Stderr, even if os.Stderr changes during the lifetime of the
// stderrWriter value.
type stderrWriter struct{}

func (stderrWriter) Write(buf []byte) (int, error) {
	return os.Stderr.Write(buf)
}

type logWriter struct {
	logger *log.Logger
}

func (l logWriter) Write(buf []byte) (int, error) {
	l.logger.Printf("%s", buf)
	return len(buf), nil
}

// logsDir returns the directory to use for log configuration and
// buffer storage.
func logsDir() string {
	systemdCacheDir := os.Getenv("CACHE_DIRECTORY")
	if systemdCacheDir != "" {
		return systemdCacheDir
	}

	cacheDir, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(cacheDir, "Tailscale")
	}

	// No idea where to put stuff. This only happens when $HOME is
	// unset, which os.UserCacheDir doesn't like. Use the current
	// working directory and hope for the best.
	return ""
}

// New returns a new log policy (a logger and its instance ID) for a
// given collection name.
func New(collection string) *Policy {
	var lflags int
	if terminal.IsTerminal(2) || runtime.GOOS == "windows" {
		lflags = 0
	} else {
		lflags = log.LstdFlags
	}
	console := log.New(stderrWriter{}, "", lflags)

	dir := logsDir()
	cfgPath := filepath.Join(dir, fmt.Sprintf("%s.log.conf", version.CmdName()))
	var oldc *Config
	data, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		log.Printf("logpolicy.Read %v: %v\n", cfgPath, err)
		oldc = &Config{}
		oldc.Collection = collection
	} else {
		oldc, err = ConfigFromBytes(data)
		if err != nil {
			log.Printf("logpolicy.Config unmarshal: %v\n", err)
			oldc = &Config{}
		}
	}

	newc := *oldc
	if newc.Collection != collection {
		log.Printf("logpolicy.Config: config collection %q does not match %q", newc.Collection, collection)
		// We picked up an incompatible config file.
		// Regenerate the private ID.
		newc.PrivateID = logtail.PrivateID{}
		newc.Collection = collection
	}
	if newc.PrivateID == (logtail.PrivateID{}) {
		newc.PrivateID, err = logtail.NewPrivateID()
		if err != nil {
			log.Fatalf("logpolicy: NewPrivateID() should never fail")
		}
	}
	newc.PublicID = newc.PrivateID.Public()
	if newc != *oldc {
		if err := newc.save(cfgPath); err != nil {
			log.Printf("logpolicy.Config.Save: %v\n", err)
		}
	}

	c := logtail.Config{
		Collection: newc.Collection,
		PrivateID:  newc.PrivateID,
		Stderr:     logWriter{console},
		NewZstdEncoder: func() logtail.Encoder {
			w, err := zstd.NewWriter(nil)
			if err != nil {
				panic(err)
			}
			return w
		},
	}

	filchBuf, filchErr := filch.New(filepath.Join(dir, version.CmdName()), filch.Options{})
	if filchBuf != nil {
		c.Buffer = filchBuf
	}
	lw := logtail.Log(c)
	log.SetFlags(0) // other logflags are set on console, not here
	log.SetOutput(lw)

	log.Printf("Program starting: v%v: %#v\n", version.LONG, os.Args)
	log.Printf("LogID: %v\n", newc.PublicID)
	if filchErr != nil {
		log.Printf("filch failed: %v", err)
	}

	return &Policy{
		Logtail:  lw,
		PublicID: newc.PublicID,
	}
}

// Close immediately shuts down the logger.
func (p *Policy) Close() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p.Shutdown(ctx)
}

// Shutdown gracefully shuts down the logger, finishing any current
// log upload if it can be done before ctx is canceled.
func (p *Policy) Shutdown(ctx context.Context) error {
	if p.Logtail != nil {
		log.Printf("flushing log.\n")
		return p.Logtail.Shutdown(ctx)
	}
	return nil
}
