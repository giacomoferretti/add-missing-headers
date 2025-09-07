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

package add_missing_headers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	add_missing_headers "github.com/giacomoferretti/add-missing-headers-traefik-plugin"
)

func TestRequestHeaders_StrictMode(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.StrictHeaderCheck = true
	cfg.RequestHeaders["X-Custom-Header"] = "custom-value"
	cfg.RequestHeaders["X-Default-Header"] = "default-value"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Verify headers were added correctly
		assertHeader(t, req, "X-Custom-Header", "custom-value")
		assertHeader(t, req, "X-Default-Header", "default-value")
		assertHeader(t, req, "X-Existing-Header", "existing-value")
		assertHeader(t, req, "X-Empty-Header", "") // Should remain empty in strict mode
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set existing headers to test they won't be overridden
	req.Header.Set("X-Existing-Header", "existing-value")
	req.Header.Set("X-Empty-Header", "") // Explicitly empty header

	handler.ServeHTTP(recorder, req)
}

func TestRequestHeaders_LooseMode(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.StrictHeaderCheck = false
	cfg.RequestHeaders["X-Custom-Header"] = "custom-value"
	cfg.RequestHeaders["X-Empty-Header"] = "filled-value"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Verify headers were added correctly
		assertHeader(t, req, "X-Custom-Header", "custom-value")
		assertHeader(t, req, "X-Empty-Header", "filled-value") // Should be overwritten in loose mode
		assertHeader(t, req, "X-Existing-Header", "existing-value")
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set existing headers to test behavior
	req.Header.Set("X-Existing-Header", "existing-value")
	req.Header.Set("X-Empty-Header", "") // Should be overwritten in loose mode

	handler.ServeHTTP(recorder, req)
}

func TestResponseHeaders_StrictMode(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.StrictHeaderCheck = true
	cfg.ResponseHeaders["X-Custom-Response"] = "custom-response"
	cfg.ResponseHeaders["X-Cache-Control"] = "no-cache"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Set some response headers before writing
		rw.Header().Set("X-Existing-Response", "existing")
		rw.Header().Set("X-Empty-Response", "") // Explicitly empty
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Check response headers
	assertResponseHeader(t, recorder, "X-Custom-Response", "custom-response")
	assertResponseHeader(t, recorder, "X-Cache-Control", "no-cache")
	assertResponseHeader(t, recorder, "X-Existing-Response", "existing")
	assertResponseHeader(t, recorder, "X-Empty-Response", "") // Should remain empty in strict mode
}

func TestResponseHeaders_LooseMode(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.StrictHeaderCheck = false
	cfg.ResponseHeaders["X-Custom-Response"] = "custom-response"
	cfg.ResponseHeaders["X-Empty-Response"] = "filled-response"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Set some response headers before writing
		rw.Header().Set("X-Existing-Response", "existing")
		rw.Header().Set("X-Empty-Response", "") // Should be overwritten in loose mode
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Check response headers
	assertResponseHeader(t, recorder, "X-Custom-Response", "custom-response")
	assertResponseHeader(t, recorder, "X-Empty-Response", "filled-response") // Should be overwritten
	assertResponseHeader(t, recorder, "X-Existing-Response", "existing")
}

func TestNoResponseHeaders(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.RequestHeaders["X-Request-Header"] = "request-value"
	// No response headers configured

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assertHeader(t, req, "X-Request-Header", "request-value")
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Should pass through directly when no response headers
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()

	// Test default values
	if cfg.StrictHeaderCheck != true {
		t.Error("Expected StrictHeaderCheck to default to true")
	}
	if cfg.DisableExplicitFlush != false {
		t.Error("Expected DisableExplicitFlush to default to false")
	}
	if len(cfg.RequestHeaders) != 0 {
		t.Error("Expected RequestHeaders to be empty by default")
	}
	if len(cfg.ResponseHeaders) != 0 {
		t.Error("Expected ResponseHeaders to be empty by default")
	}
	if len(cfg.BypassHeaders) != 0 {
		t.Error("Expected BypassHeaders to be empty by default")
	}
}

