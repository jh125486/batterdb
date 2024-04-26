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
		lrw := &loggingResponseWriter{ResponseWriter: w}
		h.ServeHTTP(lrw, r)
		slog.Info(fmt.Sprintf("%s %v %d", r.Method, r.URL.Path, lrw.StatusCode))
	})
}
