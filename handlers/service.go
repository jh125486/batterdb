package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
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

type (
	Service struct {
		Repository *repository.Repository
		API        huma.API

		server    *http.Server
		startedAt time.Time
		buildInfo *debug.BuildInfo
		platform  string
		savefile  string
		port      atomic.Int32
		persistDB bool
		pid       int
	}
	Option func(*Service)
)

func New(opts ...Option) *Service {
	// defaults.
	s := &Service{
		platform:   fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH),
		pid:        os.Getpid(),
		startedAt:  time.Now().UTC(),
		Repository: repository.New(),
		savefile:   ".batterdb.gob",
	}
	for _, opt := range opts {
		opt(s)
	}

	mux := http.NewServeMux()
	config := huma.DefaultConfig("BatterDB", "1.0.0")
	config.Info.Contact = &huma.Contact{
		Name:  "Jacob Hochstetler",
		URL:   "https://github.com/jh125486",
		Email: "jacob.hochstetler@gmail.com",
	}
	config.Info.Description = "A simple in-memory stack database."

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

func WithBuildInfo(buildInfo *debug.BuildInfo) Option {
	return func(s *Service) {
		s.buildInfo = buildInfo
	}
}

func WithPort(port int32) Option {
	return func(s *Service) {
		s.port.Store(port)
	}
}

func WithRepoFile(repofile string) Option {
	return func(s *Service) {
		s.savefile = repofile
	}
}

func WithPersistDB(persist bool) Option {
	return func(s *Service) {
		s.persistDB = persist
	}
}

func (s *Service) AddRoutes(api huma.API) {
	s.registerMain(api)
	s.registerDatabases(api)
	s.registerStacks(api)
}

func (s *Service) Port() int32 { return s.port.Load() }

func (s *Service) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port()))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// Save the actual port from the listener.
	s.port.Store(int32(listener.Addr().(*net.TCPAddr).Port))
	s.server.Addr = fmt.Sprintf("127.0.0.1:%d", s.port.Load())

	if err := s.LoadToFile(); err != nil {
		return fmt.Errorf("failed to load repository: %w", err)
	}

	s.initMsg()

	if err := s.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
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

func (s *Service) Shutdown(ctx context.Context) error {
	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	if err := s.server.Shutdown(ctx); err != nil {
		return err
	}

	if err := s.SaveToFile(); err != nil {
		return err
	}

	return nil
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
	slog.Info(fmt.Sprintf("Version:      %v", s.buildInfo.Main.Version))
	slog.Info(fmt.Sprintf("Go version:   %v", s.buildInfo.GoVersion))
	slog.Info(fmt.Sprintf("Host:         %v", s.platform))
	slog.Info(fmt.Sprintf("Port:         %v", s.Port()))
	slog.Info(fmt.Sprintf("PID:          %v", s.pid))
	if s.persistDB {
		slog.Info(fmt.Sprintf("Loaded repo:  %v", s.savefile))
		slog.Info(fmt.Sprintf("Databases:    %v", s.Repository.Len()))
	}
	slog.Info(fmt.Sprintf("Serving:      http://%v", s.server.Addr))
	slog.Info(fmt.Sprintf("Docs:         http://%v/docs#/", s.server.Addr))
}

func (s *Service) SaveToFile() error {
	if !s.persistDB {
		return nil
	}
	if err := s.Repository.Persist(s.savefile); err != nil {
		return err
	}
	slog.Info("Repository saved to disk", slog.Int("databases", s.Repository.Len()))

	return nil
}

func (s *Service) LoadToFile() error {
	if !s.persistDB {
		return nil
	}
	return s.Repository.Load(s.savefile)
}
