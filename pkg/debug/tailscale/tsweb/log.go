// Copyright (c) 2020 Tailscale Inc & AUTHORS All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tsweb

import (
	"encoding/json"
	"strings"
	"time"
)

// AccessLogRecord is a record of one HTTP request served.
type AccessLogRecord struct {
	// Timestamp at which request processing started.
	When time.Time `json:"when"`
	// Time it took to finish processing the request. It does not
	// include the entire lifetime of the underlying connection in
	// cases like connection hijacking, only the lifetime of the HTTP
	// request handler.
	Seconds float64 `json:"duration"`

	// The client's ip:port.
	RemoteAddr string `json:"remote_addr"`
	// The HTTP protocol version, usually "HTTP/1.1 or HTTP/2".
	Proto string `json:"proto"`
	// Whether the request was received over TLS.
	TLS bool `json:"tls"`
	// The target hostname in the request.
	Host string `json:"host"`
	// The HTTP method invoked.
	Method string `json:"method"`
	// The unescaped request URI, including query parameters.
	RequestURI string `json:"request_uri"`

	// The client's user-agent
	UserAgent string `json:"user_agent"`
	// Where the client was before making this request.
	Referer string `json:"referer"`

	// The HTTP response code sent to the client.
	Code int `json:"code"`
	// Number of bytes sent in response body to client. If the request
	// was hijacked, only includes bytes sent up to the point of
	// hijacking.
	Bytes int `json:"bytes"`
	// Error encountered during request processing.
	Err string `json:"err"`
}

// String returns m as a JSON string.
func (m AccessLogRecord) String() string {
	if m.When.IsZero() {
		m.When = time.Now()
	}
	var buf strings.Builder
	json.NewEncoder(&buf).Encode(m)
	return strings.TrimRight(buf.String(), "\n")
}
