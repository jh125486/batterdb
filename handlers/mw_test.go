package handlers_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jh125486/batterdb/handlers"
)

func TestLoggingHandler(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req, err := http.NewRequestWithContext(context.TODO(), tt.method, tt.path, tt.body)
			if err != nil {
				t.Fatal(err)
			}

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.wantStatus != 0 {
					w.WriteHeader(tt.wantStatus)
				}
				if r.Body != nil {
					b, _ := io.ReadAll(r.Body)
					defer r.Body.Close()
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
