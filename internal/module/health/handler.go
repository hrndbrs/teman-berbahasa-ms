package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Handler struct {
	pool *pgxpool.Pool
}

func NewHandler(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool}
}

func (h *Handler) Register(r chi.Router) {
	r.Get("/health", h.health)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	dbStatus := "ok"
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "error"
	}

	httpStatus := http.StatusOK
	apiStatus := "ok"
	if dbStatus != "ok" {
		httpStatus = http.StatusServiceUnavailable
		apiStatus = "error"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status": apiStatus,
		"db":     dbStatus,
	})
}
