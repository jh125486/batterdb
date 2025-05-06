package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/handlers"
	"github.com/jh125486/batterdb/repository"
)

func TestBatchOperationsHandler(t *testing.T) {
	repo := repository.New()
	
	// Create a test database and stack
	db, err := repo.New("testdb")
	require.NoError(t, err)
	
	stack, err := db.New("teststack")
	require.NoError(t, err)
	
	// Push a test value to the stack
	stack.Push("initial-value")
	
	service := handlers.New(func(s *handlers.Service) {
		s.Repository = repo
	})
	
	tests := []struct {
		name           string
		operations     []map[string]interface{}
		expectedStatus int
		expectedResults []map[string]interface{}
	}{
		{
			name: "mixed operations",
			operations: []map[string]interface{}{
				{
					"type":     "peek",
					"database": "testdb",
					"stack":    "teststack",
				},
				{
					"type":     "push",
					"database": "testdb",
					"stack":    "teststack",
					"element":  "new-value",
				},
				{
					"type":     "pop",
					"database": "testdb",
					"stack":    "teststack",
				},
				{
					"type":     "pop",
					"database": "testdb",
					"stack":    "teststack",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusOK),
					"result": "initial-value",
				},
				{
					"status": float64(http.StatusOK),
					"result": "new-value",
				},
				{
					"status": float64(http.StatusOK),
					"result": "new-value",
				},
				{
					"status": float64(http.StatusOK),
					"result": "initial-value",
				},
			},
		},
		{
			name: "database not found",
			operations: []map[string]interface{}{
				{
					"type":     "peek",
					"database": "nonexistent",
					"stack":    "teststack",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusNotFound),
					"error":  "Database not found: nonexistent",
				},
			},
		},
		{
			name: "stack not found",
			operations: []map[string]interface{}{
				{
					"type":     "peek",
					"database": "testdb",
					"stack":    "nonexistent",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusNotFound),
					"error":  "Stack not found: nonexistent",
				},
			},
		},
		{
			name: "push without element",
			operations: []map[string]interface{}{
				{
					"type":     "push",
					"database": "testdb",
					"stack":    "teststack",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusBadRequest),
					"error":  "Element is required for push operation",
				},
			},
		},
		{
			name: "unknown operation type",
			operations: []map[string]interface{}{
				{
					"type":     "unknown",
					"database": "testdb",
					"stack":    "teststack",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusBadRequest),
					"error":  "Unknown operation type: unknown",
				},
			},
		},
		{
			name: "flush operation",
			operations: []map[string]interface{}{
				{
					"type":     "push",
					"database": "testdb",
					"stack":    "teststack",
					"element":  "value-to-flush",
				},
				{
					"type":     "flush",
					"database": "testdb",
					"stack":    "teststack",
				},
				{
					"type":     "peek",
					"database": "testdb",
					"stack":    "teststack",
				},
			},
			expectedStatus: http.StatusOK,
			expectedResults: []map[string]interface{}{
				{
					"status": float64(http.StatusOK),
					"result": "value-to-flush",
				},
				{
					"status": float64(http.StatusOK),
					"result": map[string]interface{}{
						"id":    stack.ID.String(),
						"name":  "teststack",
						"peek":  nil,
						"size":  float64(0),
						"created_at": stack.CreatedAt.Format(time.RFC3339Nano),
						"updated_at": stack.UpdatedAt.Format(time.RFC3339Nano),
						"read_at":   stack.ReadAt.Format(time.RFC3339Nano),
					},
				},
				{
					"status": float64(http.StatusOK),
					"result": nil,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the request body
			reqBody := map[string]interface{}{
				"operations": tt.operations,
			}
			
			reqJSON, err := json.Marshal(reqBody)
			require.NoError(t, err)
			
			// Create a request
			req := httptest.NewRequest(http.MethodPost, "/batch", bytes.NewReader(reqJSON))
			req.Header.Set("Content-Type", "application/json")
			
			// Create a response recorder
			rec := httptest.NewRecorder()
			
			// Create a test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input handlers.BatchOperationsInput
				
				// Parse the JSON body
				err := json.NewDecoder(r.Body).Decode(&input.Body)
				require.NoError(t, err)
				
				// Call the handler
				output, err := service.BatchOperationsHandler(context.Background(), &input)
				require.NoError(t, err)
				
				// Write the response
				w.WriteHeader(http.StatusOK)
				err = json.NewEncoder(w).Encode(output.Body)
				require.NoError(t, err)
			})
			
			// Serve the request
			handler.ServeHTTP(rec, req)
			
			// Check the status code
			assert.Equal(t, tt.expectedStatus, rec.Code)
			
			// Check the response body
			var response struct {
				Results []map[string]interface{} `json:"results"`
			}
			
			err = json.Unmarshal(rec.Body.Bytes(), &response)
			require.NoError(t, err)
			
			// Compare the results
			require.Equal(t, len(tt.expectedResults), len(response.Results))
			
			for i, expectedResult := range tt.expectedResults {
				actualResult := response.Results[i]
				
				// Check status
				assert.Equal(t, expectedResult["status"], actualResult["status"])
				
				// Check result or error
				if _, ok := expectedResult["result"]; ok {
					assert.Equal(t, expectedResult["result"], actualResult["result"])
				} else if _, ok := expectedResult["error"]; ok {
					assert.Equal(t, expectedResult["error"], actualResult["error"])
				}
			}
		})
	}
} 