package handlers_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/handlers"
	"github.com/jh125486/batterdb/repository"
)

func TestService_Start(t *testing.T) {
	t.Parallel()
	svc := handlers.New()
	go func() {
		_ = svc.Start(0)
	}()
	require.NoError(t, svc.Shutdown())
}

func TestService_PersistRepoToFile(t *testing.T) {
	t.Parallel()
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		persist bool
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no persist",
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "persist",
			persist: true,
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name:    "persist bad file",
			persist: true,
			args: args{
				filename: "",
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := handlers.New()
			svc.PersistDB = tt.persist
			tt.args.filename = filepath.Join(t.TempDir(), tt.args.filename)
			tt.wantErr(t, svc.PersistRepoToFile(tt.args.filename))
		})
	}
}

func TestService_LoadRepoFromFile(t *testing.T) {
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
		persist bool
		args    args
		wantErr assert.ErrorAssertionFunc
		wantDB  int
	}{
		{
			name: "no load",
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
			wantDB:  0,
		},
		{
			name:    "dne load",
			persist: true,
			args: args{
				filename: t.Name(),
			},
			wantErr: assert.NoError,
			wantDB:  0,
		},
		{
			name:    "exists load",
			persist: true,
			args: args{
				filename: persistedRepoFile,
			},
			wantErr: assert.NoError,
			wantDB:  10,
		},
		{
			name:    "persist bad file",
			persist: true,
			args: args{
				filename: os.Args[0], // use the test binary as a bad file.
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := handlers.New()
			svc.PersistDB = tt.persist
			err := svc.LoadRepoFromFile(tt.args.filename)
			if tt.wantErr(t, err); err != nil {
				return
			}
			require.Equal(t, tt.wantDB, svc.Repository.Len())
		})
	}
}

func TestService_OpenAPI(t *testing.T) {
	t.Parallel()

	svc := handlers.New()
	huma.Register(svc.API, huma.Operation{
		Method:  http.MethodGet,
		Path:    "/test-path",
		Tags:    []string{"test-tag"},
		Summary: "test summary",
	}, func(_ context.Context, _ *struct{}) (*struct{}, error) {
		return nil, nil
	})
	mustBytes := func(b []byte, err error) []byte {
		require.NoError(t, err)
		return b
	}
	type args struct {
		openapi string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "blank",
			want: nil,
		},
		{
			name: "3.1",
			args: args{
				openapi: "3.1",
			},
			want: mustBytes(svc.API.OpenAPI().YAML()),
		},
		{
			name: "3.0.3",
			args: args{
				openapi: "3.0.3",
			},
			want: mustBytes(svc.API.OpenAPI().DowngradeYAML()),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, svc.OpenAPI(tt.args.openapi))
		})
	}
}
