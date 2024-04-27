package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.StatusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.StatusCode == 0 {
		lrw.StatusCode = http.StatusOK
	}
	return lrw.ResponseWriter.Write(b)
}

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
