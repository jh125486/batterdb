package handlers_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/jh125486/batterdb/handlers"
)

func TestService_DatabaseHandlers(t *testing.T) {
	t.Parallel()
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
			name:          "get zero databases",
			method:        http.MethodGet,
			path:          "/databases",
			expStatusCode: http.StatusOK,
			expBody: `{
			  "databases": [],
			  "number_of_databases": 0
			}`,
		},
		{
			name: "get multiple databases",
			setup: func(svc *handlers.Service) {
				_, err := svc.Repository.New("dbZ")
				require.NoError(t, err)
				_, err = svc.Repository.New("dbA")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				var err error
				s, err = sjson.Set(s, "databases.0.id", "ID1")
				require.NoError(t, err)
				s, err = sjson.Set(s, "databases.1.id", "ID2")
				require.NoError(t, err)
				return s
			},
			expBody: `{
			  "databases": [
				{
				  "id": "ID1",
				  "name": "dbA",
				  "number_of_stacks": 0
				},
				{
				  "id": "ID2",
				  "name": "dbZ",
				  "number_of_stacks": 0
				}
			  ],
			  "number_of_databases": 2
			}`,
		},
		{
			name: "get single database",
			setup: func(svc *handlers.Service) {
				_, err := svc.Repository.New("dbSingle")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases/dbSingle",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				var err error
				s, err = sjson.Set(s, "id", "ID")
				require.NoError(t, err)
				return s
			},
			expBody: `{
			  "id": "ID",
			  "name": "dbSingle",
			  "number_of_stacks": 0
			}`,
		},
		{
			name: "get single database dne",
			setup: func(svc *handlers.Service) {
				_, err := svc.Repository.New("dbSingle")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases/dne",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "database not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name:          "create a database",
			method:        http.MethodPost,
			path:          "/databases",
			query:         url.Values{"name": []string{"dbName123"}},
			expStatusCode: http.StatusCreated,
			processBody: func(s string) string {
				var err error
				s, err = sjson.Set(s, "id", "ID")
				require.NoError(t, err)
				return s
			},
			expBody: `{
			  "id": "ID",
			  "name": "dbName123",
			  "number_of_stacks": 0
			}`,
		},
		{
			name: "database already exists",
			setup: func(svc *handlers.Service) {
				_, err := svc.Repository.New("dbExists")
				require.NoError(t, err)
			},
			method:        http.MethodPost,
			path:          "/databases",
			query:         url.Values{"name": []string{"dbExists"}},
			expStatusCode: http.StatusConflict,
			expBody: `{
			  "title": "Conflict",
			  "status": 409,
			  "detail": "database already exists",
			  "errors": [
				{
				  "message": "already exists"
				}
			  ]
			}`,
		},
		{
			name: "delete a database",
			setup: func(svc *handlers.Service) {
				_, err := svc.Repository.New("dbName123")
				require.NoError(t, err)
			},
			method:        http.MethodDelete,
			path:          "/databases/dbName123",
			expStatusCode: http.StatusNoContent,
		},
		{
			name:          "delete a database dne",
			method:        http.MethodDelete,
			path:          "/databases/dne",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "database not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// setup.
			_, api := humatest.New(t)
			svc, _ := handlers.New()
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
			require.JSONEq(t, tt.expBody, body)
		})
	}
}
