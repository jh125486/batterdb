package handlers_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/handlers"
	"github.com/jh125486/batterdb/repository"
)

func TestService_Start(t *testing.T) {
	t.Parallel()

	info, ok := debug.ReadBuildInfo()
	if !ok {
		t.Fatal("can't read build info")
	}

	canceledCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	cancel()

	test := []struct {
		name            string
		opts            []handlers.Option
		shutdownCtx     context.Context
		wait            time.Duration
		wantStartErr    assert.ErrorAssertionFunc
		wantShutdownErr assert.ErrorAssertionFunc
	}{
		{
			name: "no save",
			opts: []handlers.Option{
				handlers.WithPort(0),
			},
			shutdownCtx:     context.Background(),
			wait:            10 * time.Millisecond,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.NoError,
		},
		{
			name: "insane port",
			opts: []handlers.Option{
				handlers.WithPort(-666),
			},
			shutdownCtx:     context.Background(),
			wait:            10 * time.Millisecond,
			wantStartErr:    assert.Error,
			wantShutdownErr: assert.NoError,
		},
		{
			name: "save",
			opts: []handlers.Option{
				handlers.WithPersistDB(true),
				handlers.WithRepoFile(filepath.Join(t.TempDir(), t.Name())),
				handlers.WithPort(0),
			},
			shutdownCtx:     context.Background(),
			wait:            10 * time.Millisecond,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.NoError,
		},
		{
			name: "bad savefile",
			opts: []handlers.Option{
				handlers.WithPersistDB(true),
				handlers.WithPort(0),
				handlers.WithRepoFile(""),
			},
			shutdownCtx:     context.Background(),
			wait:            10 * time.Millisecond,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.Error,
		},
		{
			name:            "no wait for shutdown",
			shutdownCtx:     context.Background(),
			wait:            0,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.NoError,
		},
		{
			name:            "canceled context",
			shutdownCtx:     canceledCtx,
			wait:            100 * time.Millisecond,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.NoError,
		},
		{
			name: "secure",
			opts: []handlers.Option{
				handlers.WithSecure(true),
			},
			shutdownCtx:     canceledCtx,
			wait:            100 * time.Millisecond,
			wantStartErr:    assert.NoError,
			wantShutdownErr: assert.NoError,
		},
	}
	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.opts = append(tt.opts, handlers.WithBuildInfo(info))
			svc := handlers.New(tt.opts...)
			go func() {
				tt.wantStartErr(t, svc.Start())
			}()
			time.Sleep(tt.wait)
			tt.wantShutdownErr(t, svc.Shutdown(tt.shutdownCtx))
		})
	}
}

func TestService_SaveToFile(t *testing.T) {
	t.Parallel()
	type args struct {
		filename string
	}
	tests := []struct {
		name    string
		save    bool
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no save",
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name: "save",
			save: true,
			args: args{
				filename: "test",
			},
			wantErr: assert.NoError,
		},
		{
			name: "save bad file",
			save: true,
			args: args{
				filename: "",
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			svc := handlers.New(
				handlers.WithPersistDB(tt.save),
				handlers.WithRepoFile(filepath.Join(t.TempDir(), tt.args.filename)),
			)
			tt.wantErr(t, svc.SaveToFile())
		})
	}
}

func TestService_LoadToFile(t *testing.T) {
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
			name:    "save bad file",
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
			svc := handlers.New(
				handlers.WithPersistDB(tt.persist),
				handlers.WithRepoFile(tt.args.filename),
			)
			err := svc.LoadToFile()
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
