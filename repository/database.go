// Package repository provides the core data structures and methods for managing
// databases and stacks in the batterdb application. It includes functionality for
// creating, retrieving, sorting, and deleting stacks within a database.
//
// The package uses UUIDs for unique identification of databases and stacks,
// and employs mutex locks for concurrent access to shared data structures.
package repository

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Database represents a collection of stacks, identified by a unique UUID.
// It includes methods for managing stacks within the database.
type Database struct {
	Stacks map[name]*Stack
	Name   string
	ID     uuid.UUID
	mx     sync.RWMutex
}

// Len returns the number of stacks in the database.
func (db *Database) Len() int {
	db.mx.RLock()
	defer db.mx.RUnlock()
	return len(db.Stacks)
}

// SortStacks returns a sorted slice of stacks in the database, sorted by name.
func (db *Database) SortStacks() []*Stack {
	db.mx.RLock()
	defer db.mx.RUnlock()
	stacks := make([]*Stack, 0, len(db.Stacks))
	for _, stack := range db.Stacks {
		stacks = append(stacks, stack)
	}
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})

	return stacks
}

// Stack retrieves a stack by its ID or name. If the stack is not found, it returns an error.
func (db *Database) Stack(id string) (*Stack, error) {
	db.mx.RLock()
	defer db.mx.RUnlock()
	uid, err := uuid.Parse(id)
	if err != nil {
		// must be a name.
		if stack, ok := db.Stacks[name(id)]; ok {
			return stack, nil
		}
	}
	for _, stack := range db.Stacks {
		if stack.ID == uid {
			stack.database = db
			return stack, nil
		}
	}

	return nil, ErrNotFound
}

// New creates a new stack with the given name and adds it to the database.
// If a stack with the same name already exists, it returns an error.
func (db *Database) New(n string) (*Stack, error) {
	db.mx.Lock()
	defer db.mx.Unlock()
	if _, ok := db.Stacks[name(n)]; ok {
		return nil, ErrAlreadyExists
	}

	t := time.Now()
	stack := &Stack{
		ID:        uuid.New(),
		Name:      n,
		database:  db,
		CreatedAt: t,
		UpdatedAt: t,
		ReadAt:    t,
	}
	db.Stacks[name(n)] = stack

	return stack, nil
}

// Drop removes a stack from the database by its ID or name. If the stack is not found, it returns an error.
func (db *Database) Drop(id string) error {
	db.mx.Lock()
	defer db.mx.Unlock()
	for _, stack := range db.Stacks {
		if stack.ID.String() == id || stack.Name == id {
			delete(db.Stacks, name(stack.Name))
			return nil
		}
	}

	return ErrNotFound
}
