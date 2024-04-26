package repository

import (
	"errors"
	"sort"
	"sync"

	"github.com/google/uuid"
)

type (
	Repository struct {
		Databases map[name]*Database
		mx        sync.RWMutex
	}
	name string
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

func New() *Repository {
	return &Repository{
		Databases: make(map[name]*Database),
	}
}

func (r *Repository) Len() int {
	r.mx.RLock()
	defer r.mx.RUnlock()
	return len(r.Databases)
}

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
