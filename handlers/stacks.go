// Package handlers provides HTTP handlers for managing stacks within databases
// in the batterdb application. The handlers include operations for listing,
// creating, showing, peeking, pushing, popping, flushing, and deleting stacks.
//
// The package utilizes the huma framework for handling HTTP requests and responses,
// and interacts with the repository package to perform stack operations within databases.
package handlers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jh125486/batterdb/repository"
)

type (
	// StackInput represents the input structure for listing stacks in a database.
	// It includes the database ID and an optional key-value query parameter.
	StackInput struct {
		URLParamDatabaseID
		KV bool `default:"false" query:"kv"`
	}

	// StacksOutput represents the output structure for listing stacks in a database.
	// It contains a list of stacks.
	StacksOutput struct {
		Body struct {
			Stacks any `json:"stacks"`
		}
	}

	// Stack represents the structure of a single stack, including its ID, name,
	// size, and timestamps for creation, update, and last read.
	Stack struct {
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		ReadAt    time.Time `json:"read_at"`
		Peek      any       `json:"peek"`
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		Size      int       `json:"size"`
	}
)

// ListDatabaseStacksHandler handles the request to list all stacks in a database.
// It retrieves the stacks from the repository and returns the list.
func (s *Service) ListDatabaseStacksHandler(_ context.Context, input *StackInput) (*StacksOutput, error) {
	db, err := s.Repository.Database(input.DatabaseID)
	if err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}
	out := new(StacksOutput)
	if input.KV {
		stacks := make(map[string]any)
		for _, stack := range db.SortStacks() {
			stacks[stack.Name] = stack.Peek()
		}
		out.Body.Stacks = stacks

		return out, nil
	}

	stacks := make([]any, db.Len())
	for i, stack := range db.SortStacks() {
		stacks[i] = Stack{
			ID:        stack.ID.String(),
			Name:      stack.Name,
			Peek:      stack.Peek(),
			Size:      stack.Size(),
			CreatedAt: stack.CreatedAt,
			UpdatedAt: stack.UpdatedAt,
			ReadAt:    stack.ReadAt,
		}
	}
	out.Body.Stacks = stacks

	return out, nil
}

type (
	// CreateDatabaseStackInput represents the input structure for creating a new stack in a database.
	// It includes the database ID and the name of the new stack.
	CreateDatabaseStackInput struct {
		URLParamDatabaseID
		Name string `minLength:"7" query:"name" required:"true"`
	}

	// StackOutput represents the output structure for operations involving a single stack.
	// It contains the details of the stack.
	StackOutput struct {
		Body Stack `json:"stack"`
	}
)

// CreateDatabaseStackHandler handles the request to create a new stack in a database.
// It creates the stack in the repository and returns its details.
// If the stack already exists, it returns a conflict error.
func (s *Service) CreateDatabaseStackHandler(_ context.Context, input *CreateDatabaseStackInput) (*StackOutput, error) {
	db, err := s.Repository.Database(input.DatabaseID)
	if err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}
	stack, err := db.New(input.Name)
	if errors.Is(err, repository.ErrAlreadyExists) {
		return nil, huma.Error409Conflict("stack already exists", err)
	}

	out := new(StackOutput)
	out.Body = Stack{
		ID:        stack.ID.String(),
		Name:      stack.Name,
		Peek:      stack.Peek(),
		Size:      stack.Size(),
		CreatedAt: stack.CreatedAt,
		UpdatedAt: stack.UpdatedAt,
		ReadAt:    stack.ReadAt,
	}

	return out, nil
}

type (
	// DatabaseStackInput represents the input structure for operations involving a single stack in a database.
	// It includes the database ID and the stack ID.
	DatabaseStackInput struct {
		URLParamDatabaseID
		URLParamStackID
	}

	// URLParamStackID represents the URL parameter for a stack ID, which can be either the stack ID or name.
	URLParamStackID struct {
		StackID string `doc:"can be the stack ID or name" path:"stack"`
	}
)

