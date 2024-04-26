// batterdb is a stack-based database engine.
// Databases are created with a unique name and can contain multiple stacks.
// Stacks are created within a database and can contain multiple elements.
// Docs are served at /docs.

//In the case of **batterdb**, this way is by pushes **_Elements_** in **_Stacks_,** so you only have access to the _Element_ on top,
//keeping the rest of them underneath.

package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/jh125486/batterdb/handlers"
)

func main() {
	var (
		port      int
		persistDB bool
		openapi   string
	)
	flag.IntVar(&port, "port", 1205, "The port to listen on.")
	flag.BoolVar(&persistDB, "persist", false, "Persist the database to disk.")
	flag.StringVar(&openapi, "openapi", "", "Print the OpenAPI spec version: 3.1 and 3.0.3 available.")
	flag.Parse()

	svc := handlers.New(
		handlers.WithPort(port),
		handlers.WithPersistDB(persistDB),
	)

	if openapi != "" {
		_, _ = os.Stdout.Write(svc.OpenAPI(openapi))
		os.Exit(0)
	}

	// Listen for interrupt signals.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run the service in a goroutine so that it doesn't block.
	go func() {
		if err := svc.Start(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("failed to start service", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	// Block until we receive our signal.
	<-stop

	// Begin graceful shutdown.
	if err := svc.Shutdown(context.Background()); err != nil {
		slog.Error("failed to gracefully shutdown", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
