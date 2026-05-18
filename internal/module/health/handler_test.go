package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	interndb "github.com/hrndbrs/teman-berbahasa-ms/internal/db"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/health"
)

func TestHealth_OK(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := interndb.Connect(context.Background(), dsn)
	require.NoError(t, err)
	defer pool.Close()

	r := chi.NewRouter()
	health.NewHandler(pool).Register(r)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

	var body map[string]string
	require.NoError(t, json.NewDecoder(rr.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "ok", body["db"])
}
