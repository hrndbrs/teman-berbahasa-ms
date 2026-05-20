package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	dbq "github.com/hrndbrs/teman-berbahasa-ms/internal/db/query"
)

type pgAuthRepository struct {
	q *dbq.Queries
}

func NewRepository(pool *pgxpool.Pool) *pgAuthRepository {
	return &pgAuthRepository{q: dbq.New(pool)}
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func toPgTime(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func (r *pgAuthRepository) GetUserByEmail(ctx context.Context, email string) (dbq.GetUserByEmailRow, error) {
	return r.q.GetUserByEmail(ctx, email)
}

func (r *pgAuthRepository) GetUserByID(ctx context.Context, id uuid.UUID) (dbq.GetUserByIDRow, error) {
	return r.q.GetUserByID(ctx, id)
}

func (r *pgAuthRepository) IncrementFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	return r.q.IncrementFailedAttempts(ctx, userID)
}

func (r *pgAuthRepository) ResetFailedAttempts(ctx context.Context, userID uuid.UUID) error {
	return r.q.ResetFailedAttempts(ctx, userID)
}

func (r *pgAuthRepository) UpdateUserPassword(ctx context.Context, userID uuid.UUID, passwordHash string) error {
	return r.q.UpdateUserPassword(ctx, dbq.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: passwordHash,
	})
}

func (r *pgAuthRepository) InsertRefreshToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return r.q.InsertRefreshToken(ctx, dbq.InsertRefreshTokenParams{
		ID:        id,
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: toPgTime(expiresAt),
	})
}

func (r *pgAuthRepository) GetRefreshTokenByRaw(ctx context.Context, rawToken string) (dbq.GetRefreshTokenByHashRow, error) {
	return r.q.GetRefreshTokenByHash(ctx, hashToken(rawToken))
}

func (r *pgAuthRepository) DeleteRefreshToken(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteRefreshToken(ctx, id)
}

func (r *pgAuthRepository) DeleteRefreshTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeleteRefreshTokensByUserID(ctx, userID)
}

func (r *pgAuthRepository) InsertPasswordResetToken(ctx context.Context, userID uuid.UUID, rawToken string, expiresAt time.Time) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return r.q.InsertPasswordResetToken(ctx, dbq.InsertPasswordResetTokenParams{
		ID:        id,
		UserID:    userID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: toPgTime(expiresAt),
	})
}

func (r *pgAuthRepository) GetPasswordResetTokenByRaw(ctx context.Context, rawToken string) (dbq.GetPasswordResetTokenByHashRow, error) {
	return r.q.GetPasswordResetTokenByHash(ctx, hashToken(rawToken))
}

func (r *pgAuthRepository) DeletePasswordResetToken(ctx context.Context, id uuid.UUID) error {
	return r.q.DeletePasswordResetToken(ctx, id)
}

func (r *pgAuthRepository) DeletePasswordResetTokensByUserID(ctx context.Context, userID uuid.UUID) error {
	return r.q.DeletePasswordResetTokensByUserID(ctx, userID)
}
