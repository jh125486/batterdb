// Package handlers provides HTTP handlers for the batterdb application, including
// endpoints for retrieving application status and handling ping requests.
//
// The package utilizes the Go standard library and external libraries for handling
// HTTP requests and responses, as well as for managing application status information.
package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/alecthomas/units"
	"github.com/ccoveille/go-safecast/v2"
)

type (
	// StatusOutput represents the output structure for the StatusHandler.
	// It contains detailed information about the application status.
	StatusOutput struct {
		Body StatusBody
	}

	// StatusBody represents the structure of the status information, including
	// the start time, status code, version, Go version, host, memory allocation,
	// runtime duration, process ID, and number of goroutines.
	StatusBody struct {
		StartedAt        time.Time `json:"started_at"        yaml:"startedAt"`
		Code             string    `json:"status"            yaml:"code"`
		Version          string    `json:"version"           yaml:"version"`
		GoVersion        string    `json:"go_version"        yaml:"goVersion"`
		Host             string    `json:"host"              yaml:"host"`
		MemoryAlloc      string    `json:"memory_alloc"      yaml:"memoryAlloc"`
		RunningFor       float64   `json:"running_for"       yaml:"runningFor"`
		PID              int       `json:"pid"               yaml:"pid"`
		NumberGoroutines int       `json:"number_goroutines" yaml:"numberGoroutines"`
	}
)

// StatusHandler handles the request to retrieve the application status.
// It gathers various runtime statistics and returns them in the response.
func (s *Service) StatusHandler(_ context.Context, _ *struct{}) (*StatusOutput, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	allocs, err := safecast.Convert[int64](mem.Alloc)
	if err != nil {
		return nil, err
	}

	out := new(StatusOutput)
	out.Body.Code = http.StatusText(http.StatusOK)
	out.Body.Version = s.buildInfo.Main.Version
	out.Body.GoVersion = s.buildInfo.GoVersion
	out.Body.Host = s.platform
	out.Body.PID = s.pid
	out.Body.StartedAt = s.startedAt
	out.Body.RunningFor = time.Since(s.startedAt).Seconds()
	out.Body.NumberGoroutines = runtime.NumGoroutine()
	out.Body.MemoryAlloc = units.Base2Bytes(allocs).Round(1).String()

	return out, nil
}

// PingOutput represents the output structure for the PingHandler.
// It contains a plain text response.
type PingOutput struct {
	Body []byte `contentType:"text/plain"`
}

// PingHandler handles the request to check the application's availability.
// It returns a "pong" response in plain text.
func PingHandler(_ context.Context, _ *struct{}) (*PingOutput, error) {
	out := new(PingOutput)
	out.Body = []byte("pong")

	return out, nil
}
