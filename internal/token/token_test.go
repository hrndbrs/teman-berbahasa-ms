package token_test

import (
	"os"
	"testing"
	"time"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestManager(t *testing.T) *token.Manager {
	t.Helper()
	if _, err := os.Stat("../../private.pem"); err != nil {
		t.Skip("RSA key files not found — run `make keys` first")
	}
	m, err := token.NewManager("../../private.pem", "../../public.pem")
	require.NoError(t, err)
	return m
}

func TestSign_Verify_RoundTrip(t *testing.T) {
	m := newTestManager(t)

	tok, err := m.Sign("user-123", "admin", 15*time.Minute)
	require.NoError(t, err)
	assert.NotEmpty(t, tok)

	claims, err := m.Verify(tok)
	require.NoError(t, err)
	assert.Equal(t, "user-123", claims.UserID)
	assert.Equal(t, "admin", claims.Role)
}

func TestVerify_ExpiredToken(t *testing.T) {
	m := newTestManager(t)

	tok, err := m.Sign("user-123", "admin", -time.Second)
	require.NoError(t, err)

	_, err = m.Verify(tok)
	assert.Error(t, err)
}

func TestVerify_Tampered(t *testing.T) {
	m := newTestManager(t)

	tok, err := m.Sign("user-123", "admin", 15*time.Minute)
	require.NoError(t, err)

	_, err = m.Verify(tok + "x")
	assert.Error(t, err)
}
