package cli

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/alecthomas/kong"

	"github.com/jh125486/batterdb/handlers"
)

type Ctx struct {
	*debug.BuildInfo
	service *handlers.Service
	io.Writer
	Stop chan os.Signal
}

//nolint:govet
type (
	CLI struct {
		Port     int32  `short:"p" default:"1205"        help:"Port to listen on."`
		Store    bool   `short:"s"                       help:"Persist the database to disk."`
		RepoFile string `          default:"${RepoFile}" help:"The file to persist the database to."`

		Server ServerCmd `default:"1" help:"Start the server." cmd:""`

		OpenAPI OpenAPICmd `help:"Output the OpenAPI specification version." cmd:"" optional:""`

		Version kong.VersionFlag `short:"v" help:"Show version."`
	}
	ServerCmd struct {
	}
	OpenAPICmd struct {
		Spec string `default:"3.1" help:"OpenAPI specification version." enum:"3.1,3.0.3"`
	}
)

func New(args []string, opts ...kong.Option) (*kong.Context, error) {
	var cli CLI
	k, err := kong.New(&cli, opts...)
	if err != nil {
		return nil, err
	}

	return k.Parse(args)
}

func (cmd *CLI) Validate() error {
	if cmd.Port < 1 || cmd.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

func (cmd *CLI) AfterApply(ctx *Ctx) error {
	ctx.service = handlers.New(
		handlers.WithBuildInfo(ctx.BuildInfo),
		handlers.WithPort(cmd.Port),
		handlers.WithPersistDB(cmd.Store),
		handlers.WithRepoFile(cmd.RepoFile),
	)

	return nil
}

func (cmd *ServerCmd) Run(ctx *Ctx) error {
	// Run the service in a goroutine so that it doesn't block.
	go func() {
		if err := ctx.service.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start service", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	// Block until we receive our signal.
	<-ctx.Stop

	// Begin graceful shutdown.
	return ctx.service.Shutdown(context.Background())
}

func (cmd *OpenAPICmd) Run(ctx *Ctx) error {
	_, err := ctx.Write(ctx.service.OpenAPI(cmd.Spec))
	return err
}
