package handlers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jh125486/batterdb/repository"
)

type (
	DatabasesOutput struct {
		Body struct {
			Databases         []Database `json:"databases"`
			NumberOfDatabases int        `json:"number_of_databases"`
		}
	}
	Database struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		NumberOfStacks int    `json:"number_of_stacks"`
	}
)

func (s *Service) ListDatabasesHandler(_ context.Context, _ *struct{}) (*DatabasesOutput, error) {
	out := new(DatabasesOutput)
	out.Body.NumberOfDatabases = s.Repository.Len()
	out.Body.Databases = make([]Database, 0, out.Body.NumberOfDatabases)
	for _, db := range s.Repository.SortDatabases() {
		out.Body.Databases = append(out.Body.Databases, Database{
			ID:             db.ID.String(),
			Name:           db.Name,
			NumberOfStacks: db.Len(),
		})
	}

	return out, nil
}

type (
	URLParamDatabaseID struct {
		DatabaseID string `doc:"can be the database ID or name" path:"database"`
	}
	SingleDatabaseInput struct {
		URLParamDatabaseID
	}
	DatabaseOutput struct {
		Body Database
	}
)

func (s *Service) ShowDatabaseHandler(_ context.Context, input *SingleDatabaseInput) (*DatabaseOutput, error) {
	db, err := s.database(input.DatabaseID)
	if err != nil {
		return nil, err
	}

	out := new(DatabaseOutput)
	out.Body = Database{
		ID:             db.ID.String(),
		Name:           db.Name,
		NumberOfStacks: db.Len(),
	}

	return out, nil
}

type (
	CreateDatabaseInput struct {
		Name string `minLength:"7" query:"name" required:"true"`
	}
	CreateDatabaseOutput struct {
		Body Database
	}
)

func (s *Service) CreateDatabaseHandler(_ context.Context, input *CreateDatabaseInput) (*CreateDatabaseOutput, error) {
	db, err := s.Repository.New(input.Name)
	if errors.Is(err, repository.ErrAlreadyExists) {
		return nil, huma.Error409Conflict("database already exists", err)
	}

	return &CreateDatabaseOutput{
		Body: Database{
			ID:             db.ID.String(),
			Name:           db.Name,
			NumberOfStacks: db.Len(),
		},
	}, nil
}

func (s *Service) DeleteDatabaseHandler(_ context.Context, input *SingleDatabaseInput) (*struct{}, error) {
	if err := s.Repository.Drop(input.DatabaseID); err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}

	return nil, nil
}

func (s *Service) database(dbID string) (*repository.Database, error) {
	db, err := s.Repository.Database(dbID)
	if err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}

	return db, nil
}
