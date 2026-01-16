package cli_test

import (
	"bytes"
	"debug/buildinfo"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jh125486/batterdb/cli"
)

func TestNew(t *testing.T) {
	t.Parallel()
	type args struct {
		args []string
		opts []kong.Option
	}
	tests := []struct {
		name    string
		args    args
		want    assert.ValueAssertionFunc
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "no var defined",
			want:    assert.Nil,
			wantErr: assert.Error,
		},
		{
			name: "invalid port",
			args: args{
				args: []string{"-p", "0"},
				opts: []kong.Option{
					kong.Vars{"RepoFile": ".batterdb.gob"},
				},
			},
			want:    assert.Nil,
			wantErr: assert.Error,
		},
		{
			name: "valid",
			args: args{
				opts: []kong.Option{
					kong.Vars{"RepoFile": ".batterdb.gob"},
				},
			},
			want:    assert.NotNil,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.args.opts = append(tt.args.opts, kong.Bind(&cli.Ctx{}))
			got, err := cli.New(tt.args.args, tt.args.opts...)
			tt.want(t, got)
			tt.wantErr(t, err)
		})
	}
}

func TestServerCmd_Run(t *testing.T) {
	t.Parallel()
	type args struct {
		ctx *cli.Ctx
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "no error",
			args: args{
				ctx: &cli.Ctx{
					BuildInfo: &buildinfo.BuildInfo{
						GoVersion: "RussCoxExpress",
						Main: debug.Module{
							Version: "v1.2.3",
						},
					},
					Writer: new(bytes.Buffer),
					Stop:   make(chan os.Signal, 1),
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.args.ctx.Stop = make(chan os.Signal, 1)

			c := cli.CLI{}
			require.NoError(t, c.AfterApply(tt.args.ctx))
			cmd := &cli.ServerCmd{}
			tt.args.ctx.Stop <- os.Interrupt
			tt.wantErr(t, cmd.Run(tt.args.ctx))
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestServerCmd_Run_Cases(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		port    int32
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "normal shutdown",
			port:    1205,
			wantErr: assert.NoError,
		},
		{
			name:    "start error",
			port:    -666, // invalid port to force Start() to error
			wantErr: assert.Error,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := &cli.Ctx{
				BuildInfo: &buildinfo.BuildInfo{
					GoVersion: "Test",
					Main:      debug.Module{Version: "v0.0.0"},
				},
				Writer: new(bytes.Buffer),
				Stop:   make(chan os.Signal, 1),
			}

			c := cli.CLI{}
			c.Port = tc.port
			require.NoError(t, c.AfterApply(ctx))
			cmd := &cli.ServerCmd{}

			// trigger shutdown
			ctx.Stop <- os.Interrupt
			tc.wantErr(t, cmd.Run(ctx))
			time.Sleep(100 * time.Millisecond)
		})
	}
}

func TestOpenAPICmd_Run(t *testing.T) {
	t.Parallel()
	type fields struct {
		Spec string
	}
	type args struct {
		ctx *cli.Ctx
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "valid",
			fields: fields{
				Spec: "3.1",
			},
			args: args{
				ctx: &cli.Ctx{
					Writer: new(bytes.Buffer),
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := cli.CLI{}
			require.NoError(t, c.AfterApply(tt.args.ctx))
			cmd := &cli.OpenAPICmd{
				Spec: tt.fields.Spec,
			}
			tt.wantErr(t, cmd.Run(tt.args.ctx))
		})
	}
}
