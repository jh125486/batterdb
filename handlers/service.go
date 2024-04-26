package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/alecthomas/units"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor" // Register the CBOR format.

	_ "github.com/jh125486/batterdb/formats/text" // Register the text format.
	_ "github.com/jh125486/batterdb/formats/yaml" // Register the YAML format.
	"github.com/jh125486/batterdb/repository"
)

const logo = ` 
______       _   _           ____________
| ___ \     | | | |          |  _  \ ___ \
| |_/ / __ _| |_| |_ ___ _ __| | | | |_/ /
| ___ \/ _' | __| __/ _ \ '__| | | | ___ \
| |_/ / (_| | |_| ||  __/ |  | |/ /| |_/ /
\____/ \__,_|\__|\__\___|_|  |___/ \____/
`

const repofile = ".repository.gob"

type Service struct {
	StartedAt  time.Time
	Repository *repository.Repository
	API        huma.API
	server     *http.Server
	Version    string
	GoVersion  string
	Platform   string
	PersistDB  bool
	PID        int
	Port       int
}

func New() *Service {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		panic("couldn't read build info")
	}

	s := &Service{
		Version:    info.Main.Version,
		GoVersion:  info.GoVersion,
		Platform:   fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
		PID:        os.Getpid(),
		StartedAt:  time.Now().UTC(),
		Repository: repository.New(),
	}

	mux := http.NewServeMux()
	config := huma.DefaultConfig("BatterDB", "1.0.0")
	s.API = humago.New(mux, config)
	s.AddRoutes(s.API)
	s.server = &http.Server{
		Handler:        LoggingHandler(mux),
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: int(units.MiB),
	}

	return s
}

func (s *Service) AddRoutes(api huma.API) {
	s.registerMain(api)
	s.registerDatabases(api)
	s.registerStacks(api)
}

func (s *Service) Start(port int) error {
	s.Port = port
	s.server.Addr = fmt.Sprintf("127.0.0.1:%d", s.Port)
	s.initMsg()
	if err := s.LoadRepoFromFile(repofile); err != nil {
		slog.Error("Failed to load repository", slog.String("err", err.Error()))
		os.Exit(1)
	}

	return s.server.ListenAndServe()
}

// OpenAPI return the OpenAPI spec as a string in the requested version.
func (s *Service) OpenAPI(openapi string) []byte {
	switch openapi {
	case "3.1":
		b, _ := s.API.OpenAPI().YAML()
		return b
	case "3.0.3":
		// Use downgrade to return OpenAPI 3.0.3 YAML since oapi-codegen doesn't
		// support OpenAPI 3.1 fully yet.
		b, _ := s.API.OpenAPI().DowngradeYAML()
		return b
	default:
		return nil
	}
}

func (s *Service) Shutdown() error {
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	return s.PersistRepoToFile(repofile)
}

