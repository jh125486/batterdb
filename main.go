package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/danielgtaylor/huma/v2"

	"github.com/jh125486/batterdb/handlers"
)

// XXX Readme -> How to genearate a client from the OpenAPI spec.
// coverage badge
// godoc badge
// save the OpenAPI spec to a file with GH action

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

	svc, api := handlers.New()
	svc.PersistDB = persistDB // super lazy options setting here.

	openAPI(openapi, api)

	// Listen for interrupt signals.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Run the service in a goroutine so that it doesn't block.
	go func() {
		if err := svc.Start(port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listening error", slog.String("err", err.Error()))
			os.Exit(1)
		}
	}()

	// Block until we receive our signal.
	<-stop

	// Begin graceful shutdown.
	if err := svc.Shutdown(); err != nil {
		slog.Error("failed to gracefully shutdown", slog.String("err", err.Error()))
		os.Exit(1)
	}
}

// openAPI is slightly icky, but I just want to output the OpenAPI spec if set and exit.
func openAPI(openapi string, api huma.API) {
	switch openapi {
	case "3.1":
		b, _ := api.OpenAPI().YAML()
		fmt.Println(string(b))
	case "3.0.3":
		// Use downgrade to return OpenAPI 3.0.3 YAML since oapi-codegen doesn't
		// support OpenAPI 3.1 fully yet.
		b, _ := api.OpenAPI().DowngradeYAML()
		fmt.Println(string(b))
	default:
		return
	}

	os.Exit(0)
}
