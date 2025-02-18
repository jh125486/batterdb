// Package cli provides the command-line interface for the batterdb application.
// It includes functionality for starting the server, outputting the OpenAPI specification,
// and handling various command-line options such as setting the port, enabling HTTPS,
// and persisting the database to disk.
//
// The package uses the kong library for command-line parsing and integrates with the
// handlers package to manage the server service.
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

// Ctx represents the context for the CLI commands, including build information,
// service instance, writer for output, and a channel to handle OS signals.

type Ctx struct {
	*debug.BuildInfo
	service *handlers.Service
	io.Writer
	Stop chan os.Signal
}

//nolint:govet // Order of fields must be maintained for CLI help output.
type (
	// CLI defines the structure for the command-line interface, including options
	// for port, persistence, repository file, and HTTPS, as well as commands for
	// starting the server and outputting the OpenAPI specification.
	CLI struct {
		Port     int32  `short:"p" default:"1205"        help:"Port to listen on."`
		Store    bool   `short:"s"                       help:"Persist the database to disk."`
		RepoFile string `          default:"${RepoFile}" help:"The file to persist the database to."`
		Secure   bool   `short:"S"                       help:"Enable HTTPS."`

		Server ServerCmd `default:"1" help:"Start the server." cmd:""`

		OpenAPI OpenAPICmd `help:"Output the OpenAPI specification version." cmd:"" optional:""`

		Version kong.VersionFlag `short:"v" help:"Show version."`
	}

	// ServerCmd represents the command to start the server.
	ServerCmd struct {
	}

	// OpenAPICmd represents the command to output the OpenAPI specification.
	OpenAPICmd struct {
		Spec string `default:"3.1" help:"OpenAPI specification version." enum:"3.1,3.0.3"`
	}
)

// New initializes and parses the command-line arguments.
func New(args []string, opts ...kong.Option) (*kong.Context, error) {
	var cli CLI
	k, err := kong.New(&cli, opts...)
	if err != nil {
		return nil, err
	}

	return k.Parse(args)
}

// Validate validates the command-line options.
func (cmd *CLI) Validate() error {
	if cmd.Port < 1 || cmd.Port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}
	return nil
}

// AfterApply applies the command-line options to the context and initializes the service.
func (cmd *CLI) AfterApply(ctx *Ctx) error {
	ctx.service = handlers.New(
		handlers.WithBuildInfo(ctx.BuildInfo),
		handlers.WithPort(cmd.Port),
		handlers.WithPersistDB(cmd.Store),
		handlers.WithRepoFile(cmd.RepoFile),
		handlers.WithSecure(cmd.Secure),
	)

	return nil
}

// Run starts the server service in a goroutine and blocks until an OS signal is received,
// then it initiates a graceful shutdown of the service.
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

// Run outputs the OpenAPI specification to the context writer.
func (cmd *OpenAPICmd) Run(ctx *Ctx) error {
	_, err := ctx.Write(ctx.service.OpenAPI(cmd.Spec))
	return err
}
