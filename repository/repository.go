// Package repository provides the core data structures and methods for managing
// databases and stacks in the batterdb application. It includes functionality for
// creating, retrieving, sorting, and deleting databases, as well as persisting
// and loading the repository state to and from a file.
//
// The package uses UUIDs for unique identification of databases and stacks,
// and employs mutex locks for concurrent access to shared data structures.
package repository

import (
	"encoding/gob"
	"errors"
	"log/slog"
	"os"
	"sort"
	"sync"

	"github.com/google/uuid"
)

type (
	// Repository represents a collection of databases.
	// It includes methods for managing databases within the repository.
	Repository struct {
		Databases map[name]*Database
		mx        sync.RWMutex
	}

	// name represents the name of a database or stack.
	name string
)

var (
	// ErrNotFound is returned when a database or stack is not found.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists is returned when a database or stack already exists.
	ErrAlreadyExists = errors.New("already exists")
)

// New creates a new instance of Repository.
func New() *Repository {
	return &Repository{
		Databases: make(map[name]*Database),
	}
}

// Len returns the number of databases in the repository.
func (r *Repository) Len() int {
	r.mx.RLock()
	defer r.mx.RUnlock()
	return len(r.Databases)
}

// SortDatabases returns a sorted slice of databases in the repository, sorted by name.
func (r *Repository) SortDatabases() []*Database {
	r.mx.RLock()
	defer r.mx.RUnlock()
	dbs := make([]*Database, 0, len(r.Databases))
	for _, db := range r.Databases {
		dbs = append(dbs, db)
	}
	sort.Slice(dbs, func(i, j int) bool {
		return dbs[i].Name < dbs[j].Name
	})

	return dbs
}

// Database retrieves a database by its ID or name. If the database is not found, it returns an error.
func (r *Repository) Database(id string) (*Database, error) {
	r.mx.RLock()
	defer r.mx.RUnlock()
	uid, err := uuid.Parse(id)
	if err != nil {
		// must be a name.
		if db, ok := r.Databases[name(id)]; ok {
			return db, nil
		}
	}

	for _, db := range r.Databases {
		if db.ID == uid {
			return db, nil
		}
	}

	return nil, ErrNotFound
}

// New creates a new database with the given name and adds it to the repository.
// If a database with the same name already exists, it returns an error.
func (r *Repository) New(n string) (*Database, error) {
	r.mx.Lock()
	defer r.mx.Unlock()
	if _, ok := r.Databases[name(n)]; ok {
		return nil, ErrAlreadyExists
	}

	db := &Database{
		ID:     uuid.New(),
		Name:   n,
		Stacks: make(map[name]*Stack),
	}
	r.Databases[name(n)] = db

	return db, nil
}

// Drop removes a database from the repository by its ID or name. If the database is not found, it returns an error.
func (r *Repository) Drop(id string) error {
	r.mx.Lock()
	defer r.mx.Unlock()
	for _, db := range r.Databases {
		if db.ID.String() == id || db.Name == id {
			delete(r.Databases, name(db.Name))
			return nil
		}
	}

	return ErrNotFound
}

// Persist saves the repository state to a file.
func (r *Repository) Persist(filename string) error {
	r.mx.RLock()
	defer r.mx.RUnlock()

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()

	return gob.NewEncoder(file).Encode(r)
}

// Load loads the repository state from a file.
func (r *Repository) Load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet.
			slog.Info("No repository file found", slog.String("filename", filename))
			return nil
		}

		return err
	}
	defer func() {
		_ = file.Close()
	}()

	return gob.NewDecoder(file).Decode(r)
}
