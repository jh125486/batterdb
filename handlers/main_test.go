package handlers_test

import (
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/jh125486/batterdb/handlers"
)

func TestService_MainHandlers(t *testing.T) {
	t.Parallel()
	const (
		goVersion = "superCoolVer"
		version   = "v1.2.3"
	)
	tests := []struct {
		name          string
		setup         func(*handlers.Service)
		method        string
		path          string
		query         url.Values
		expStatusCode int
		processBody   func(string) string
		expBody       string
	}{
		{
			name:          "get status",
			method:        http.MethodGet,
			path:          "/_status",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				for k, v := range map[string]string{
					"started_at":        "$StartedAt",
					"host":              "$Host",
					"memory_alloc":      "$MemoryAlloc",
					"running_for":       "$RunningFor",
					"pid":               "$PID",
					"number_goroutines": "$NumberGoroutines",
				} {
					var err error
					s, err = sjson.Set(s, k, v)
					require.NoError(t, err)
				}
				return s
			},
			expBody: `{
			  "started_at": "$StartedAt",
			  "status": "OK",
			  "version": "` + version + `",
			  "go_version": "` + goVersion + `",
			  "host": "$Host",
			  "memory_alloc": "$MemoryAlloc",
			  "running_for": "$RunningFor",
			  "pid": "$PID",
			  "number_goroutines": "$NumberGoroutines"
			}`,
		},
		{
			name:          "get ping",
			method:        http.MethodGet,
			path:          "/_ping",
			expStatusCode: http.StatusOK,
			expBody:       `pong`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// setup.
			_, api := humatest.New(t)
			svc := handlers.New(handlers.WithBuildInfo(&debug.BuildInfo{
				GoVersion: goVersion,
				Main: debug.Module{
					Version: version,
				},
			}))
			svc.AddRoutes(api)
			if tt.setup != nil {
				tt.setup(svc)
			}
			if tt.query != nil {
				tt.path += "?" + tt.query.Encode()
			}

			// test.
			resp := api.Do(tt.method, tt.path)
			require.Equal(t, tt.expStatusCode, resp.Code)
			body := resp.Body.String()
			if tt.expBody == "" {
				require.Empty(t, body)
				return
			}
			if tt.processBody != nil {
				body = tt.processBody(body)
			}
			if strings.HasPrefix(tt.expBody, "[") || strings.HasPrefix(tt.expBody, "{") {
				require.JSONEq(t, tt.expBody, body)
			} else {
				require.Equal(t, tt.expBody, body)
			}
		})
	}
}
