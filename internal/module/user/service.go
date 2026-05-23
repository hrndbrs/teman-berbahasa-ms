package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/hrndbrs/teman-berbahasa-ms/internal/patch"
	"github.com/hrndbrs/teman-berbahasa-ms/internal/pagination"
)

var (
	ErrEmailConflict = errors.New("email already in use")
	ErrNotFound      = errors.New("user not found")
	ErrForbidden     = errors.New("forbidden")
)

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type User struct {
	ID        uuid.UUID
	FirstName string
	LastName  string
	Email     string
	Role      string
	Phone     *string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ListParams struct {
	Page    int
	PerPage int
	Role    *string
	Status  *string
}

type CreateUserRequest struct {
	FirstName string
	LastName  string
	Email     string
	Role      string
	Phone     *string
}

type UpdateUserRequest struct {
	FirstName *string
	LastName  *string
	Email     *string
	Role      *string
	Phone     *patch.Patchable[string]
	Status    *string
}

type FullUserUpdate struct {
	FirstName string
	LastName  string
	Email     string
	Role      string
	Phone     *string
	Status    string
}

type ListResponse struct {
	Data       []User
	Pagination PaginationMeta
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type UserRepository interface {
	GetUserByIDFull(ctx context.Context, id uuid.UUID) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, error)
	ListUsers(ctx context.Context, params ListParams) ([]User, int64, error)
	CreateUser(ctx context.Context, id uuid.UUID, req CreateUserRequest, passwordHash string) (User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req FullUserUpdate) (User, error)
	InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error
	DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error
	DeleteUserSessions(ctx context.Context, userID uuid.UUID) error
}

type EmailSender interface {
	SendInvite(ctx context.Context, toEmail, firstName, rawToken string) error
}

type UserService struct {
	repo    UserRepository
	emailer EmailSender
}

func NewService(repo UserRepository, emailer EmailSender) *UserService {
	return &UserService{repo: repo, emailer: emailer}
}

func (s *UserService) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	_, err := s.repo.GetUserByEmail(ctx, req.Email)
	if err == nil {
		return nil, ErrEmailConflict
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	rawPass := make([]byte, 32)
	if _, err := rand.Read(rawPass); err != nil {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword(rawPass, 12)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Email = strings.TrimSpace(req.Email)

	u, err := s.repo.CreateUser(ctx, id, req, string(hash))
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailConflict
		}
		return nil, err
	}

	rawToken, err := generateRawToken()
	if err != nil {
		return nil, err
	}
	_ = s.repo.DeletePasswordResetTokensByUserID(ctx, id)
	if err := s.repo.InsertPasswordResetToken(ctx, id, rawToken, time.Now().Add(time.Hour)); err != nil {
		return nil, err
	}
	if err := s.emailer.SendInvite(ctx, u.Email, u.FirstName, rawToken); err != nil {
		slog.ErrorContext(ctx, "failed to send invite email", "error", err, "user_id", id)
	}

	return &u, nil
}

func (s *UserService) GetUser(ctx context.Context, callerID, callerRole, targetID string) (*User, error) {
	if callerID == "" {
		return nil, ErrForbidden
	}
	if callerRole != "admin" && callerID != targetID {
		return nil, ErrForbidden
	}
	id, err := uuid.Parse(targetID)
	if err != nil {
		return nil, ErrNotFound
	}
	u, err := s.repo.GetUserByIDFull(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserService) ListUsers(ctx context.Context, params ListParams) (*ListResponse, error) {
	users, total, err := s.repo.ListUsers(ctx, params)
	if err != nil {
		return nil, err
	}
	return &ListResponse{
		Data: users,
		Pagination: PaginationMeta{
			Page:       params.Page,
			PerPage:    params.PerPage,
			Total:      int(total),
			TotalPages: pagination.TotalPages(total, params.PerPage),
		},
	}, nil
}

func MergeUserUpdate(existing User, req UpdateUserRequest) FullUserUpdate {
	u := FullUserUpdate{
		FirstName: existing.FirstName,
		LastName:  existing.LastName,
		Email:     existing.Email,
		Role:      existing.Role,
		Phone:     existing.Phone,
		Status:    existing.Status,
	}
	if req.FirstName != nil {
		u.FirstName = *req.FirstName
	}
	if req.LastName != nil {
		u.LastName = *req.LastName
	}
	if req.Email != nil {
		u.Email = *req.Email
	}
	if req.Role != nil {
		u.Role = *req.Role
	}
	if req.Status != nil {
		u.Status = *req.Status
	}
	u.Phone = req.Phone.ValueOr(u.Phone)
	return u
}

func (s *UserService) UpdateUser(ctx context.Context, targetID string, req UpdateUserRequest) (*User, error) {
	id, err := uuid.Parse(targetID)
	if err != nil {
		return nil, ErrNotFound
	}
	existing, err := s.repo.GetUserByIDFull(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	merged := MergeUserUpdate(existing, req)
	if req.Email != nil && *req.Email != existing.Email {
		_, err := s.repo.GetUserByEmail(ctx, *req.Email)
		if err == nil {
			return nil, ErrEmailConflict
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	u, err := s.repo.UpdateUser(ctx, id, merged)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailConflict
		}
		return nil, err
	}
	roleChanged := req.Role != nil && *req.Role != existing.Role
	statusDeactivated := req.Status != nil && *req.Status == "inactive"
	if roleChanged || statusDeactivated {
		if err := s.repo.DeleteUserSessions(ctx, id); err != nil {
			return nil, fmt.Errorf("revoking sessions: %w", err)
		}
	}
	return &u, nil
}

func generateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
