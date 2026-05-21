package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/token"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrAccountLocked      = errors.New("account locked")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
	resetTokenTTL   = time.Hour
)

type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (dbq.GetUserByEmailRow, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (dbq.GetUserByIDRow, error)
	IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error
	ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error
	UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error
	InsertRefreshToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error
	GetRefreshTokenByRaw(ctx context.Context, rawToken string) (dbq.GetRefreshTokenByHashRow, error)
	DeleteRefreshToken(ctx context.Context, id uuid.UUID) error
	DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error
	InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error
	GetPasswordResetTokenByRaw(ctx context.Context, rawToken string) (dbq.GetPasswordResetTokenByHashRow, error)
	DeletePasswordResetToken(ctx context.Context, id uuid.UUID) error
	DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error
}

type EmailSender interface {
	SendPasswordReset(ctx context.Context, toEmail, rawToken string) error
}

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int      `json:"expires_in"`
	User         UserInfo `json:"user"`
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

type UserInfo struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

type AuthService struct {
	tm      *token.Manager
	repo    AuthRepository
	emailer EmailSender
}

func NewService(tm *token.Manager, repo AuthRepository, emailer EmailSender) *AuthService {
	return &AuthService{tm: tm, repo: repo, emailer: emailer}
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if user.Status == "inactive" {
		return nil, ErrInvalidCredentials
	}
	if user.FailedAttempts >= 10 {
		slog.WarnContext(ctx, "login rejected: account locked", "user_id", user.ID)
		return nil, ErrAccountLocked
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		_ = s.repo.IncrementFailedAttempts(ctx, user.ID)
		slog.WarnContext(ctx, "login failed: invalid credentials", "user_id", user.ID, "attempts", user.FailedAttempts+1)
		return nil, ErrInvalidCredentials
	}
	_ = s.repo.ResetFailedAttempts(ctx, user.ID)

	accessToken, err := s.tm.Sign(user.ID.String(), user.Role, accessTokenTTL)
	if err != nil {
		return nil, err
	}
	rawRefresh, err := generateRawToken()
	if err != nil {
		return nil, err
	}
	if err := s.repo.InsertRefreshToken(ctx, user.ID, rawRefresh, time.Now().Add(refreshTokenTTL)); err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "login", "user_id", user.ID, "role", user.Role)
	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
		User: UserInfo{
			ID:        user.ID.String(),
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
		},
	}, nil
}

func (s *AuthService) Me(ctx context.Context, userID string) (*UserInfo, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil || user.Status != "active" {
		return nil, ErrInvalidToken
	}
	return &UserInfo{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*TokenPair, error) {
	row, err := s.repo.GetRefreshTokenByRaw(ctx, rawToken)
	if err != nil {
		return nil, ErrInvalidToken
	}
	if time.Now().After(row.ExpiresAt.Time) {
		return nil, ErrInvalidToken
	}

	user, err := s.repo.GetUserByID(ctx, row.UserID)
	if err != nil || user.Status != "active" {
		return nil, ErrInvalidToken
	}

	if err := s.repo.DeleteRefreshToken(ctx, row.ID); err != nil {
		return nil, err
	}

	accessToken, err := s.tm.Sign(row.UserID.String(), user.Role, accessTokenTTL)
	if err != nil {
		return nil, err
	}
	newRaw, err := generateRawToken()
	if err != nil {
		return nil, err
	}
	if err := s.repo.InsertRefreshToken(ctx, row.UserID, newRaw, time.Now().Add(refreshTokenTTL)); err != nil {
		return nil, err
	}

	slog.DebugContext(ctx, "token refreshed", "user_id", row.UserID)
	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRaw,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *AuthService) GetMe(ctx context.Context, userID string) (*UserInfo, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil || user.Status != "active" {
		return nil, ErrInvalidToken
	}
	return &UserInfo{
		ID:        user.ID.String(),
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Role:      user.Role,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	row, err := s.repo.GetRefreshTokenByRaw(ctx, rawToken)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := s.repo.DeleteRefreshTokensByUserID(ctx, row.UserID); err != nil {
		return err
	}
	slog.DebugContext(ctx, "logout", "user_id", row.UserID)
	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	user, err := s.repo.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}

	rawToken, err := generateRawToken()
	if err != nil {
		return err
	}
	_ = s.repo.DeletePasswordResetTokensByUserID(ctx, user.ID)
	if err := s.repo.InsertPasswordResetToken(ctx, user.ID, rawToken, time.Now().Add(resetTokenTTL)); err != nil {
		return err
	}

	slog.InfoContext(ctx, "password reset requested", "user_id", user.ID)
	if err := s.emailer.SendPasswordReset(ctx, user.Email, rawToken); err != nil {
		slog.ErrorContext(ctx, "failed to send password reset email", "error", err, "user_id", user.ID)
	}
	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, rawToken, newPassword string) error {
	row, err := s.repo.GetPasswordResetTokenByRaw(ctx, rawToken)
	if err != nil {
		return ErrInvalidToken
	}
	if time.Now().After(row.ExpiresAt.Time) {
		return ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserPassword(ctx, row.UserID, string(hash)); err != nil {
		return err
	}
	if err := s.repo.DeletePasswordResetToken(ctx, row.ID); err != nil {
		slog.ErrorContext(ctx, "failed to delete reset token after password update", "error", err)
	}
	_ = s.repo.DeleteRefreshTokensByUserID(ctx, row.UserID)
	slog.InfoContext(ctx, "password reset completed", "user_id", row.UserID)
	return nil
}

func generateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
