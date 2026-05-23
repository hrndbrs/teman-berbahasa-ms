package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgUserRepository struct {
	q    *dbq.Queries
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *pgUserRepository {
	return &pgUserRepository{q: dbq.New(pool), pool: pool}
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func (r *pgUserRepository) GetUserByIDFull(ctx context.Context, id uuid.UUID) (User, error) {
	row, err := r.q.GetUserByIDFull(ctx, id)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Role:      row.Role,
		Phone:     row.Phone,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func (r *pgUserRepository) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		return User{}, err
	}
	return User{
		ID:     row.ID,
		Email:  row.Email,
		Role:   row.Role,
		Status: row.Status,
	}, nil
}

func (r *pgUserRepository) ListUsers(ctx context.Context, params ListParams) ([]User, int64, error) {
	rows, err := r.q.ListUsers(ctx, dbq.ListUsersParams{
		Role:       params.Role,
		Status:     params.Status,
		PageSize:   int32(params.PerPage),
		PageOffset: int32((params.Page - 1) * params.PerPage),
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountUsers(ctx, dbq.CountUsersParams{
		Role:   params.Role,
		Status: params.Status,
	})
	if err != nil {
		return nil, 0, err
	}
	users := make([]User, len(rows))
	for i, row := range rows {
		users[i] = User{
			ID:        row.ID,
			FirstName: row.FirstName,
			LastName:  row.LastName,
			Email:     row.Email,
			Role:      row.Role,
			Phone:     row.Phone,
			Status:    row.Status,
			CreatedAt: row.CreatedAt.Time,
			UpdatedAt: row.UpdatedAt.Time,
		}
	}
	return users, total, nil
}

func (r *pgUserRepository) CreateUser(ctx context.Context, id uuid.UUID, req CreateUserRequest, passwordHash string) (User, error) {
	row, err := r.q.CreateUser(ctx, dbq.CreateUserParams{
		ID:           id,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Email:        req.Email,
		PasswordHash: passwordHash,
		Role:         req.Role,
		Phone:        req.Phone,
	})
	if err != nil {
		return User{}, err
	}
	return User{
		ID:        row.ID,
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Role:      row.Role,
		Phone:     row.Phone,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}, nil
}

func scanUserRow(row pgx.Row) (User, error) {
	var u User
	var createdAt, updatedAt pgtype.Timestamptz
	err := row.Scan(
		&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Role,
		&u.Phone, &u.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return User{}, err
	}
	u.CreatedAt = createdAt.Time
	u.UpdatedAt = updatedAt.Time
	return u, nil
}

func (r *pgUserRepository) UpdateUser(ctx context.Context, id uuid.UUID, req FullUserUpdate) (User, error) {
	var phone pgtype.Text
	if req.Phone != nil {
		phone = pgtype.Text{String: *req.Phone, Valid: true}
	}
	row := r.pool.QueryRow(ctx,
		`UPDATE users SET
			first_name = $1,
			last_name  = $2,
			email      = $3,
			role       = $4,
			phone      = $5,
			status     = $6,
			updated_at = NOW()
		WHERE id = $7
		RETURNING id, first_name, last_name, email, role, phone, status, created_at, updated_at`,
		req.FirstName, req.LastName, req.Email, req.Role, phone, req.Status, id,
	)
	return scanUserRow(row)
}

func (r *pgUserRepository) InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return r.q.InsertPasswordResetToken(ctx, dbq.InsertPasswordResetTokenParams{
		ID:        id,
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
}

func (r *pgUserRepository) DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeletePasswordResetTokensByUserID(ctx, userID)
}

func (r *pgUserRepository) DeleteUserSessions(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteUserSessions(ctx, userID)
}
