package repository_test

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/repository"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{
			name: "New repository",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := repository.New()
			require.NotNil(t, repo)
			require.NotNil(t, repo.Databases)
			require.Empty(t, repo.Databases)
		})
	}
}

func TestRepository_New(t *testing.T) {
	t.Parallel()
	type args struct {
		dbname string
	}
	tests := []struct {
		name    string
		setup   func() *repository.Repository
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "New database",
			setup: func() *repository.Repository {
				return repository.New()
			},
			args: args{
				dbname: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name: "database already exists",
			setup: func() *repository.Repository {
				r := repository.New()
				_, err := r.New("abcd")
				require.NoError(t, err)
				return r
			},
			args: args{
				dbname: "abcd",
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.setup()
			l := repo.Len()
			db, err := repo.New(tt.args.dbname)
			if tt.wantErr(t, err); err != nil {
				require.Equal(t, l, repo.Len())
				return
			}
			require.Equal(t, l+1, repo.Len())
			require.Equal(t, tt.args.dbname, db.Name)
		})
	}
}

func TestRepository_Len(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) *repository.Repository
		len   int
	}{
		{
			name: "Empty repository",
			setup: func(_ *testing.T) *repository.Repository {
				return repository.New()
			},
			len: 0,
		},
		{
			name: "Repository with one database",
			setup: func(t *testing.T) *repository.Repository {
				t.Helper()
				r := repository.New()
				_, err := r.New(t.Name())
				require.NoError(t, err)
				return r
			},
			len: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.setup(t)
			assert.Equal(t, tt.len, repo.Len())
		})
	}
}

func TestRepository_SortDatabases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(t *testing.T) *repository.Repository
		dbs   []string
	}{
		{
			name: "Sort two databases",
			setup: func(t *testing.T) *repository.Repository {
				t.Helper()
				r := repository.New()
				_, err := r.New("test2")
				require.NoError(t, err)
				_, err = r.New("test1")
				require.NoError(t, err)
				return r
			},
			dbs: []string{"test1", "test2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := tt.setup(t)
			db := repo.SortDatabases()
			for i, name := range tt.dbs {
				assert.Equal(t, name, db[i].Name)
			}
		})
	}
}

func TestRepository_Database(t *testing.T) {
	t.Parallel()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		setup   func(t *testing.T) *repository.Repository
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Get database",
			setup: func(t *testing.T) *repository.Repository {
				t.Helper()
				r := repository.New()
				_, err := r.New("abcd")
				require.NoError(t, err)
				return r
			},
			args: args{
				id: "abcd",
			},
			wantErr: assert.NoError,
		},
		{
			name: "No database found",
			setup: func(t *testing.T) *repository.Repository {
				t.Helper()
				r := repository.New()
				_, err := r.New("abcd")
				require.NoError(t, err)
				return r
			},
			args: args{
				id: "dne",
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			repo := tt.setup(t)
			db, err := repo.Database(tt.args.id)
			if tt.wantErr(t, err); err != nil {
				return
			}
			require.Equal(t, tt.args.id, db.Name)
			assert.NotEqual(t, uuid.Nil, db.ID)
			db, err = repo.Database(db.ID.String())
			require.NoError(t, err)
			require.Equal(t, tt.args.id, db.Name)
		})
	}
}

func TestRepository_Drop(t *testing.T) {
	t.Parallel()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		setup   func() *repository.Repository
		args    args
		wantErr require.ErrorAssertionFunc
		wantLen int
	}{
		{
			name: "Drop by name",
			setup: func() *repository.Repository {
				r := repository.New()
				_, err := r.New("abcde")
				require.NoError(t, err)
				return r
			},
			args: args{
				id: "abcde",
			},
			wantErr: require.NoError,
			wantLen: 0,
		},
		{
			name: "Does not exist",
			setup: func() *repository.Repository {
				r := repository.New()
				_, err := r.New("abcd")
				require.NoError(t, err)
				return r
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

			repo := tt.setup()
			tt.wantErr(t, repo.Drop(tt.args.id))
			assert.Equal(t, tt.wantLen, repo.Len())
		})
	}
}

func TestRepository_Persist(t *testing.T) {
	t.Parallel()
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "persist",
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name: "persist bad file",
			args: args{
				filename: "",
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := repository.New()
			tt.args.filename = filepath.Join(t.TempDir(), tt.args.filename)
			tt.wantErr(t, repo.Persist(tt.args.filename))
		})
	}
}

func TestRepository_Load(t *testing.T) {
	t.Parallel()

	persistedRepo := repository.New()
	for i := range 10 {
		_, err := persistedRepo.New("database" + strconv.Itoa(i))
		require.NoError(t, err)
	}
	persistedRepoFile := filepath.Join(t.TempDir(), t.Name())
	require.NoError(t, persistedRepo.Persist(persistedRepoFile))

	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
		wantDB  int
	}{
		{
			name: "load no file",
			args: args{
				filename: filepath.Join(t.TempDir(), "file"),
			},
			wantErr: assert.NoError,
			wantDB:  0,
		},
		{
			name: "load",
			args: args{
				filename: persistedRepoFile,
			},
			wantErr: assert.NoError,
			wantDB:  persistedRepo.Len(),
		},
		{
			name: "load bad file",
			args: args{
				filename: os.Args[0], // use the test binary as a bad file.
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := repository.New()
			err := repo.Load(tt.args.filename)
			if tt.wantErr(t, err); err != nil {
				return
			}
			require.Equal(t, tt.wantDB, repo.Len())
		})
	}
}
