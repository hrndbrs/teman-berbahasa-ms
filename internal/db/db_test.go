package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/db"
)

func TestConnect_Success(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	pool, err := db.Connect(context.Background(), dsn)
	require.NoError(t, err)
	defer pool.Close()

	assert.NotNil(t, pool)
	assert.NoError(t, pool.Ping(context.Background()))
}

func TestConnect_InvalidDSN(t *testing.T) {
	_, err := db.Connect(context.Background(), "postgres://invalid-host:5432/db?connect_timeout=1")
	require.Error(t, err)
}
