package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/alecthomas/units"
)

type (
	StatusOutput struct {
		Body StatusBody
	}
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

func (s *Service) StatusHandler(_ context.Context, _ *struct{}) (*StatusOutput, error) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	out := new(StatusOutput)
	out.Body.Code = http.StatusText(http.StatusOK)
	out.Body.Version = s.buildInfo.Main.Version
	out.Body.GoVersion = s.buildInfo.GoVersion
	out.Body.Host = s.platform
	out.Body.PID = s.pid
	out.Body.StartedAt = s.startedAt
	out.Body.RunningFor = time.Since(s.startedAt).Seconds()
	out.Body.NumberGoroutines = runtime.NumGoroutine()
	out.Body.MemoryAlloc = units.Base2Bytes(mem.Alloc).Round(1).String()

	return out, nil
}

type PingOutput struct {
	Body []byte `contentType:"text/plain"`
}

func PingHandler(_ context.Context, _ *struct{}) (*PingOutput, error) {
	out := new(PingOutput)
	out.Body = []byte("pong")

	return out, nil
}
