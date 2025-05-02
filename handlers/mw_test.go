package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/handlers"
)

func TestLoggingHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		header     http.Header
		method     string
		path       string
		body       io.Reader
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     http.MethodGet,
			path:       "/",
			body:       http.NoBody,
			wantStatus: http.StatusOK,
		},
		{
			name:   "PATCH request",
			method: http.MethodPatch,
			path:   "/",
			body:   strings.NewReader("abcd"),
		},
		{
			name:       "POST request",
			method:     http.MethodPost,
			path:       "/",
			body:       strings.NewReader("abc"),
			wantStatus: http.StatusOK,
		},
		{
			name: "WS request",
			header: map[string][]string{
				"Upgrade": {"websocket"},
			},
			method:     http.MethodGet,
			path:       "/",
			body:       strings.NewReader("abc"),
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequestWithContext(context.TODO(), tt.method, tt.path, tt.body)
			require.NoError(t, err)
			for k, v := range tt.header {
				req.Header.Set(k, v[0])
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantStatus != 0 {
					w.WriteHeader(tt.wantStatus)
				}
				if r.Body != nil {
					b, _ := io.ReadAll(r.Body)
					defer func() {
						_ = r.Body.Close()
					}()
					_, _ = w.Write(b)
				}
			})

			loggingHandler := handlers.LoggingHandler(handler)
			loggingHandler.ServeHTTP(rr, req)
			if tt.wantStatus == 0 {
				tt.wantStatus = http.StatusOK
			}
			assert.Equal(t, tt.wantStatus, rr.Code)
		})
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		timeoutMs      string
		sleepMs        int
		expectedStatus int
	}{
		{
			name:           "no timeout param, completes normally",
			timeoutMs:      "",
			sleepMs:        50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "timeout=0, completes normally",
			timeoutMs:      "0",
			sleepMs:        50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "timeout sufficient, completes normally",
			timeoutMs:      "100",
			sleepMs:        50,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "timeout exceeded, returns 408",
			timeoutMs:      "50",
			sleepMs:        100,
			expectedStatus: http.StatusRequestTimeout,
		},
		{
			name:           "invalid timeout, returns 400",
			timeoutMs:      "invalid",
			sleepMs:        50,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "negative timeout, returns 400",
			timeoutMs:      "-50",
			sleepMs:        50,
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test handler that sleeps for the specified duration
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(tt.sleepMs) * time.Millisecond)
				w.WriteHeader(http.StatusOK)
			})

			// Wrap the handler with the TimeoutMiddleware
			middleware := handlers.TimeoutMiddleware(handler)

			// Create a test request
			url := "/"
			if tt.timeoutMs != "" {
				url = "/?timeout=" + tt.timeoutMs
			}
			req, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Create a test response recorder
			rec := httptest.NewRecorder()

			// Serve the request
			middleware.ServeHTTP(rec, req)

			// Check the status code
			if rec.Code != tt.expectedStatus {
				t.Errorf("Expected status code %d, got %d", tt.expectedStatus, rec.Code)
			}
		})
	}
}
