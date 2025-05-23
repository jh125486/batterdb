// Package handlers provides the core HTTP service and middleware for the batterdb
// application. This includes setting up the HTTP server, configuring the API, handling
// secure connections, and managing the lifecycle of the service.
//
// The package utilizes the huma framework for API routing and Prometheus for metrics.
// It also supports self-signed TLS certificate generation for secure connections.
package handlers

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alecthomas/units"
	"github.com/arl/statsviz"
	"github.com/ccoveille/go-safecast"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	_ "github.com/danielgtaylor/huma/v2/formats/cbor" // Register the CBOR format.
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/jh125486/batterdb/formats/text"
	"github.com/jh125486/batterdb/formats/yaml"
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
	// Service represents the main service structure which holds the repository, API, server configuration,
	// build information, platform details, and other service-related configurations.
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
		secure    bool
		pid       int
	}

	// Option represents a configuration option for the Service.
	Option func(*Service)
)

// New creates a new instance of the Service with the provided options.
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

	// Crete the API with the app info, contact and formats.
	s.API = humago.New(mux, config(
		"BatterDB", "1.0.0", "A simple in-memory stack database.",
		&huma.Contact{
			Name:  "Jacob Hochstetler",
			URL:   "https://github.com/jh125486",
			Email: "jacob.hochstetler@gmail.com",
		},
		map[string]huma.Format{
			"application/json": huma.DefaultJSONFormat,
			"json":             huma.DefaultJSONFormat,
			"application/yaml": yaml.DefaultYAMLFormat(),
			"yaml":             yaml.DefaultYAMLFormat(),
			"plain/text":       text.DefaultTextFormat(),
			"text":             text.DefaultTextFormat(),
		},
	))

	// Register Prometheus metric.
	mux.Handle("/metrics", promhttp.Handler())

	// Register the API routes.
	s.AddRoutes(s.API)

	// Register statsviz.
	_ = statsviz.Register(mux)

	// Create the server.
	s.server = server(s.secure, mux)

	return s
}

func config(title, version, description string, contact *huma.Contact, formats map[string]huma.Format) huma.Config {
	schemaPrefix := "#/components/schemas/"
	schemasPath := "/schemas"

	registry := huma.NewMapRegistry(schemaPrefix, huma.DefaultSchemaNamer)

	return huma.Config{
		OpenAPI: &huma.OpenAPI{
			OpenAPI: "3.1.0",
			Info: &huma.Info{
				Title:       title,
				Version:     version,
				Description: description,
				Contact:     contact,
			},
			Components: &huma.Components{
				Schemas: registry,
			},
		},
		OpenAPIPath:   "/openapi",
		DocsPath:      "/docs",
		SchemasPath:   schemasPath,
		Formats:       formats,
		DefaultFormat: "application/json",
		CreateHooks: []func(huma.Config) huma.Config{
			func(c huma.Config) huma.Config {
				linkTransformer := huma.NewSchemaLinkTransformer(schemaPrefix, c.SchemasPath)
				c.OnAddOperation = append(c.OnAddOperation, linkTransformer.OnAddOperation)
				c.Transformers = append(c.Transformers, linkTransformer.Transform)
				return c
			},
		},
	}
}

// server creates a new HTTP server with optional TLS configuration.
func server(secure bool, mux *http.ServeMux) *http.Server {
	var tlsConfig *tls.Config
	if secure {
		cert, err := generateSelfSignedCert()
		if err != nil {
			slog.Error(err.Error())
			os.Exit(1)
		}
		tlsConfig = &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{cert},
		}
	}

	return &http.Server{
		Handler:        LoggingHandler(mux),
		TLSConfig:      tlsConfig,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: int(units.MiB),
	}
}

// WithBuildInfo sets the build information for the Service.
func WithBuildInfo(buildInfo *debug.BuildInfo) Option {
	return func(s *Service) {
		s.buildInfo = buildInfo
	}
}

// WithPort sets the port for the Service.
func WithPort(port int32) Option {
	return func(s *Service) {
		s.port.Store(port)
	}
}

// WithRepoFile sets the repository file for the Service.
func WithRepoFile(repofile string) Option {
	return func(s *Service) {
		s.savefile = repofile
	}
}

// WithSecure sets the secure flag for the Service.
func WithSecure(secure bool) Option {
	return func(s *Service) {
		s.secure = secure
	}
}

// WithPersistDB sets the persistDB flag for the Service.
func WithPersistDB(persist bool) Option {
	return func(s *Service) {
		s.persistDB = persist
	}
}

// AddRoutes registers the API routes for the Service.
func (s *Service) AddRoutes(api huma.API) {
	s.registerMain(api)
	s.registerDatabases(api)
	s.registerStacks(api)
}

// Port returns the current port the Service is running on.
func (s *Service) Port() int32 { return s.port.Load() }

// Start starts the Service and listens for incoming requests.
func (s *Service) Start() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port()))
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	// Save the actual port from the listener.
	port, err := safecast.ToInt32(l.Addr().(*net.TCPAddr).Port)
	if err != nil {
		return err
	}
	s.port.Store(port)
	s.server.Addr = fmt.Sprintf("localhost:%d", s.port.Load())

	if err := s.LoadToFile(); err != nil {
		return fmt.Errorf("failed to load repository: %w", err)
	}

	s.loadInitMsg()

	return s.serve(l)
}

// serve starts the HTTP or HTTPS server based on the secure flag.
func (s *Service) serve(l net.Listener) error {
	var err error
	if s.secure {
		err = s.server.ServeTLS(l, "", "")
	} else {
		err = s.server.Serve(l)
	}
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

// OpenAPI returns the OpenAPI spec as a string in the requested version.
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

// Shutdown gracefully shuts down the Service and saves the repository to file.
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

// registerMain registers the main API routes for the Service.
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

// registerDatabases registers the API routes for database operations.
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

// registerStacks registers the API routes for stack operations.
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

// registerStacksCRUD registers the CRUD API routes for stacks.
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

// loadInitMsg logs the initial message with service details when the service starts.
func (s *Service) loadInitMsg() {
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
	baseURL := "http://" + s.server.Addr
	if s.secure {
		baseURL = "https://" + s.server.Addr
	}
	slog.Info(fmt.Sprintf("Serving:      %v", baseURL))
	slog.Info(fmt.Sprintf("Docs:         %v/docs#/", baseURL))
	slog.Info(fmt.Sprintf("Metrics:      %v/metrics", baseURL))
	slog.Info(fmt.Sprintf("StatsViz:     %v/debug/statsviz", baseURL))
}

// SaveToFile saves the repository to a file if the persistDB flag is set.
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

// LoadToFile loads the repository from a file if the persistDB flag is set.
func (s *Service) LoadToFile() error {
	if !s.persistDB {
		return nil
	}
	return s.Repository.Load(s.savefile)
}

// generateSelfSignedCert generates a self-signed TLS certificate for secure connections.
func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate a new private key.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create a new random serial number for the certificate.
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create a simple certificate template.
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"github.com/jh125486"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour), // Valid for one year.
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		BasicConstraintsValid: true,
	}

	// Create a self-signed certificate.
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	// PEM encode the certificate and private key.
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	// Load the certificate and private key to create a tls.Certificate.
	return tls.X509KeyPair(certPEM, keyPEM)
}
