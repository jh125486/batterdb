package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
