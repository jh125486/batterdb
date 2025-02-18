// Package handlers provides HTTP handlers for managing database operations
// within the batterdb application. The handlers include listing all databases,
// showing a specific database, creating a new database, and deleting an existing
// database.
//
// The package utilizes the huma framework for handling HTTP requests and responses,
// and interacts with the repository package to perform database operations.
package handlers

import (
	"context"
	"errors"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jh125486/batterdb/repository"
)

type (
	// DatabasesOutput represents the output structure for the ListDatabasesHandler.
	// It contains a list of databases and the total number of databases.
	DatabasesOutput struct {
		Body struct {
			Databases         []Database `json:"databases"`
			NumberOfDatabases int        `json:"number_of_databases"`
		}
	}

	// Database represents the structure of a single database, including its ID, name,
	// and the number of stacks it contains.
	Database struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		NumberOfStacks int    `json:"number_of_stacks"`
	}
)

// ListDatabasesHandler handles the request to list all databases.
// It retrieves the list of databases from the repository, sorts them for stability,
// and returns the list along with the total number of databases.
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
	// URLParamDatabaseID represents the URL parameter for a database ID, which can
	// be either the database ID or name.
	URLParamDatabaseID struct {
		DatabaseID string `doc:"can be the database ID or name" path:"database"`
	}

	// SingleDatabaseInput represents the input structure for the ShowDatabaseHandler
	// and DeleteDatabaseHandler, containing the database ID.
	SingleDatabaseInput struct {
		URLParamDatabaseID
	}

	// DatabaseOutput represents the output structure for the ShowDatabaseHandler,
	// containing the details of a single database.
	DatabaseOutput struct {
		Body Database
	}
)

// ShowDatabaseHandler handles the request to show the details of a specific database.
// It retrieves the database from the repository and returns its details.
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
	// CreateDatabaseInput represents the REST input request for the CreateDatabaseHandler.
	CreateDatabaseInput struct {
		Name string `minLength:"7" query:"name" required:"true"`
	}

	// CreateDatabaseOutput represents the REST output response for the CreateDatabaseHandler.
	CreateDatabaseOutput struct {
		Body Database
	}
)

// CreateDatabaseHandler handles the request to create a new database.
// It creates the database in the repository and returns its details.
// If the database already exists, it returns a conflict error.
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

// DeleteDatabaseHandler handles the request to delete an existing database.
// It removes the database from the repository.
// If the database is not found, it returns a not found error.
func (s *Service) DeleteDatabaseHandler(_ context.Context, input *SingleDatabaseInput) (*struct{}, error) {
	if err := s.Repository.Drop(input.DatabaseID); err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}

	return nil, nil
}

// database retrieves a database from the repository by its ID. If the database
// is not found, it returns a not found error.
func (s *Service) database(dbID string) (*repository.Database, error) {
	db, err := s.Repository.Database(dbID)
	if err != nil {
		return nil, huma.Error404NotFound("database not found", err)
	}

	return db, nil
}
