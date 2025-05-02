// Package handlers provides HTTP middleware for the batterdb application,
// including logging of HTTP requests and responses.
//
// The package includes functionality to wrap HTTP handlers with additional
// behaviors, such as logging request details and response status codes.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"
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

// TimeoutMiddleware creates a middleware that adds a timeout to the request context
// based on the 'timeout' query parameter. If no timeout is specified, the default is used.
// The timeout is specified in milliseconds. A value of 0 means no timeout.
func TimeoutMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeoutParam := r.URL.Query().Get("timeout")
		if timeoutParam == "" {
			// No timeout specified, use the default by just passing through
			next.ServeHTTP(w, r)
			return
		}

		// Parse the timeout value (in milliseconds)
		timeoutMs, err := strconv.Atoi(timeoutParam)
		if err != nil || timeoutMs < 0 {
			http.Error(w, "Invalid timeout value, must be a non-negative integer", http.StatusBadRequest)
			return
		}

		// If timeout is 0, it means no timeout
		if timeoutMs == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Create a context with the specified timeout
		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(timeoutMs)*time.Millisecond)
		defer cancel()

		// Create a new request with the timeout context
		r = r.WithContext(ctx)

		// Use a channel to signal when the response is complete
		doneChan := make(chan bool)
		go func() {
			next.ServeHTTP(w, r)
			doneChan <- true
		}()

		// Wait for either the request to complete or the timeout to expire
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				http.Error(w, "Request timed out", http.StatusRequestTimeout)
			}
		case <-doneChan:
			// Request completed normally
		}
	})
}
