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

type MembershipRepository struct {
	pool *pgxpool.Pool
}

func NewMembershipRepository(pool *pgxpool.Pool) *MembershipRepository {
	return &MembershipRepository{pool: pool}
}

func (r *MembershipRepository) Create(ctx context.Context, m *domain.Membership) error {
	query := `
		INSERT INTO memberships (id, organization_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.pool.Exec(ctx, query, m.ID, m.OrganizationID, m.UserID, m.Role, m.CreatedAt, m.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *MembershipRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	query := `
		SELECT id, organization_id, user_id, role, created_at, updated_at
		FROM memberships
		WHERE id = $1
	`
	var m domain.Membership
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID,
		&m.OrganizationID,
		&m.UserID,
		&m.Role,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, mapPgError(err)
	}
	return &m, nil
}

func (r *MembershipRepository) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.Membership, error) {
	query := `
		SELECT id, organization_id, user_id, role, created_at, updated_at
		FROM memberships
		WHERE organization_id = $1 AND user_id = $2
	`
	var m domain.Membership
	err := r.pool.QueryRow(ctx, query, orgID, userID).Scan(
		&m.ID,
		&m.OrganizationID,
		&m.UserID,
		&m.Role,
		&m.CreatedAt,
		&m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrMembershipNotFound
		}
		return nil, mapPgError(err)
	}
	return &m, nil
}

func (r *MembershipRepository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]domain.MembershipDetail, error) {
	query := `
		SELECT m.id, m.organization_id, m.user_id, m.role, u.email, u.display_name, m.created_at, m.updated_at
		FROM memberships m
		INNER JOIN users u ON u.id = m.user_id
		WHERE m.organization_id = $1
		ORDER BY m.created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var result []domain.MembershipDetail
	for rows.Next() {
		var d domain.MembershipDetail
		err := rows.Scan(
			&d.ID,
			&d.OrganizationID,
			&d.UserID,
			&d.Role,
			&d.Email,
			&d.DisplayName,
			&d.CreatedAt,
			&d.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, d)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *MembershipRepository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM memberships
		WHERE organization_id = $1 AND role = 'owner'
	`
	var count int
	err := r.pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, mapPgError(err)
	}
	return count, nil
}

func (r *MembershipRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapPgError(err)
	}
	defer tx.Rollback(ctx)

	var orgID uuid.UUID
	var currentRole domain.Role
	err = tx.QueryRow(ctx, `SELECT organization_id, role FROM memberships WHERE id = $1 FOR UPDATE`, id).Scan(&orgID, &currentRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrMembershipNotFound
		}
		return mapPgError(err)
	}

	if currentRole == domain.RoleOwner && role != domain.RoleOwner {
		var dummy uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM organizations WHERE id = $1 FOR UPDATE`, orgID).Scan(&dummy); err != nil {
			return mapPgError(err)
		}

		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM memberships WHERE organization_id = $1 AND role = 'owner'`, orgID).Scan(&count); err != nil {
			return mapPgError(err)
		}
		if count <= 1 {
			return domain.ErrCannotDemoteSoleOwner
		}
	}

	query := `
		UPDATE memberships
		SET role = $2, updated_at = $3
		WHERE id = $1
	`
	tag, err := tx.Exec(ctx, query, id, role, time.Now())
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}
	return tx.Commit(ctx)
}

func (r *MembershipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return mapPgError(err)
	}
	defer tx.Rollback(ctx)

	var orgID uuid.UUID
	var currentRole domain.Role
	err = tx.QueryRow(ctx, `SELECT organization_id, role FROM memberships WHERE id = $1 FOR UPDATE`, id).Scan(&orgID, &currentRole)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrMembershipNotFound
		}
		return mapPgError(err)
	}

	if currentRole == domain.RoleOwner {
		var dummy uuid.UUID
		if err := tx.QueryRow(ctx, `SELECT id FROM organizations WHERE id = $1 FOR UPDATE`, orgID).Scan(&dummy); err != nil {
			return mapPgError(err)
		}

		var count int
		if err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM memberships WHERE organization_id = $1 AND role = 'owner'`, orgID).Scan(&count); err != nil {
			return mapPgError(err)
		}
		if count <= 1 {
			return domain.ErrCannotRemoveSoleOwner
		}
	}

	query := `
		DELETE FROM memberships
		WHERE id = $1
	`
	tag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}
	return tx.Commit(ctx)
}
