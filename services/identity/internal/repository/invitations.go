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

type InvitationRepository struct {
	pool *pgxpool.Pool
}

func NewInvitationRepository(pool *pgxpool.Pool) *InvitationRepository {
	return &InvitationRepository{pool: pool}
}

func (r *InvitationRepository) Create(ctx context.Context, inv *domain.Invitation) error {
	query := `
		INSERT INTO invitations (id, organization_id, email, role, token_hash, invited_by_user_id, status, expires_at, accepted_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		inv.ID,
		inv.OrganizationID,
		inv.Email,
		inv.Role,
		inv.TokenHash,
		inv.InvitedByUserID,
		inv.Status,
		inv.ExpiresAt,
		inv.AcceptedAt,
		inv.CreatedAt,
		inv.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *InvitationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error) {
	query := `
		SELECT id, organization_id, email, role, token_hash, invited_by_user_id, status, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE token_hash = $1
	`
	var inv domain.Invitation
	err := r.pool.QueryRow(ctx, query, tokenHash).Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.InvitedByUserID,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, mapPgError(err)
	}
	return &inv, nil
}

func (r *InvitationRepository) GetPendingByOrgAndEmail(
	ctx context.Context,
	orgID uuid.UUID,
	email string,
) (*domain.Invitation, error) {
	query := `
		SELECT id, organization_id, email, role, token_hash, invited_by_user_id, status, expires_at, accepted_at, created_at, updated_at
		FROM invitations
		WHERE organization_id = $1 AND email = $2 AND status = 'pending' AND expires_at > NOW()
		LIMIT 1
	`
	var inv domain.Invitation
	err := r.pool.QueryRow(ctx, query, orgID, email).Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&inv.Role,
		&inv.TokenHash,
		&inv.InvitedByUserID,
		&inv.Status,
		&inv.ExpiresAt,
		&inv.AcceptedAt,
		&inv.CreatedAt,
		&inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvitationNotFound
		}
		return nil, mapPgError(err)
	}
	return &inv, nil
}

func (r *InvitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	query := `
		UPDATE invitations
		SET status = $2, accepted_at = $3, updated_at = $4
		WHERE id = $1
	`
	tag, err := r.pool.Exec(ctx, query, inv.ID, inv.Status, inv.AcceptedAt, inv.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvitationNotFound
	}
	return nil
}

func (r *InvitationRepository) AcceptInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
	acceptedAt time.Time,
	m *domain.Membership,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	invQuery := `
		UPDATE invitations
		SET status = 'accepted', accepted_at = $2, updated_at = $2
		WHERE id = $1 AND status = 'pending'
	`
	tag, err := tx.Exec(ctx, invQuery, invitationID, acceptedAt)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrInvitationAlreadyUsed
	}

	memQuery := `
		INSERT INTO memberships (id, organization_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(ctx, memQuery, m.ID, m.OrganizationID, m.UserID, m.Role, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}

	return tx.Commit(ctx)
}
