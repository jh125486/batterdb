// Package repository provides the core data structures and methods for managing
// stacks in the batterdb application. It includes functionality for creating,
// interacting with, and managing stacks within a database.
//
// The package uses UUIDs for unique identification of stacks and employs mutex
// locks for concurrent access to shared data structures.
package repository

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Stack represents a stack data structure with metadata including
// creation, update, and read timestamps, and a reference to the database
// it belongs to. It supports concurrent access through a mutex lock.

type Stack struct {
	CreatedAt time.Time
	UpdatedAt time.Time
	ReadAt    time.Time
	database  *Database
	Name      string
	Data      []any
	mx        sync.RWMutex
	ID        uuid.UUID
}

// setUpdateTime sets the update and read timestamps of the stack to the given time.
func (s *Stack) setUpdateTime(t time.Time) {
	s.setReadTime(t)
	s.UpdatedAt = t
}

// setReadTime sets the read timestamp of the stack to the given time.
func (s *Stack) setReadTime(t time.Time) { s.ReadAt = t }

// Database returns the database the stack belongs to.
func (s *Stack) Database() *Database { return s.database }

// Push adds an element to the top of the stack and updates the timestamps.
func (s *Stack) Push(element any) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.setUpdateTime(time.Now())
	s.Data = append(s.Data, element)
	s.UpdatedAt = time.Now()
}

// Pop removes and returns the top element of the stack. It updates the timestamps.
func (s *Stack) Pop() any {
	s.mx.Lock()
	defer s.mx.Unlock()
	if len(s.Data) == 0 {
		s.setReadTime(time.Now())
		return nil
	}
	s.setUpdateTime(time.Now())
	res := s.Data[len(s.Data)-1]
	s.Data = s.Data[:len(s.Data)-1]

	return res
}

// Size returns the number of elements in the stack.
func (s *Stack) Size() int {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return len(s.Data)
}

// Peek returns the top element of the stack without removing it,
// and updates the read timestamp.
func (s *Stack) Peek() any {
	s.mx.RLock()
	defer s.mx.RUnlock()
	s.setReadTime(time.Now())
	if len(s.Data) == 0 {
		return nil
	}

	return s.Data[len(s.Data)-1]
}

// Flush removes all elements from the stack and updates the timestamps.
func (s *Stack) Flush() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.setUpdateTime(time.Now())
	s.Data = nil
}
