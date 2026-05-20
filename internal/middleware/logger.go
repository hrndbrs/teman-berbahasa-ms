package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		rec := &logRecord{}
		r = r.WithContext(context.WithValue(r.Context(), contextKeyLog{}, rec))

		next.ServeHTTP(ww, r)

		level := slog.LevelInfo
		if r.URL.Path == "/health" {
			level = slog.LevelDebug
		}

		args := append([]any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.status,
			"duration_ms", time.Since(start).Milliseconds(),
		}, rec.snapshot()...)

		slog.Log(r.Context(), level, "request", args...)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}
