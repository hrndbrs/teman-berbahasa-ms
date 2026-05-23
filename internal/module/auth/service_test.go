package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/module/auth"
	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockRepo struct {
	getUserByEmailFn              func(ctx context.Context, email string) (dbq.GetUserByEmailRow, error)
	getUserByIDFn                 func(ctx context.Context, id uuid.UUID) (dbq.GetUserByIDRow, error)
	incrementFailedAttemptsFn     func(ctx context.Context, id uuid.UUID) error
	resetFailedAttemptsFn         func(ctx context.Context, id uuid.UUID) error
	updateUserPasswordFn          func(ctx context.Context, id uuid.UUID, hash string) error
	insertRefreshTokenFn          func(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error
	getRefreshTokenByRawFn        func(ctx context.Context, raw string) (dbq.GetRefreshTokenByHashRow, error)
	deleteRefreshTokenFn          func(ctx context.Context, id uuid.UUID) error
	deleteRefreshTokensByUserIDFn func(ctx context.Context, userID uuid.UUID) error
	insertPasswordResetTokenFn    func(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error
	getPasswordResetTokenByRawFn  func(ctx context.Context, raw string) (dbq.GetPasswordResetTokenByHashRow, error)
	deletePasswordResetTokenFn    func(ctx context.Context, id uuid.UUID) error
	deletePasswordResetByUserIDFn func(ctx context.Context, userID uuid.UUID) error
}

func (m *mockRepo) GetUserByEmail(ctx context.Context, email string) (dbq.GetUserByEmailRow, error) {
	return m.getUserByEmailFn(ctx, email)
}
func (m *mockRepo) GetUserByID(ctx context.Context, id uuid.UUID) (dbq.GetUserByIDRow, error) {
	if m.getUserByIDFn != nil {
		return m.getUserByIDFn(ctx, id)
	}
	return dbq.GetUserByIDRow{}, nil
}
func (m *mockRepo) IncrementFailedAttempts(ctx context.Context, id uuid.UUID) error {
	if m.incrementFailedAttemptsFn != nil {
		return m.incrementFailedAttemptsFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) ResetFailedAttempts(ctx context.Context, id uuid.UUID) error {
	if m.resetFailedAttemptsFn != nil {
		return m.resetFailedAttemptsFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) UpdateUserPassword(ctx context.Context, id uuid.UUID, hash string) error {
	if m.updateUserPasswordFn != nil {
		return m.updateUserPasswordFn(ctx, id, hash)
	}
	return nil
}
func (m *mockRepo) InsertRefreshToken(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error {
	if m.insertRefreshTokenFn != nil {
		return m.insertRefreshTokenFn(ctx, userID, raw, exp)
	}
	return nil
}
func (m *mockRepo) GetRefreshTokenByRaw(ctx context.Context, raw string) (dbq.GetRefreshTokenByHashRow, error) {
	return m.getRefreshTokenByRawFn(ctx, raw)
}
func (m *mockRepo) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	if m.deleteRefreshTokenFn != nil {
		return m.deleteRefreshTokenFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deleteRefreshTokensByUserIDFn != nil {
		return m.deleteRefreshTokensByUserIDFn(ctx, userID)
	}
	return nil
}
func (m *mockRepo) InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, raw string, exp time.Time) error {
	if m.insertPasswordResetTokenFn != nil {
		return m.insertPasswordResetTokenFn(ctx, userID, raw, exp)
	}
	return nil
}
func (m *mockRepo) GetPasswordResetTokenByRaw(ctx context.Context, raw string) (dbq.GetPasswordResetTokenByHashRow, error) {
	return m.getPasswordResetTokenByRawFn(ctx, raw)
}
func (m *mockRepo) DeletePasswordResetToken(ctx context.Context, id uuid.UUID) error {
	if m.deletePasswordResetTokenFn != nil {
		return m.deletePasswordResetTokenFn(ctx, id)
	}
	return nil
}
func (m *mockRepo) DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	if m.deletePasswordResetByUserIDFn != nil {
		return m.deletePasswordResetByUserIDFn(ctx, userID)
	}
	return nil
}

type mockEmailSender struct {
	sendPasswordResetFn func(ctx context.Context, toEmail, rawToken string) error
}

func (m *mockEmailSender) SendPasswordReset(ctx context.Context, toEmail, rawToken string) error {
	if m.sendPasswordResetFn != nil {
		return m.sendPasswordResetFn(ctx, toEmail, rawToken)
	}
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

type noopTxBeginner struct{}

func (n *noopTxBeginner) Begin(ctx context.Context) (pgx.Tx, error) {
	return nil, errors.New("unexpected tx begin")
}

func newTestTokenManager(t *testing.T) *token.Manager {
	t.Helper()
	m, err := token.NewManager("../../../private.pem", "../../../public.pem")
	if err != nil {
		t.Skip("RSA key files not found — run `make keys` first")
	}
	return m
}

func mustHashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	require.NoError(t, err)
	return string(h)
}

func validUserRow(t *testing.T) dbq.GetUserByEmailRow {
	t.Helper()
	id, _ := uuid.NewV7()
	return dbq.GetUserByEmailRow{
		ID:             id,
		FirstName:      "Ana",
		LastName:       "Budi",
		Email:          "ana@school.com",
		PasswordHash:   mustHashPassword(t, "password123"),
		Role:           "admin",
		Status:         "active",
		FailedAttempts: 0,
	}
}

func futureTimestamp(d time.Duration) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(d), Valid: true}
}

func pastTimestamp(d time.Duration) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: time.Now().Add(-d), Valid: true}
}