func TestBypassHeaders_HeaderPresence(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.RequestHeaders["X-Test-Header"] = "test-value"
	cfg.ResponseHeaders["X-Response-Header"] = "response-value"
	cfg.BypassHeaders["X-Accel-Buffering"] = "" // Empty value means check for presence only

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// This handler should be called without any header modifications
		if req.Header.Get("X-Test-Header") != "" {
			t.Error("Request headers should not be modified when bypassed")
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set the bypass header
	req.Header.Set("X-Accel-Buffering", "no")

	handler.ServeHTTP(recorder, req)

	// Check that response headers were not added
	if recorder.Header().Get("X-Response-Header") != "" {
		t.Error("Response headers should not be added when bypassed")
	}
}

func TestBypassHeaders_HeaderValue(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.RequestHeaders["X-Test-Header"] = "test-value"
	cfg.ResponseHeaders["X-Response-Header"] = "response-value"
	cfg.BypassHeaders["X-Skip-Processing"] = "true" // Specific value match

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// This handler should be called without any header modifications
		if req.Header.Get("X-Test-Header") != "" {
			t.Error("Request headers should not be modified when bypassed")
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set the bypass header with correct value
	req.Header.Set("X-Skip-Processing", "true")

	handler.ServeHTTP(recorder, req)

	// Check that response headers were not added
	if recorder.Header().Get("X-Response-Header") != "" {
		t.Error("Response headers should not be added when bypassed")
	}
}

func TestBypassHeaders_NoBypass(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.RequestHeaders["X-Test-Header"] = "test-value"
	cfg.ResponseHeaders["X-Response-Header"] = "response-value"
	cfg.BypassHeaders["X-Skip-Processing"] = "true"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Headers should be added in this case
		assertHeader(t, req, "X-Test-Header", "test-value")
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("response"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Set the bypass header with WRONG value (should not bypass)
	req.Header.Set("X-Skip-Processing", "false")

	handler.ServeHTTP(recorder, req)

	// Check that response headers were added
	assertResponseHeader(t, recorder, "X-Response-Header", "response-value")
}

func TestBypassHeaders_MultipleBypassConditions(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.RequestHeaders["X-Test-Header"] = "test-value"
	cfg.BypassHeaders["X-Accel-Buffering"] = ""     // Presence check
	cfg.BypassHeaders["X-Skip-Processing"] = "true" // Value check

	testCases := []struct {
		name         string
		headers      map[string]string
		shouldBypass bool
		description  string
	}{
		{
			name:         "Bypass on X-Accel-Buffering presence",
			headers:      map[string]string{"X-Accel-Buffering": "no"},
			shouldBypass: true,
			description:  "Should bypass when X-Accel-Buffering is present",
		},
		{
			name:         "Bypass on X-Skip-Processing value",
			headers:      map[string]string{"X-Skip-Processing": "true"},
			shouldBypass: true,
			description:  "Should bypass when X-Skip-Processing equals true",
		},
		{
			name:         "No bypass on wrong value",
			headers:      map[string]string{"X-Skip-Processing": "false"},
			shouldBypass: false,
			description:  "Should not bypass when X-Skip-Processing equals false",
		},
		{
			name:         "No bypass without headers",
			headers:      map[string]string{},
			shouldBypass: false,
			description:  "Should not bypass when no bypass headers are present",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				if tc.shouldBypass {
					if req.Header.Get("X-Test-Header") != "" {
						t.Error("Request headers should not be modified when bypassed")
					}
				} else {
					assertHeader(t, req, "X-Test-Header", "test-value")
				}
				rw.WriteHeader(http.StatusOK)
			})

			handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Set test headers
			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			handler.ServeHTTP(recorder, req)
		})
	}
}

func TestFlushingBehavior(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.DisableExplicitFlush = true
	cfg.ResponseHeaders["X-Test"] = "test"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("test"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Should work normally even with flushing disabled
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", recorder.Code)
	}
	assertResponseHeader(t, recorder, "X-Test", "test")
}

func TestDefaultStatusCodeBehavior(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.ResponseHeaders["X-Custom-Header"] = "custom-value"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Only call Write() without explicit WriteHeader() - should default to 200 OK
		_, _ = rw.Write([]byte("response without explicit status"))
	})

	handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	// Should default to 200 OK even when WriteHeader is not called explicitly
	if recorder.Code != http.StatusOK {
		t.Errorf("Expected default status 200 OK when WriteHeader not called, got %d", recorder.Code)
	}

	// Headers should still be added
	assertResponseHeader(t, recorder, "X-Custom-Header", "custom-value")

	// Response body should be preserved
	expectedBody := "response without explicit status"
	if recorder.Body.String() != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, recorder.Body.String())
	}
}

func TestExplicitStatusCodePreservation(t *testing.T) {
	cfg := add_missing_headers.CreateConfig()
	cfg.ResponseHeaders["X-Status-Test"] = "status-test"

	testCases := []struct {
		name         string
		statusCode   int
		expectedCode int
	}{
		{"Created", http.StatusCreated, http.StatusCreated},
		{"Not Found", http.StatusNotFound, http.StatusNotFound},
		{"Internal Server Error", http.StatusInternalServerError, http.StatusInternalServerError},
		{"Accepted", http.StatusAccepted, http.StatusAccepted},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(tc.statusCode)
				_, _ = rw.Write([]byte("test response"))
			})

			handler, err := add_missing_headers.New(ctx, next, cfg, "test-plugin")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", nil)
			if err != nil {
				t.Fatal(err)
			}

			handler.ServeHTTP(recorder, req)

			// Explicit status codes should be preserved
			if recorder.Code != tc.expectedCode {
				t.Errorf("Expected status %d, got %d", tc.expectedCode, recorder.Code)
			}

			// Headers should still be added regardless of status code
			assertResponseHeader(t, recorder, "X-Status-Test", "status-test")
		})
	}
}

func assertHeader(t *testing.T, req *http.Request, key, expected string) {
	t.Helper()
	actual := req.Header.Get(key)
	if actual != expected {
		t.Errorf("Request header %s: expected %q, got %q", key, expected, actual)
	}
}

func assertResponseHeader(t *testing.T, recorder *httptest.ResponseRecorder, key, expected string) {
	t.Helper()
	actual := recorder.Header().Get(key)
	if actual != expected {
		t.Errorf("Response header %s: expected %q, got %q", key, expected, actual)
	}
}
