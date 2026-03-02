// batterdb is a stack-based database engine.
// Databases are created with a unique name and can contain multiple stacks.
// Stacks are created within a database and can contain multiple elements.
// Docs are served at /docs.
//
// In the case of **batterdb**, this way is by pushes **_Elements_** in **_Stacks_,** so you only have access to the _Element_ on top,
// keeping the rest of them underneath.

package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"github.com/alecthomas/kong"

	"github.com/jh125486/batterdb/cli"
)

// XXX add otel

func main() {
	// Listen for interrupt signals.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Read build info.
	info, ok := debug.ReadBuildInfo()
	if !ok {
		slog.ErrorContext(context.Background(), "couldn't read build info")
		os.Exit(1)
	}
	ctx, err := cli.New(
		os.Args[1:],
		kong.Name("batterdb"),
		kong.Description("A simple stacked-based database 🔋."),
		kong.Vars{"RepoFile": ".batterdb.gob"},
		kong.Vars{"version": info.Main.Version},
		kong.Bind(&cli.Ctx{
			Stop:      stop,
			BuildInfo: info,
			Writer:    os.Stdout,
		}),
	)
	if err != nil {
		slog.ErrorContext(context.Background(), "Failed to parse command line", slog.Any("error", err))
		os.Exit(1)
	}

	ctx.FatalIfErrorf(ctx.Run())
}
