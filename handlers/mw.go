// Package handlers provides HTTP middleware for the batterdb application,
// including logging of HTTP requests and responses.
//
// The package includes functionality to wrap HTTP handlers with additional
// behaviors, such as logging request details and response status codes.
package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
)

// loggingResponseWriter is a custom HTTP response writer that captures the status code for logging purposes.
type loggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader captures the status code and calls the underlying ResponseWriter's WriteHeader method.
func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// Write ensures that the status code is set to 200 (OK) if no status code has been set before writing the response body.
func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.StatusCode == 0 {
		lrw.StatusCode = http.StatusOK
	}
	return lrw.ResponseWriter.Write(b)
}

// LoggingHandler is a middleware that logs HTTP requests and their response status codes.
// It bypasses logging for WebSocket upgrade requests.
func LoggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if this is a WebSocket upgrade request.
		if upgrade := r.Header.Get("Upgrade"); upgrade == "websocket" {
			// If it is, bypass the logging and pass the request directly to the next handler.
			h.ServeHTTP(w, r)
			return
		}

		// If it's not a WebSocket upgrade request, proceed with the logging as usual.
		lrw := &loggingResponseWriter{ResponseWriter: w}
		h.ServeHTTP(lrw, r)
		slog.Info(fmt.Sprintf("%s %v %d", r.Method, r.URL.Path, lrw.StatusCode))
	})
}
