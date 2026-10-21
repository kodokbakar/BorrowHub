package repository

import (
	"context"
	"errors"

	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OrganizationRepository struct {
	pool *pgxpool.Pool
}

func NewOrganizationRepository(pool *pgxpool.Pool) *OrganizationRepository {
	return &OrganizationRepository{pool: pool}
}

func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	query := `
		INSERT INTO organizations (id, name, slug, timezone, visibility, max_loan_duration_days, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		org.ID,
		org.Name,
		org.Slug,
		org.Timezone,
		org.Visibility,
		org.MaxLoanDurationDays,
		org.Status,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	return nil
}

func (r *OrganizationRepository) CreateWithInitialOwner(
	ctx context.Context,
	org *domain.Organization,
	ownerMembership *domain.Membership,
) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	orgQuery := `
		INSERT INTO organizations (id, name, slug, timezone, visibility, max_loan_duration_days, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err = tx.Exec(
		ctx,
		orgQuery,
		org.ID,
		org.Name,
		org.Slug,
		org.Timezone,
		org.Visibility,
		org.MaxLoanDurationDays,
		org.Status,
		org.CreatedAt,
		org.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}

	memberQuery := `
		INSERT INTO memberships (id, organization_id, user_id, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = tx.Exec(
		ctx,
		memberQuery,
		ownerMembership.ID,
		ownerMembership.OrganizationID,
		ownerMembership.UserID,
		ownerMembership.Role,
		ownerMembership.CreatedAt,
		ownerMembership.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}

	return tx.Commit(ctx)
}

func (r *OrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	query := `
		SELECT id, name, slug, timezone, visibility, max_loan_duration_days, status, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`
	var org domain.Organization
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Timezone,
		&org.Visibility,
		&org.MaxLoanDurationDays,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrgNotFound
		}
		return nil, mapPgError(err)
	}
	return &org, nil
}

func (r *OrganizationRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	query := `
		SELECT id, name, slug, timezone, visibility, max_loan_duration_days, status, created_at, updated_at
		FROM organizations
		WHERE slug = $1
	`
	var org domain.Organization
	err := r.pool.QueryRow(ctx, query, slug).Scan(
		&org.ID,
		&org.Name,
		&org.Slug,
		&org.Timezone,
		&org.Visibility,
		&org.MaxLoanDurationDays,
		&org.Status,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrOrgNotFound
		}
		return nil, mapPgError(err)
	}
	return &org, nil
}

func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	query := `
		UPDATE organizations
		SET name = $2, timezone = $3, visibility = $4, max_loan_duration_days = $5, status = $6, updated_at = $7
		WHERE id = $1
	`
	tag, err := r.pool.Exec(
		ctx,
		query,
		org.ID,
		org.Name,
		org.Timezone,
		org.Visibility,
		org.MaxLoanDurationDays,
		org.Status,
		org.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOrgNotFound
	}
	return nil
}

func (r *OrganizationRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error) {
	query := `
		SELECT o.id, o.name, o.slug, o.timezone, o.visibility, o.max_loan_duration_days, o.status, o.created_at, o.updated_at, m.role
		FROM organizations o
		INNER JOIN memberships m ON m.organization_id = o.id
		WHERE m.user_id = $1
		ORDER BY o.name ASC
	`
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var result []domain.UserOrganization
	for rows.Next() {
		var item domain.UserOrganization
		err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Timezone,
			&item.Visibility,
			&item.MaxLoanDurationDays,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Role,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
