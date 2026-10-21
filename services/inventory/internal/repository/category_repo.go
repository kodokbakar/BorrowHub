package repository

import (
	"context"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CategoryRepository struct {
	pool *pgxpool.Pool
}

func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) Create(ctx context.Context, c *domain.Category) error {
	query := `
		INSERT INTO categories (id, organization_id, name, slug, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		c.ID,
		c.OrganizationID,
		c.Name,
		c.Slug,
		c.Description,
		c.CreatedAt,
		c.UpdatedAt,
	)
	return mapPgError(err)
}

func (r *CategoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	query := `
		SELECT id, organization_id, name, slug, description, created_at, updated_at
		FROM categories
		WHERE id = $1
	`
	var c domain.Category
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID,
		&c.OrganizationID,
		&c.Name,
		&c.Slug,
		&c.Description,
		&c.CreatedAt,
		&c.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &c, nil
}

func (r *CategoryRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Category, error) {
	query := `
		SELECT id, organization_id, name, slug, description, created_at, updated_at
		FROM categories
		WHERE organization_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var categories []*domain.Category
	for rows.Next() {
		var c domain.Category
		if err := rows.Scan(
			&c.ID,
			&c.OrganizationID,
			&c.Name,
			&c.Slug,
			&c.Description,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		categories = append(categories, &c)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	if categories == nil {
		categories = []*domain.Category{}
	}
	return categories, nil
}

func (r *CategoryRepository) Update(ctx context.Context, c *domain.Category) error {
	query := `
		UPDATE categories
		SET name = $2, slug = $3, description = $4, updated_at = $5
		WHERE id = $1
	`
	cmd, err := r.pool.Exec(ctx, query, c.ID, c.Name, c.Slug, c.Description, c.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CategoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM categories WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
