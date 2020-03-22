// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tsweb contains code used in various Tailscale webservers.
package tsweb

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"expvar"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"tailscale.com/metrics"
	"tailscale.com/net/interfaces"
	"tailscale.com/types/logger"
)

// DevMode controls whether extra output in shown, for when the binary is being run in dev mode.
var DevMode bool

// NewMux returns a new ServeMux with debugHandler registered (and protected) at /debug/.
func NewMux(debugHandler http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	registerCommonDebug(mux)
	mux.Handle("/debug/", Protected(debugHandler))
	return mux
}

func registerCommonDebug(mux *http.ServeMux) {
	expvar.Publish("counter_uptime_sec", expvar.Func(func() interface{} { return int64(Uptime().Seconds()) }))
	mux.Handle("/debug/pprof/", Protected(http.DefaultServeMux)) // to net/http/pprof
	mux.Handle("/debug/vars", Protected(http.DefaultServeMux))   // to expvar
	mux.Handle("/debug/varz", Protected(http.HandlerFunc(varzHandler)))
}

func DefaultCertDir(leafDir string) string {
	cacheDir, err := os.UserCacheDir()
	if err == nil {
		return filepath.Join(cacheDir, "tailscale", leafDir)
	}
	return ""
}

// IsProd443 reports whether addr is a Go listen address for port 443.
func IsProd443(addr string) bool {
	_, port, _ := net.SplitHostPort(addr)
	return port == "443" || port == "https"
}

// AllowDebugAccess reports whether r should be permitted to access
// various debug endpoints.
func AllowDebugAccess(r *http.Request) bool {
	if r.Header.Get("X-Forwarded-For") != "" {
		// TODO if/when needed. For now, conservative:
		return false
	}
	ipStr, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(ipStr)
	if interfaces.IsTailscaleIP(ip) || ip.IsLoopback() || ipStr == os.Getenv("TS_ALLOW_DEBUG_IP") {
		return true
	}
	if r.Method == "GET" {
		urlKey := r.FormValue("debugkey")
		keyPath := os.Getenv("TS_DEBUG_KEY_PATH")
		if urlKey != "" && keyPath != "" {
			slurp, err := ioutil.ReadFile(keyPath)
			if err == nil && string(bytes.TrimSpace(slurp)) == urlKey {
				return true
			}
		}
	}
	return false
}

// Protected wraps a provided debug handler, h, returning a Handler
// that enforces AllowDebugAccess and returns forbiden replies for
// unauthorized requests.
func Protected(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !AllowDebugAccess(r) {
			msg := "debug access denied"
			if DevMode {
				ipStr, _, _ := net.SplitHostPort(r.RemoteAddr)
				msg += fmt.Sprintf("; to permit access, set TS_ALLOW_DEBUG_IP=%v", ipStr)
			}
			http.Error(w, msg, http.StatusForbidden)
			return
		}
		h.ServeHTTP(w, r)
	})
}

var timeStart = time.Now()

func Uptime() time.Duration { return time.Since(timeStart).Round(time.Second) }

// Port80Handler is the handler to be given to
// autocert.Manager.HTTPHandler.  The inner handler is the mux
// returned by NewMux containing registered /debug handlers.
type Port80Handler struct{ Main http.Handler }

func (h Port80Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.RequestURI
	if path == "/debug" || strings.HasPrefix(path, "/debug") {
		h.Main.ServeHTTP(w, r)
		return
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Use HTTPS", http.StatusBadRequest)
		return
	}
	if path == "/" && AllowDebugAccess(r) {
		// Redirect authorized user to the debug handler.
		path = "/debug/"
	}
	target := "https://" + stripPort(r.Host) + path
	http.Redirect(w, r, target, http.StatusFound)
}

func stripPort(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport
	}
	return net.JoinHostPort(host, "443")
}

// Handler is like net/http.Handler, but the handler can return an
// error instead of writing to its ResponseWriter.
type Handler interface {
	// ServeHTTPErr is like http.Handler.ServeHTTP, except that
	// it can choose to return an error instead of writing to its
	// http.ResponseWriter.
	//
	// If ServeHTTPErr returns an error, it caller should handle
	// an error by serving an HTTP 500 response to the user. The
	// error details should not be sent to the client, as they may
	// contain sensitive information. If the error is an
	// HTTPError, though, callers should use the HTTP response
	// code and message as the response to the client.
	ServeHTTPErr(http.ResponseWriter, *http.Request) error
}

// HandlerFunc is an adapter to allow the use of ordinary functions as
// Handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler that calls f.
type HandlerFunc func(http.ResponseWriter, *http.Request) error

func (h HandlerFunc) ServeHTTPErr(w http.ResponseWriter, r *http.Request) error {
	return h(w, r)
}

