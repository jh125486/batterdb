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
	StackInput struct {
		URLParamDatabaseID
		KV bool `query:"kv" default:"false"`
	}
	StacksOutput struct {
		Body struct {
			Stacks any `json:"stacks"`
		}
	}
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
	CreateDatabaseStackInput struct {
		URLParamDatabaseID
		Name string `query:"name" minLength:"7" required:"true"`
	}
	StackOutput struct {
		Body Stack `json:"stack"`
	}
)

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
	DatabaseStackInput struct {
		URLParamDatabaseID
		URLParamStackID
	}
	URLParamStackID struct {
		StackID string `path:"stack" doc:"can be the stack ID or name"`
	}
)

func (s *Service) ShowDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackOutput, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
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

type StackElement struct {
	Body struct {
		Element any `json:"element"`
	}
}

func (s *Service) PeekDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackElement, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}

	out := new(StackElement)
	out.Body.Element = stack.Peek()

	return out, nil
}

type PushDatabaseStackElementInput struct {
	Body struct {
		Element any `json:"element"`
	}
	DatabaseStackInput
}

func (s *Service) PushDatabaseStackHandler(_ context.Context, input *PushDatabaseStackElementInput) (*StackElement, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	stack.Push(input.Body.Element)
	out := new(StackElement)
	out.Body.Element = input.Body.Element

	return out, nil
}

type PopDatabaseStackElementOutput struct {
	Body struct {
		Element any `json:"element"`
	}
	Status int
}

func (s *Service) PopDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*PopDatabaseStackElementOutput, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
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

func (s *Service) FlushDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*StackOutput, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
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

func (s *Service) DeleteDatabaseStackHandler(_ context.Context, input *DatabaseStackInput) (*struct{}, error) {
	stack, err := s.stack(input.DatabaseID, input.StackID)
	if err != nil {
		return nil, err
	}
	if err := stack.Database().Drop(input.StackID); err != nil {
		return nil, err
	}

	return nil, nil
}

func (s *Service) stack(dbID, sID string) (*repository.Stack, error) {
	db, err := s.Repository.Database(dbID)
	if err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}
	stack, err := db.Stack(sID)
	if err != nil {
		return nil, huma.Error404NotFound("stack not found", err)
	}

	return stack, nil
}
