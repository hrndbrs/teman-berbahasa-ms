package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/config"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost/db")
	t.Setenv("JWT_PRIVATE_KEY_PATH", "/keys/private.pem")
	t.Setenv("JWT_PUBLIC_KEY_PATH", "/keys/public.pem")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
}

func TestLoad_AllPresent(t *testing.T) {
	setValidEnv(t)

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "postgres://user:pass@localhost/db", cfg.DatabaseURL)
	assert.Equal(t, "/keys/private.pem", cfg.JWTPrivateKeyPath)
	assert.Equal(t, []string{"http://localhost:3000"}, cfg.CORSAllowedOrigins)
	assert.Equal(t, "8080", cfg.Port) // default
}

func TestLoad_CustomPort(t *testing.T) {
	setValidEnv(t)
	t.Setenv("PORT", "9090")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "9090", cfg.Port)
}

func TestLoad_MultipleOrigins(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,https://app.example.com")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, []string{"http://localhost:3000", "https://app.example.com"}, cfg.CORSAllowedOrigins)
}

func TestLoad_MissingDatabaseURL(t *testing.T) {
	setValidEnv(t)
	t.Setenv("DATABASE_URL", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "DATABASE_URL")
}

func TestLoad_MissingJWTPrivateKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_PRIVATE_KEY_PATH", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PRIVATE_KEY_PATH")
}

func TestLoad_MissingJWTPublicKey(t *testing.T) {
	setValidEnv(t)
	t.Setenv("JWT_PUBLIC_KEY_PATH", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_PUBLIC_KEY_PATH")
}

func TestLoad_MissingCORSOrigins(t *testing.T) {
	setValidEnv(t)
	t.Setenv("CORS_ALLOWED_ORIGINS", "")

	_, err := config.Load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "CORS_ALLOWED_ORIGINS")
}