// StdHandler converts a Handler into a standard http.Handler.
// Handled requests are logged using logf, as are any errors. Errors
// are handled as specified by the Handler interface.
func StdHandler(h Handler, logf logger.Logf) http.Handler {
	return stdHandler(h, logf, time.Now)
}

func stdHandler(h Handler, logf logger.Logf, now func() time.Time) http.Handler {
	return handler{h, logf, now}
}

// handler is an http.Handler that wraps a Handler and handles errors.
type handler struct {
	h       Handler
	logf    logger.Logf
	timeNow func() time.Time
}

// ServeHTTP implements the http.Handler interface.
func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	msg := AccessLogRecord{
		When:       h.timeNow(),
		RemoteAddr: r.RemoteAddr,
		Proto:      r.Proto,
		TLS:        r.TLS != nil,
		Host:       r.Host,
		Method:     r.Method,
		RequestURI: r.URL.RequestURI(),
		UserAgent:  r.UserAgent(),
		Referer:    r.Referer(),
	}

	lw := loggingResponseWriter{ResponseWriter: w, logf: h.logf}
	err := h.h.ServeHTTPErr(&lw, r)
	hErr, hErrOK := err.(HTTPError)

	msg.Seconds = h.timeNow().Sub(msg.When).Seconds()
	msg.Code = lw.code
	msg.Bytes = lw.bytes

	switch {
	case lw.hijacked:
		// Connection no longer belongs to us, just log that we
		// switched protocols away from HTTP.
		if msg.Code == 0 {
			msg.Code = http.StatusSwitchingProtocols
		}
	case err != nil && r.Context().Err() == context.Canceled:
		msg.Code = 499 // nginx convention: Client Closed Request
		msg.Err = context.Canceled.Error()
	case hErrOK:
		// Handler asked us to send an error. Do so, if we haven't
		// already sent a response.
		msg.Err = hErr.Err.Error()
		if lw.code != 0 {
			h.logf("[unexpected] handler returned HTTPError %v, but already sent a response with code %d", hErr, lw.code)
			break
		}
		msg.Code = hErr.Code
		if msg.Code == 0 {
			h.logf("[unexpected] HTTPError %v did not contain an HTTP status code, sending internal server error", hErr)
			msg.Code = http.StatusInternalServerError
		}
		http.Error(&lw, hErr.Msg, msg.Code)
	case err != nil:
		// Handler returned a generic error. Serve an internal server
		// error, if necessary.
		msg.Err = err.Error()
		if lw.code == 0 {
			msg.Code = http.StatusInternalServerError
			http.Error(&lw, "internal server error", msg.Code)
		}
	case lw.code == 0:
		// Handler exited successfully, but didn't generate a
		// response. Synthesize an internal server error.
		msg.Code = http.StatusInternalServerError
		msg.Err = "[unexpected] handler did not respond to the client"
		http.Error(&lw, "internal server error", msg.Code)
	}

	// Cleanup below is common to all success and error paths. msg has
	// been populated with relevant information either way.

	// TODO(danderson): needed? Copied from existing code, but
	// doesn't HTTPServer do this by itself?
	if f, _ := w.(http.Flusher); !lw.hijacked && f != nil {
		f.Flush()
	}
	h.logf("%s", msg)
}

// loggingResponseWriter wraps a ResponseWriter and record the HTTP
// response code that gets sent, if any.
type loggingResponseWriter struct {
	http.ResponseWriter
	code     int
	bytes    int
	hijacked bool
	logf     logger.Logf
}

// WriteHeader implements http.Handler.
func (l *loggingResponseWriter) WriteHeader(statusCode int) {
	if l.code != 0 {
		l.logf("[unexpected] HTTP handler set statusCode twice (%d and %d)", l.code, statusCode)
		return
	}
	l.code = statusCode
	l.ResponseWriter.WriteHeader(statusCode)
}

// Write implements http.Handler.
func (l *loggingResponseWriter) Write(bs []byte) (int, error) {
	if l.code == 0 {
		l.code = 200
	}
	n, err := l.ResponseWriter.Write(bs)
	l.bytes += n
	return n, err
}

// Hijack implements http.Hijacker. Note that hijacking can still fail
// because the wrapped ResponseWriter is not required to implement
// Hijacker, as this breaks HTTP/2.
func (l *loggingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := l.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("ResponseWriter is not a Hijacker")
	}
	conn, buf, err := h.Hijack()
	if err == nil {
		l.hijacked = true
	}
	return conn, buf, err
}

func (l loggingResponseWriter) Flush() {
	f, _ := l.ResponseWriter.(http.Flusher)
	if f == nil {
		l.logf("[unexpected] tried to Flush a ResponseWriter that can't flush")
		return
	}
	f.Flush()
}

