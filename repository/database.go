package repository

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Database struct {
	Stacks map[name]*Stack
	Name   string
	ID     uuid.UUID
	mx     sync.RWMutex
}

func (db *Database) Len() int {
	db.mx.RLock()
	defer db.mx.RUnlock()
	return len(db.Stacks)
}

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
