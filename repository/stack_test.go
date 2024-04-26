package repository_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/repository"
)

func TestStack_Database(t *testing.T) {
	t.Parallel()

	db, err := repository.New().New("test")
	require.NoError(t, err)
	stack, err := db.New("test")
	require.NoError(t, err)
	require.Equal(t, db.ID, stack.Database().ID)
}

func TestStack_Push(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stack *repository.Stack
		item  int
		want  []any
	}{
		{
			name:  "push to empty stack",
			stack: &repository.Stack{},
			item:  1,
			want:  []any{1},
		},
		{
			name:  "push to non-empty stack",
			stack: &repository.Stack{Data: []any{1, 2, 3}},
			item:  4,
			want:  []any{1, 2, 3, 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.stack.Push(tt.item)
			assert.Equal(t, tt.want, tt.stack.Data)
		})
	}
}

func TestStack_Pop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		stack     *repository.Stack
		wantItem  any
		wantStack []any
	}{
		{
			name:      "pop from stack with one item",
			stack:     &repository.Stack{Data: []any{1}},
			wantItem:  1,
			wantStack: []any{},
		},
		{
			name:      "pop from stack with multiple items",
			stack:     &repository.Stack{Data: []any{1, 2, 3}},
			wantItem:  3,
			wantStack: []any{1, 2},
		},
		{
			name:      "pop from empty stack",
			stack:     &repository.Stack{Data: []any{}},
			wantItem:  nil,
			wantStack: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			item := tt.stack.Pop()
			assert.Equal(t, tt.wantItem, item)
			assert.Equal(t, tt.wantStack, tt.stack.Data)
		})
	}
}

func TestStack_Size(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stack *repository.Stack
		want  int
	}{
		{
			name:  "size of empty stack",
			stack: &repository.Stack{},
			want:  0,
		},
		{
			name:  "size of non-empty stack",
			stack: &repository.Stack{Data: []any{1, 2, 3}},
			want:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.stack.Size())
		})
	}
}

func TestStack_Peek(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stack *repository.Stack
		want  any
	}{
		{
			name:  "peek from empty stack",
			stack: &repository.Stack{},
			want:  nil,
		},
		{
			name:  "peek from non-empty stack",
			stack: &repository.Stack{Data: []any{1, 2, 3}},
			want:  3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.stack.Peek())
		})
	}
}

func TestStack_Flush(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		stack *repository.Stack
	}{
		{
			name:  "flush empty stack",
			stack: &repository.Stack{},
		},
		{
			name:  "flush non-empty stack",
			stack: &repository.Stack{Data: []any{1, 2, 3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.stack.Flush()
			assert.Empty(t, tt.stack.Data)
		})
	}
}
