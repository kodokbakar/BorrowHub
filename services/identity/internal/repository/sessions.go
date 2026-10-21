package repository

import (
	"context"
	"errors"
	"time"

	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, s *domain.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		s.ID,
		s.UserID,
		s.RefreshTokenHash,
		s.UserAgent,
		s.IPAddress,
		s.ExpiresAt,
		s.RevokedAt,
		s.CreatedAt,
		s.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *SessionRepository) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, revoked_at, created_at, updated_at
		FROM sessions
		WHERE refresh_token_hash = $1
	`
	var s domain.Session
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&s.ID,
		&s.UserID,
		&s.RefreshTokenHash,
		&s.UserAgent,
		&s.IPAddress,
		&s.ExpiresAt,
		&s.RevokedAt,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrSessionNotFound
		}
		return nil, mapPgError(err)
	}
	return &s, nil
}

func (r *SessionRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) (bool, error) {
	query := `
		UPDATE sessions
		SET revoked_at = $2, updated_at = $2
		WHERE id = $1 AND revoked_at IS NULL
	`
	tag, err := r.pool.Exec(ctx, query, id, revokedAt)
	if err != nil {
		return false, mapPgError(err)
	}
	return tag.RowsAffected() > 0, nil
}

func (r *SessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	query := `
		UPDATE sessions
		SET revoked_at = $2, updated_at = $2
		WHERE user_id = $1 AND revoked_at IS NULL
	`
	_, err := r.pool.Exec(ctx, query, userID, revokedAt)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}
