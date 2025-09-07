// Copyright 2025 Giacomo Ferretti
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package add_missing_headers provides a Traefik plugin for adding missing HTTP headers.
package add_missing_headers

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
)

// Config holds the plugin configuration.
type Config struct {
	RequestHeaders       map[string]string `yaml:"requestHeaders,omitempty"`
	ResponseHeaders      map[string]string `yaml:"responseHeaders,omitempty"`
	DisableExplicitFlush bool              `yaml:"disableExplicitFlush,omitempty"`
	StrictHeaderCheck    bool              `yaml:"strictHeaderCheck,omitempty"`
	BypassHeaders        map[string]string `yaml:"bypassHeaders,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		RequestHeaders:       make(map[string]string),
		ResponseHeaders:      make(map[string]string),
		DisableExplicitFlush: false,
		StrictHeaderCheck:    true, // Default to strict (only add if header doesn't exist)
		BypassHeaders:        make(map[string]string),
	}
}

// Plugin holds the necessary components of a Traefik plugin.
type Plugin struct {
	name                 string
	next                 http.Handler
	requestHeaders       map[string]string
	responseHeaders      map[string]string
	disableExplicitFlush bool
	strictHeaderCheck    bool
	bypassHeaders        map[string]string
}

// New instantiates and returns the required components used to handle an HTTP request.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	return &Plugin{
		name:                 name,
		next:                 next,
		requestHeaders:       config.RequestHeaders,
		responseHeaders:      config.ResponseHeaders,
		disableExplicitFlush: config.DisableExplicitFlush,
		strictHeaderCheck:    config.StrictHeaderCheck,
		bypassHeaders:        config.BypassHeaders,
	}, nil
}

// ServeHTTP implements the http.Handler interface.
func (p *Plugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Check if we should bypass the middleware
	if p.shouldBypass(req) {
		p.next.ServeHTTP(rw, req)
		return
	}

	// Add missing request headers
	p.addMissingHeaders(req.Header, p.requestHeaders)

	// If no response headers to add, pass through directly
	if len(p.responseHeaders) == 0 {
		p.next.ServeHTTP(rw, req)
		return
	}

	// Use response modifier to add missing response headers
	p.next.ServeHTTP(newResponseModifier(p.responseHeaders, p.disableExplicitFlush, p.strictHeaderCheck, rw), req)
}

// shouldAddHeader determines if a header should be added based on the strict check setting.
func shouldAddHeader(header http.Header, key string, strictCheck bool) bool {
	if strictCheck {
		// Strict: only add if header doesn't exist at all
		return header.Values(key) == nil
	}
	// Loose: add if header doesn't exist or is empty
	return header.Get(key) == ""
}

// shouldBypass determines if the middleware should be bypassed based on request headers.
func (p *Plugin) shouldBypass(req *http.Request) bool {
	for headerName, expectedValue := range p.bypassHeaders {
		actualValue := req.Header.Get(headerName)

		// If expectedValue is empty, bypass if header exists with any value
		if expectedValue == "" && req.Header.Values(headerName) != nil {
			return true
		}

		// If expectedValue is not empty, check for exact match
		if expectedValue != "" && actualValue == expectedValue {
			return true
		}
	}
	return false
}

// addMissingHeaders adds headers to the target header map if they don't already exist.
func (p *Plugin) addMissingHeaders(target http.Header, headers map[string]string) {
	for key, value := range headers {
		if shouldAddHeader(target, key, p.strictHeaderCheck) {
			target.Set(key, value)
		}
	}
}

// responseModifier wraps http.ResponseWriter to add missing response headers.
type responseModifier struct {
	rw                   http.ResponseWriter
	flusher              http.Flusher
	responseHeaders      map[string]string
	disableExplicitFlush bool
	strictHeaderCheck    bool
	headersSent          bool
	code                 int
}

// newResponseModifier creates a new response modifier.
func newResponseModifier(responseHeaders map[string]string, disableExplicitFlush bool, strictHeaderCheck bool, w http.ResponseWriter) http.ResponseWriter {
	rm := &responseModifier{
		rw:                   w,
		code:                 http.StatusOK,
		responseHeaders:      responseHeaders,
		disableExplicitFlush: disableExplicitFlush,
		strictHeaderCheck:    strictHeaderCheck,
	}

	// Check if the underlying ResponseWriter supports flushing
	if f, ok := w.(http.Flusher); ok {
		rm.flusher = f
	}

	return rm
}

// Header returns the header map that will be sent by WriteHeader.
func (r *responseModifier) Header() http.Header {
	return r.rw.Header()
}

// WriteHeader sends an HTTP response header with the provided status code.
func (r *responseModifier) WriteHeader(code int) {
	if r.headersSent {
		return
	}

	r.addMissingResponseHeaders()
	r.rw.WriteHeader(code)

	r.code = code
	r.headersSent = true
}

// addMissingResponseHeaders adds missing headers to the response.
func (r *responseModifier) addMissingResponseHeaders() {
	for key, value := range r.responseHeaders {
		if shouldAddHeader(r.rw.Header(), key, r.strictHeaderCheck) {
			r.rw.Header().Set(key, value)
		}
	}
}

// Write writes the data to the connection as part of an HTTP reply.
func (r *responseModifier) Write(b []byte) (int, error) {
	r.WriteHeader(r.code)

	n, err := r.rw.Write(b)

	// Explicitly flush after write if enabled and supported
	if !r.disableExplicitFlush && r.flusher != nil {
		r.flusher.Flush()
	}

	return n, err
}

// Hijack hijacks the connection if the underlying ResponseWriter supports hijacking.
func (r *responseModifier) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.rw.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("responseWriter does not support hijacking: %T", r.rw)
	}
	return hijacker.Hijack()
}

// Flush sends any buffered data to the client if flushing is supported.
func (r *responseModifier) Flush() {
	if r.flusher != nil {
		r.flusher.Flush()
	}
}
