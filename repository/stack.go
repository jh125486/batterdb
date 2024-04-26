package repository

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

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

func (s *Stack) setUpdateTime(t time.Time) {
	s.setReadTime(t)
	s.UpdatedAt = t
}
func (s *Stack) setReadTime(t time.Time) { s.ReadAt = t }

func (s *Stack) Database() *Database { return s.database }

func (s *Stack) Push(element any) {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.setUpdateTime(time.Now())
	s.Data = append(s.Data, element)
	s.UpdatedAt = time.Now()
}

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

func (s *Stack) Size() int {
	s.mx.RLock()
	defer s.mx.RUnlock()
	return len(s.Data)
}

func (s *Stack) Peek() any {
	s.mx.RLock()
	defer s.mx.RUnlock()
	s.setReadTime(time.Now())
	if len(s.Data) == 0 {
		return nil
	}
	return s.Data[len(s.Data)-1]
}

func (s *Stack) Flush() {
	s.mx.Lock()
	defer s.mx.Unlock()
	s.setUpdateTime(time.Now())
	s.Data = nil
}
