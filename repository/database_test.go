package repository_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/repository"
)

func TestDatabase_Len(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *repository.Database
		want  int
	}{
		{
			name: "zero",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				return db
			},
			want: 0,
		},
		{
			name: "one",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				_, err = db.New("test")
				require.NoError(t, err)
				return db
			},
			want: 1,
		},
		{
			name: "many",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				for range 1 << 10 {
					_, err = db.New(uuid.NewString())
					require.NoError(t, err)
				}
				return db
			},
			want: 1_024,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := tt.setup()
			assert.Equal(t, tt.want, db.Len())
		})
	}
}

func TestDatabase_SortStacks(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func() *repository.Database
		want  []string
	}{
		{
			name: "zero",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				return db
			},
			want: []string{},
		},
		{
			name: "few",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				for _, n := range []string{"zzzz", "aaaa", "bbbb", "cccc"} {
					_, err = db.New(n)
					require.NoError(t, err)
				}
				return db
			},
			want: []string{
				"aaaa",
				"bbbb",
				"cccc",
				"zzzz",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := tt.setup()
			stacks := db.SortStacks()
			require.Len(t, stacks, len(tt.want))

			for i, name := range tt.want {
				assert.Equal(t, name, stacks[i].Name)
			}
		})
	}
}

func TestDatabase_Stack(t *testing.T) {
	t.Parallel()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		setup   func() *repository.Database
		args    args
		wantErr require.ErrorAssertionFunc
		want    string
	}{
		{
			name: "dne",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				return db
			},
			args: args{
				id: "dne",
			},
			wantErr: require.Error,
		},
		{
			name: "found",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				for _, n := range []string{"zzzz", "aaaa", "bbbb", "cccc"} {
					_, err = db.New(n)
					require.NoError(t, err)
				}
				return db
			},
			args: args{
				id: "bbbb",
			},
			wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			db := tt.setup()

			stack, err := db.Stack(tt.args.id)
			if tt.wantErr(t, err); err == nil {
				assert.Equal(t, tt.args.id, stack.Name)
				stack, err = db.Stack(stack.ID.String())
				require.NoError(t, err)
				require.Equal(t, tt.args.id, stack.Name)
			}
		})
	}
}

func TestDatabase_New(t *testing.T) {
	t.Parallel()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		setup   func() *repository.Database
		args    args
		wantErr require.ErrorAssertionFunc
		want    string
	}{
		{
			name: "already exists",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New(t.Name())
				require.NoError(t, err)
				_, err = db.New("abcd")
				require.NoError(t, err)
				return db
			},
			args: args{
				id: "abcd",
			},
			wantErr: require.Error,
		},
		{
			name: "found",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("abcd")
				require.NoError(t, err)
				for _, n := range []string{"zzzz", "aaaa", "bbbb", "cccc"} {
					_, err = db.New(n)
					require.NoError(t, err)
				}
				return db
			},
			args: args{
				id: "newone",
			},
			wantErr: require.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setup()
			l := db.Len()
			stack, err := db.New(tt.args.id)
			if tt.wantErr(t, err); err == nil {
				l++
				require.Equal(t, tt.args.id, stack.Name)
			}
			require.Equal(t, l, db.Len())
		})
	}
}

func TestDatabase_Drop(t *testing.T) {
	t.Parallel()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		setup   func() *repository.Database
		args    args
		wantErr require.ErrorAssertionFunc
		wantLen int
	}{
		{
			name: "Drop by name",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("test")
				require.NoError(t, err)
				_, err = db.New("abcde")
				require.NoError(t, err)
				return db
			},
			args: args{
				id: "abcde",
			},
			wantErr: require.NoError,
			wantLen: 0,
		},
		{
			name: "Does not exist",
			setup: func() *repository.Database {
				r := repository.New()
				db, err := r.New("test")
				require.NoError(t, err)
				_, err = db.New("abcde")
				require.NoError(t, err)
				return db
			},
			args: args{
				id: "dne",
			},
			wantErr: require.Error,
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db := tt.setup()
			tt.wantErr(t, db.Drop(tt.args.id))
			assert.Equal(t, tt.wantLen, db.Len())
		})
	}
}
