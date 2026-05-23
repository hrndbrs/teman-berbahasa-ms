package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/getsentry/sentry-go"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.ErrorContext(r.Context(), "panic recovered",
					"panic", rec,
					"stack", string(debug.Stack()),
				)
				sentry.CurrentHub().RecoverWithContext(r.Context(), rec)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