func (s *Service) registerMain(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-status",
		Method:      http.MethodGet,
		Path:        "/_status",
		Summary:     "Status",
		Description: "Show server status.",
		Tags:        []string{"Main"},
	}, s.StatusHandler)
	huma.Register(api, huma.Operation{
		OperationID: "get-ping",
		Method:      http.MethodGet,
		Path:        "/_ping",
		Summary:     "Ping",
		Description: "Sends a ping to the server, that will answer pong if it is running.",
		Tags:        []string{"Main"},
	}, PingHandler)
}
func (s *Service) registerDatabases(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "post-database",
		Method:        http.MethodPost,
		Path:          "/databases",
		Summary:       "Create",
		DefaultStatus: http.StatusCreated,
		Description:   "Create a database.",
		Tags:          []string{"Databases"},
	}, s.CreateDatabaseHandler)
	huma.Register(api, huma.Operation{
		OperationID: "get-databases",
		Method:      http.MethodGet,
		Path:        "/databases",
		Summary:     "Databases",
		Description: "Show databases.",
		Tags:        []string{"Databases"},
	}, s.ListDatabasesHandler)
	huma.Register(api, huma.Operation{
		OperationID: "get-database",
		Method:      http.MethodGet,
		Path:        "/databases/{database}",
		Summary:     "Database",
		Description: "Show a database.",
		Tags:        []string{"Databases"},
	}, s.ShowDatabaseHandler)
	huma.Register(api, huma.Operation{
		OperationID: "delete-database",
		Method:      http.MethodDelete,
		Path:        "/databases/{database}",
		Summary:     "Delete",
		Description: "Delete a database.",
		Tags:        []string{"Databases"},
	}, s.DeleteDatabaseHandler)
}
func (s *Service) registerStacks(api huma.API) {
	s.registerStacksCRUD(api)
	huma.Register(api, huma.Operation{
		OperationID: "peek-stack",
		Method:      http.MethodGet,
		Path:        "/databases/{database}/stacks/{stack}/peek",
		Summary:     "Peek",
		Description: "`PEEK` operation on a stack.",
		Tags:        []string{"Stack Operations"},
	}, s.PeekDatabaseStackHandler)
	huma.Register(api, huma.Operation{
		OperationID: "push-stack",
		Method:      http.MethodPut,
		Path:        "/databases/{database}/stacks/{stack}",
		Summary:     "Push",
		Description: "`PUSH` operation on a stack.",
		Tags:        []string{"Stack Operations"},
	}, s.PushDatabaseStackHandler)
	huma.Register(api, huma.Operation{
		OperationID: "pop-stack",
		Method:      http.MethodDelete,
		Path:        "/databases/{database}/stacks/{stack}",
		Summary:     "Pop",
		Description: "`POP` operation on a stack.",
		Tags:        []string{"Stack Operations"},
	}, s.PopDatabaseStackHandler)
	huma.Register(api, huma.Operation{
		OperationID: "flush-stack",
		Method:      http.MethodDelete,
		Path:        "/databases/{database}/stacks/{stack}/flush",
		Summary:     "Flush",
		Description: "`FLUSH` operation on a stack.",
		Tags:        []string{"Stack Operations"},
	}, s.FlushDatabaseStackHandler)
}

func (s *Service) registerStacksCRUD(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-stack",
		Method:        http.MethodPost,
		Path:          "/databases/{database}/stacks",
		Summary:       "Create",
		Description:   "Create a stack from a database.",
		DefaultStatus: http.StatusCreated,
		Tags:          []string{"Stacks"},
	}, s.CreateDatabaseStackHandler)
	huma.Register(api, huma.Operation{
		OperationID: "get-stacks",
		Method:      http.MethodGet,
		Path:        "/databases/{database}/stacks",
		Summary:     "Stacks",
		Description: "Show stacks of a database.",
		Tags:        []string{"Stacks"},
	}, s.ListDatabaseStacksHandler)
	huma.Register(api, huma.Operation{
		OperationID: "get-stack",
		Method:      http.MethodGet,
		Path:        "/databases/{database}/stacks/{stack}",
		Summary:     "Stack",
		Description: "Show a stack of a database.",
		Tags:        []string{"Stacks"},
	}, s.ShowDatabaseStackHandler)
	huma.Register(api, huma.Operation{
		OperationID: "delete-stack",
		Method:      http.MethodDelete,
		Path:        `/databases/{database}/stacks/{stack}/nuke`,
		Summary:     "Delete",
		Description: "Delete a stack from a database.",
		Tags:        []string{"Stacks"},
	}, s.DeleteDatabaseStackHandler)
}

func (s *Service) initMsg() {
	for _, l := range strings.Split(logo, "\n") {
		slog.Info(l)
	}
	slog.Info(fmt.Sprintf("Version:      %v", s.Version))
	slog.Info(fmt.Sprintf("Go version:   %v", s.GoVersion))
	slog.Info(fmt.Sprintf("Host:         %v", s.Platform))
	slog.Info(fmt.Sprintf("Port:         %v", s.Port))
	slog.Info(fmt.Sprintf("PID:          %v", s.PID))
}

func (s *Service) PersistRepoToFile(filename string) error {
	if !s.PersistDB {
		return nil
	}
	return s.Repository.Persist(filename)
}

func (s *Service) LoadRepoFromFile(filename string) error {
	if !s.PersistDB {
		return nil
	}
	return s.Repository.Load(filename)
}