// suppress unused import
var _ = errors.New

// ── Login ─────────────────────────────────────────────────────────────────────

func TestLogin_UserNotFound(t *testing.T) {
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return dbq.GetUserByEmailRow{}, pgx.ErrNoRows
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	_, err := svc.Login(context.Background(), "nobody@school.com", "pass")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogin_InactiveUser(t *testing.T) {
	user := validUserRow(t)
	user.Status = "inactive"
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return user, nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	_, err := svc.Login(context.Background(), user.Email, "password123")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
}

func TestLogin_AccountLocked(t *testing.T) {
	user := validUserRow(t)
	user.FailedAttempts = 10
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return user, nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	_, err := svc.Login(context.Background(), user.Email, "password123")
	assert.ErrorIs(t, err, auth.ErrAccountLocked)
}

func TestLogin_WrongPassword_IncrementsAttempts(t *testing.T) {
	user := validUserRow(t)
	incremented := false
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return user, nil
		},
		incrementFailedAttemptsFn: func(_ context.Context, _ uuid.UUID) error {
			incremented = true
			return nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	_, err := svc.Login(context.Background(), user.Email, "wrongpassword")
	assert.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.True(t, incremented)
}

func TestLogin_Success(t *testing.T) {
	user := validUserRow(t)
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return user, nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	resp, err := svc.Login(context.Background(), user.Email, "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Equal(t, 900, resp.ExpiresIn)
	assert.Equal(t, user.ID.String(), resp.User.ID)
	assert.Equal(t, "admin", resp.User.Role)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

// Refresh tests now use raw SQL via pool — unit tests require pgxmock or integration.
// Skip these at the unit level.

func TestRefresh_TokenNotFound(t *testing.T) {
	t.Skip("requires pgxmock — covered by integration tests")
}

func TestRefresh_ExpiredToken(t *testing.T) {
	t.Skip("requires pgxmock — covered by integration tests")
}

func TestRefresh_Success(t *testing.T) {
	t.Skip("requires pgxmock — covered by integration tests")
}

// ── Logout ────────────────────────────────────────────────────────────────────

func TestLogout_TokenNotFound_ReturnsNil(t *testing.T) {
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getRefreshTokenByRawFn: func(_ context.Context, _ string) (dbq.GetRefreshTokenByHashRow, error) {
			return dbq.GetRefreshTokenByHashRow{}, pgx.ErrNoRows
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	err := svc.Logout(context.Background(), "nonexistent")
	assert.NoError(t, err)
}

func TestLogout_DeletesAllUserSessions(t *testing.T) {
	tokenID, _ := uuid.NewV7()
	userID, _ := uuid.NewV7()
	var deletedForUserID uuid.UUID

	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getRefreshTokenByRawFn: func(_ context.Context, _ string) (dbq.GetRefreshTokenByHashRow, error) {
			return dbq.GetRefreshTokenByHashRow{
				ID:        tokenID,
				UserID:    userID,
				ExpiresAt: futureTimestamp(7 * 24 * time.Hour),
			}, nil
		},
		deleteRefreshTokensByUserIDFn: func(_ context.Context, id uuid.UUID) error {
			deletedForUserID = id
			return nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	err := svc.Logout(context.Background(), "valid-token")
	require.NoError(t, err)
	assert.Equal(t, userID, deletedForUserID)
}

// ── ForgotPassword ────────────────────────────────────────────────────────────

func TestForgotPassword_UserNotFound_ReturnsNil(t *testing.T) {
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return dbq.GetUserByEmailRow{}, pgx.ErrNoRows
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	err := svc.ForgotPassword(context.Background(), "nobody@school.com")
	assert.NoError(t, err)
}

func TestForgotPassword_SendsEmail(t *testing.T) {
	user := validUserRow(t)
	emailSent := false

	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getUserByEmailFn: func(_ context.Context, _ string) (dbq.GetUserByEmailRow, error) {
			return user, nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{
		sendPasswordResetFn: func(_ context.Context, _ string, _ string) error {
			emailSent = true
			return nil
		},
	})

	err := svc.ForgotPassword(context.Background(), user.Email)
	require.NoError(t, err)
	assert.True(t, emailSent)
}

// ── ResetPassword ─────────────────────────────────────────────────────────────

func TestResetPassword_TokenNotFound(t *testing.T) {
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getPasswordResetTokenByRawFn: func(_ context.Context, _ string) (dbq.GetPasswordResetTokenByHashRow, error) {
			return dbq.GetPasswordResetTokenByHashRow{}, pgx.ErrNoRows
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	err := svc.ResetPassword(context.Background(), "bad-token", "newpassword123")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestResetPassword_ExpiredToken(t *testing.T) {
	resetID, _ := uuid.NewV7()
	userID, _ := uuid.NewV7()
	svc := auth.NewService(newTestTokenManager(t), &mockRepo{
		getPasswordResetTokenByRawFn: func(_ context.Context, _ string) (dbq.GetPasswordResetTokenByHashRow, error) {
			return dbq.GetPasswordResetTokenByHashRow{
				ID:        resetID,
				UserID:    userID,
				ExpiresAt: pastTimestamp(time.Hour),
			}, nil
		},
	}, &noopTxBeginner{}, &mockEmailSender{})

	err := svc.ResetPassword(context.Background(), "expired-token", "newpassword123")
	assert.ErrorIs(t, err, auth.ErrInvalidToken)
}

func TestResetPassword_Success(t *testing.T) {
	t.Skip("uses raw SQL via pool — requires pgxmock or integration DB")
}
