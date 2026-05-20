package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/middleware"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
)

// --- Recovery ---

func TestRecovery_NoPanic(t *testing.T) {
	handler := middleware.Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRecovery_Panic(t *testing.T) {
	handler := middleware.Recovery(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something broke")
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	require.NotPanics(t, func() { handler.ServeHTTP(rr, req) })

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "INTERNAL_ERROR")
}

// --- CORS ---

func TestCORS_AllowedOrigin(t *testing.T) {
	handler := middleware.CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, "http://localhost:3000", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	handler := middleware.CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORS_Preflight(t *testing.T) {
	called := false
	handler := middleware.CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.False(t, called, "next handler must not be called for preflight")
}

func TestCORS_NoOriginHeader(t *testing.T) {
	handler := middleware.CORS([]string{"http://localhost:3000"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil) // no Origin header
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, rr.Code)
}

// --- Auth ---

func newTestTokenManager(t *testing.T) *token.Manager {
	t.Helper()
	m, err := token.NewManager("../../private.pem", "../../public.pem")
	if err != nil {
		t.Skip("RSA key files not found — run `make keys` first")
	}
	return m
}

func TestAuth_MissingHeader(t *testing.T) {
	tm := newTestTokenManager(t)
	h := middleware.Auth(tm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_InvalidToken(t *testing.T) {
	tm := newTestTokenManager(t)
	h := middleware.Auth(tm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func TestAuth_ValidToken_SetsContext(t *testing.T) {
	tm := newTestTokenManager(t)
	var gotUserID, gotRole string

	h := middleware.Auth(tm)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = middleware.UserIDFromCtx(r.Context())
		gotRole = middleware.UserRoleFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	tok, err := tm.Sign("user-abc", "teacher", 15*time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "user-abc", gotUserID)
	assert.Equal(t, "teacher", gotRole)
}

func TestRequireRole_AllowedRole(t *testing.T) {
	tm := newTestTokenManager(t)
	h := middleware.Auth(tm)(middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	tok, err := tm.Sign("user-abc", "admin", 15*time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRequireRole_ForbiddenRole(t *testing.T) {
	tm := newTestTokenManager(t)
	h := middleware.Auth(tm)(middleware.RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	tok, err := tm.Sign("user-abc", "teacher", 15*time.Minute)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
