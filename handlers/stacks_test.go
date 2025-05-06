package handlers_test

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/sjson"

	"github.com/jh125486/batterdb/handlers"
	"github.com/jh125486/batterdb/repository"
)

func TestService_StacksHandlers(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		setup         func(*repository.Database)
		method        string
		path          string
		query         url.Values
		body          map[string]any
		expStatusCode int
		processBody   func(string) string
		expBody       string
	}{
		{
			name:          "get stacks database dne",
			method:        http.MethodGet,
			path:          "/databases/{dne}/stacks",
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
			name:          "get zero stacks",
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks",
			expStatusCode: http.StatusOK,
			expBody: `{
			  "stacks": []
			}`,
		},
		{
			name: "get multiple stacks",
			setup: func(db *repository.Database) {
				_, err := db.New("stackZ")
				require.NoError(t, err)
				_, err = db.New("stackA")
				require.NoError(t, err)
				_, err = db.New("stackZZ")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				var err error
				for i := range 3 {
					id := strconv.Itoa(i)
					for k, v := range map[string]string{
						"created_at": "CreatedAt",
						"updated_at": "UpdatedAt",
						"read_at":    "ReadAt",
						"id":         "ID",
					} {
						s, err = sjson.Set(s, "stacks."+id+"."+k, v+id)
						require.NoError(t, err)
					}
				}
				return s
			},
			expBody: `{
			  "stacks": [
				{
				  "created_at": "CreatedAt0",
				  "updated_at": "UpdatedAt0",
				  "read_at": "ReadAt0",
				  "peek": null,
				  "id": "ID0",
				  "name": "stackA",
				  "size": 0
				},
				{
				  "created_at": "CreatedAt1",
				  "updated_at": "UpdatedAt1",
				  "read_at": "ReadAt1",
				  "peek": null,
				  "id": "ID1",
				  "name": "stackZ",
				  "size": 0
				},
				{
				  "created_at": "CreatedAt2",
				  "updated_at": "UpdatedAt2",
				  "read_at": "ReadAt2",
				  "peek": null,
				  "id": "ID2",
				  "name": "stackZZ",
				  "size": 0
				}
			  ]
			}`,
		},
		{
			name: "get multiple stacks kvp",
			setup: func(db *repository.Database) {
				s1, err := db.New("stackZ")
				require.NoError(t, err)
				s1.Push("v1")

				s2, err := db.New("stackA")
				require.NoError(t, err)
				s2.Push("v2")
				s2.Push("v2a")

				s3, err := db.New("stackZZ")
				require.NoError(t, err)
				s3.Push("v3")
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks",
			query:         map[string][]string{"kv": {"true"}},
			expStatusCode: http.StatusOK,
			expBody: `{
			  "stacks": {
				"stackA": "v2a",
				"stackZ": "v1",
				"stackZZ": "v3"
			  }
			}`,
		},
		{
			name: "get single stack",
			setup: func(db *repository.Database) {
				_, err := db.New("stackSingle")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks/stackSingle",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				var err error
				for k, v := range map[string]string{
					"created_at": "CreatedAt",
					"updated_at": "UpdatedAt",
					"read_at":    "ReadAt",
					"id":         "ID",
				} {
					s, err = sjson.Set(s, k, v)
					require.NoError(t, err)
				}
				return s
			},
			expBody: `{
			  "created_at": "CreatedAt",
			  "updated_at": "UpdatedAt",
			  "read_at": "ReadAt",
			  "peek": null,
			  "id": "ID",
			  "name": "stackSingle",
			  "size": 0
			}`,
		},
		{
			name: "get single stack dne",
			setup: func(db *repository.Database) {
				_, err := db.New("dbSingle")
				require.NoError(t, err)
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks/dne",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name: "peek single stack",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackSingle")
				require.NoError(t, err)
				stack.Push(map[string]any{"key": "value"})
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks/stackSingle/peek",
			expStatusCode: http.StatusOK,
			expBody: `{
			  "element": {
					"key": "value"
				}
			}`,
		},
		{
			name: "peek single stack dne",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackSingle")
				require.NoError(t, err)
				stack.Push(map[string]any{"key": "value"})
			},
			method:        http.MethodGet,
			path:          "/databases/{database}/stacks/dne/peek",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name: "push single stack",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackSingle")
				require.NoError(t, err)
				stack.Push(map[string]any{"key1": "value2"})
			},
			method: http.MethodPut,
			path:   "/databases/{database}/stacks/stackSingle",
			body: map[string]any{
				"element": map[string]any{
					"key2": "value2",
				},
			},
			expStatusCode: http.StatusOK,
			expBody: `{
			  "element": {
				"key2": "value2"
			  }
			}`,
		},
		{
			name: "push single stack dne",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackSingle")
				require.NoError(t, err)
				stack.Push(map[string]any{"key": "value"})
			},
			method: http.MethodPut,
			path:   "/databases/{database}/stacks/dne",
			body: map[string]any{
				"element": map[string]any{
					"key2": "value2",
				},
			},
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name:          "create a stack",
			method:        http.MethodPost,
			path:          "/databases/{database}/stacks",
			query:         url.Values{"name": []string{"stackName123"}},
			expStatusCode: http.StatusCreated,
			processBody: func(s string) string {
				var err error
				for k, v := range map[string]string{
					"created_at": "CreatedAt",
					"updated_at": "UpdatedAt",
					"read_at":    "ReadAt",
					"id":         "ID",
				} {
					s, err = sjson.Set(s, k, v)
					require.NoError(t, err)
				}
				return s
			},
			expBody: `{
			  "created_at": "CreatedAt",
			  "updated_at": "UpdatedAt",
			  "read_at": "ReadAt",
			  "peek": null,
			  "id": "ID",
			  "name": "stackName123",
			  "size": 0
			}`,
		},
		{
			name: "stack already exists",
			setup: func(db *repository.Database) {
				_, err := db.New("stackExists")
				require.NoError(t, err)
			},
			method:        http.MethodPost,
			path:          "/databases/{database}/stacks",
			query:         url.Values{"name": []string{"stackExists"}},
			expStatusCode: http.StatusConflict,
			expBody: `{
			  "title": "Conflict",
			  "status": 409,
			  "detail": "stack already exists",
			  "errors": [
				{
				  "message": "already exists"
				}
			  ]
			}`,
		},
		{
			name:          "create stack database dne",
			method:        http.MethodPost,
			path:          "/databases/{dne}/stacks",
			query:         url.Values{"name": []string{"stackName123"}},
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
			name: "pop a stack value",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				stack.Push(map[string]any{"this": "that"})
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/stackName123",
			expStatusCode: http.StatusOK,
			expBody: ` {
			  "element": {
				"this": "that"
			  }
			}`,
		},
		{
			name: "pop an empty stack",
			setup: func(db *repository.Database) {
				_, err := db.New("stackName123")
				require.NoError(t, err)
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/stackName123",
			expStatusCode: http.StatusNoContent,
		},
		{
			name:          "pop a stack dne",
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/dne",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name: "flush a stack",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				for i := range 10 {
					stack.Push(map[string]any{"this": i})
				}
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/stackName123/flush",
			expStatusCode: http.StatusOK,
			processBody: func(s string) string {
				var err error
				for k, v := range map[string]string{
					"created_at": "CreatedAt",
					"updated_at": "UpdatedAt",
					"read_at":    "ReadAt",
					"id":         "ID",
				} {
					s, err = sjson.Set(s, k, v)
					require.NoError(t, err)
				}
				return s
			},
			expBody: `{
			  "created_at": "CreatedAt",
			  "updated_at": "UpdatedAt",
			  "read_at": "ReadAt",
			  "peek": null,
			  "id": "ID",
			  "name": "stackName123",
			  "size": 0
			}`,
		},
		{
			name: "flush a stack dne",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				stack.Push(map[string]any{"this": "that"})
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/dne/flush",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name: "delete a stack",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				for i := range 10 {
					stack.Push(map[string]any{"this": i})
				}
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/stackName123/nuke",
			expStatusCode: http.StatusNoContent,
		},
		{
			name: "delete a stack dne",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				stack.Push(map[string]any{"this": "that"})
			},
			method:        http.MethodDelete,
			path:          "/databases/{database}/stacks/dne/nuke",
			expStatusCode: http.StatusNotFound,
			expBody: `{
			  "title": "Not Found",
			  "status": 404,
			  "detail": "stack not found",
			  "errors": [
				{
				  "message": "not found"
				}
			  ]
			}`,
		},
		{
			name: "delete a stack database dne",
			setup: func(db *repository.Database) {
				stack, err := db.New("stackName123")
				require.NoError(t, err)
				stack.Push(map[string]any{"this": "that"})
			},
			method:        http.MethodDelete,
			path:          "/databases/{dne}/stacks/stackName123/nuke",
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
			svc := handlers.New()
			svc.AddRoutes(api)
			if tt.query != nil {
				tt.path += "?" + tt.query.Encode()
			}
			db, err := svc.Repository.New("dbName123")
			require.NoError(t, err)
			if tt.setup != nil {
				tt.setup(db)
			}
			tt.path = strings.ReplaceAll(tt.path, "{database}", db.ID.String())

			// test.
			resp := api.Do(tt.method, tt.path, tt.body)
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