// ShowDatabaseStackHandler handles the request to show the details of a specific stack
// in a database. It retrieves the stack from the repository and returns its details.
func (s *Service) ShowDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackOutput, error) {
	_, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}

	out := new(StackOutput)
	out.Body = Stack{
		ID:        stack.ID.String(),
		Name:      stack.Name,
		Peek:      stack.Peek(),
		Size:      stack.Size(),
		CreatedAt: stack.CreatedAt,
		UpdatedAt: stack.UpdatedAt,
		ReadAt:    stack.ReadAt,
	}

	return out, nil
}

// StackElement represents the output structure for peeking at the top element of a stack.
// It contains the top element of the stack.
type StackElement struct {
	Body struct {
		Element any `json:"element"`
	}
}

// PeekDatabaseStackHandler handles the request to peek at the top element of a specific stack
// in a database. It retrieves the top element from the stack and returns it.
func (s *Service) PeekDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackElement, error) {
	_, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}

	out := new(StackElement)
	out.Body.Element = stack.Peek()

	return out, nil
}

// PushDatabaseStackElementInput represents the input structure for pushing a new element
// onto a stack in a database. It includes the database ID, stack ID, and the new element.
type PushDatabaseStackElementInput struct {
	Body struct {
		Element any `json:"element"`
	}
	DatabaseStackInput
}

// PushDatabaseStackHandler handles the request to push a new element onto a specific stack
// in a database. It adds the new element to the stack and returns the element.
func (s *Service) PushDatabaseStackHandler(_ context.Context, input *PushDatabaseStackElementInput) (*StackElement, error) {
	_, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	stack.Push(input.Body.Element)
	out := new(StackElement)
	out.Body.Element = input.Body.Element

	return out, nil
}

// PopDatabaseStackElementOutput represents the output structure for popping an element
// from a stack. It contains the popped element and the status code.
type PopDatabaseStackElementOutput struct {
	Body struct {
		Element any `json:"element"`
	}
	Status int
}

// PopDatabaseStackHandler handles the request to pop an element from a specific stack
// in a database. It removes the top element from the stack and returns it. If the stack
// is empty, it returns a no content status.
func (s *Service) PopDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*PopDatabaseStackElementOutput, error) {
	_, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	out := new(PopDatabaseStackElementOutput)

	v := stack.Pop()
	if v == nil {
		out.Status = http.StatusNoContent
		return out, nil
	}

	out.Status = http.StatusOK
	out.Body.Element = v

	return out, nil
}

// FlushDatabaseStackHandler handles the request to flush all elements from a specific stack
// in a database. It removes all elements from the stack and returns the details of the empty stack.
func (s *Service) FlushDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackOutput, error) {
	_, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	stack.Flush()

	out := new(StackOutput)
	out.Body = Stack{
		ID:        stack.ID.String(),
		Name:      stack.Name,
		Peek:      stack.Peek(),
		Size:      stack.Size(),
		CreatedAt: stack.CreatedAt,
		UpdatedAt: stack.UpdatedAt,
		ReadAt:    stack.ReadAt,
	}

	return out, nil
}

// DeleteDatabaseStackHandler handles the request to delete a specific stack from a database.
// It removes the stack from the repository.
func (s *Service) DeleteDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*struct{}, error) {
	db, stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	if err := db.Drop(stack.ID.String()); err != nil {
		return nil, err
	}

	return nil, nil
}

// stack retrieves a stack from the repository by its database ID and stack ID.
// If the stack or database is not found, it returns a not found error.
func (s *Service) stack(dbID, sID string) (*repository.Database, *repository.Stack, error) {
	db, err := s.Repository.Database(dbID)
	if err != nil {
		return nil, nil, huma.Error404NotFound("database not found", err)
	}
	stack, err := db.Stack(sID)
	if err != nil {
		return nil, nil, huma.Error404NotFound("stack not found", err)
	}

	return db, stack, nil
}
