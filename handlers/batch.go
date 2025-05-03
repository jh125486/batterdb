// Package handlers includes functionality for batch operations in batterdb.
// This allows executing multiple stack operations in a single API call.
package handlers

import (
	"context"
	"net/http"
)

type (
	// BatchOperation represents a single operation to be performed in a batch.
	BatchOperation struct {
		Type     string `json:"type" enum:"push,pop,peek,flush"`
		Database string `json:"database"`
		Stack    string `json:"stack"`
		Element  any    `json:"element,omitempty"`
	}

	// BatchOperationsInput represents the input for batch operations.
	BatchOperationsInput struct {
		Body struct {
			Operations []BatchOperation `json:"operations"`
		}
	}

	// BatchOperationResult represents the result of a single operation in a batch.
	BatchOperationResult struct {
		Status int  `json:"status"`
		Result any  `json:"result,omitempty"`
		Error  any  `json:"error,omitempty"`
	}

	// BatchOperationsOutput represents the output for batch operations.
	BatchOperationsOutput struct {
		Body struct {
			Results []BatchOperationResult `json:"results"`
		}
	}
)

// BatchOperationsHandler handles requests to execute multiple operations in a single batch.
// It processes each operation sequentially and returns the results.
func (s *Service) BatchOperationsHandler(_ context.Context, input *BatchOperationsInput) (*BatchOperationsOutput, error) {
	output := new(BatchOperationsOutput)
	output.Body.Results = make([]BatchOperationResult, len(input.Body.Operations))

	for i, op := range input.Body.Operations {
		var result BatchOperationResult

		// Find the database
		db, err := s.Repository.Database(op.Database)
		if err != nil {
			result.Status = http.StatusNotFound
			result.Error = "Database not found: " + op.Database
			output.Body.Results[i] = result
			continue
		}

		// Find the stack
		stack, err := db.Stack(op.Stack)
		if err != nil {
			result.Status = http.StatusNotFound
			result.Error = "Stack not found: " + op.Stack
			output.Body.Results[i] = result
			continue
		}

		// Execute the operation based on its type
		switch op.Type {
		case "push":
			if op.Element == nil {
				result.Status = http.StatusBadRequest
				result.Error = "Element is required for push operation"
				output.Body.Results[i] = result
				continue
			}
			stack.Push(op.Element)
			result.Status = http.StatusOK
			result.Result = op.Element

		case "pop":
			element := stack.Pop()
			if element == nil {
				result.Status = http.StatusNoContent
			} else {
				result.Status = http.StatusOK
				result.Result = element
			}

		case "peek":
			element := stack.Peek()
			result.Status = http.StatusOK
			result.Result = element

		case "flush":
			stack.Flush()
			result.Status = http.StatusOK
			result.Result = map[string]any{
				"id":        stack.ID.String(),
				"name":      stack.Name,
				"peek":      stack.Peek(),
				"size":      stack.Size(),
				"created_at": stack.CreatedAt,
				"updated_at": stack.UpdatedAt,
				"read_at":   stack.ReadAt,
			}

		default:
			result.Status = http.StatusBadRequest
			result.Error = "Unknown operation type: " + op.Type
		}

		output.Body.Results[i] = result
	}

	return output, nil
} 