// HTTPError is an error with embedded HTTP response information.
//
// It is the error type to be (optionally) used by Handler.ServeHTTPErr.
type HTTPError struct {
	Code int    // HTTP response code to send to client; 0 means means 500
	Msg  string // Response body to send to client
	Err  error  // Detailed error to log on the server
}

// Error implements the error interface.
func (e HTTPError) Error() string { return fmt.Sprintf("httperror{%d, %q, %v}", e.Code, e.Msg, e.Err) }

// Error returns an HTTPError containing the given information.
func Error(code int, msg string, err error) HTTPError {
	return HTTPError{Code: code, Msg: msg, Err: err}
}

// varzHandler is an HTTP handler to write expvar values into the
// prometheus export format:
//
//   https://github.com/prometheus/docs/blob/master/content/docs/instrumenting/exposition_formats.md
//
// It makes the following assumptions:
//
//   * *expvar.Int are counters (unless marked as a gauge_; see below)
//   * a *tailscale/metrics.Set is descended into, joining keys with
//     underscores. So use underscores as your metric names.
//   * an expvar named starting with "gauge_" or "counter_" is of that
//     Prometheus type, and has that prefix stripped.
//   * anything else is untyped and thus not exported.
//   * expvar.Func can return an int or int64 (for now) and anything else
//     is not exported.
//
// This will evolve over time, or perhaps be replaced.
func varzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	var dump func(prefix string, kv expvar.KeyValue)
	dump = func(prefix string, kv expvar.KeyValue) {
		name := prefix + kv.Key

		var typ string
		switch {
		case strings.HasPrefix(kv.Key, "gauge_"):
			typ = "gauge"
			name = prefix + strings.TrimPrefix(kv.Key, "gauge_")

		case strings.HasPrefix(kv.Key, "counter_"):
			typ = "counter"
			name = prefix + strings.TrimPrefix(kv.Key, "counter_")
		}

		switch v := kv.Value.(type) {
		case *expvar.Int:
			if typ == "" {
				typ = "counter"
			}
			fmt.Fprintf(w, "# TYPE %s %s\n%s %v\n", name, typ, name, v.Value())
			return
		case *metrics.Set:
			v.Do(func(kv expvar.KeyValue) {
				dump(name+"_", kv)
			})
			return
		}

		if typ == "" {
			var funcRet string
			if f, ok := kv.Value.(expvar.Func); ok {
				v := f()
				if ms, ok := v.(runtime.MemStats); ok && name == "memstats" {
					writeMemstats(w, &ms)
					return
				}
				funcRet = fmt.Sprintf(" returning %T", v)
			}
			fmt.Fprintf(w, "# skipping expvar %q (Go type %T%s) with undeclared Prometheus type\n", name, kv.Value, funcRet)
			return
		}

		switch v := kv.Value.(type) {
		case expvar.Func:
			val := v()
			switch val.(type) {
			case int64, int:
				fmt.Fprintf(w, "# TYPE %s %s\n%s %v\n", name, typ, name, val)
			default:
				fmt.Fprintf(w, "# skipping expvar func %q returning unknown type %T\n", name, val)
			}

		case *metrics.LabelMap:
			fmt.Fprintf(w, "# TYPE %s %s\n", name, typ)
			// IntMap uses expvar.Map on the inside, which presorts
			// keys. The output ordering is deterministic.
			v.Do(func(kv expvar.KeyValue) {
				fmt.Fprintf(w, "%s{%s=%q} %v\n", name, v.Label, kv.Key, kv.Value)
			})
		}
	}
	expvar.Do(func(kv expvar.KeyValue) {
		dump("", kv)
	})
}

func writeMemstats(w io.Writer, ms *runtime.MemStats) {
	out := func(name, typ string, v uint64, help string) {
		if help != "" {
			fmt.Fprintf(w, "# HELP memstats_%s %s\n", name, help)
		}
		fmt.Fprintf(w, "# TYPE memstats_%s %s\nmemstats_%s %v\n", name, typ, name, v)
	}
	g := func(name string, v uint64, help string) { out(name, "gauge", v, help) }
	c := func(name string, v uint64, help string) { out(name, "counter", v, help) }
	g("heap_alloc", ms.HeapAlloc, "current bytes of allocated heap objects (up/down smoothly)")
	c("total_alloc", ms.TotalAlloc, "cumulative bytes allocated for heap objects")
	g("sys", ms.Sys, "total bytes of memory obtained from the OS")
	c("mallocs", ms.Mallocs, "cumulative count of heap objects allocated")
	c("frees", ms.Frees, "cumulative count of heap objects freed")
	c("num_gc", uint64(ms.NumGC), "number of completed GC cycles")
}
