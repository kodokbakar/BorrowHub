package repository

import (
	"context"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ItemRepository struct {
	pool *pgxpool.Pool
}

func NewItemRepository(pool *pgxpool.Pool) *ItemRepository {
	return &ItemRepository{pool: pool}
}

func (r *ItemRepository) Create(ctx context.Context, item *domain.Item) error {
	query := `
		INSERT INTO items (id, organization_id, category_id, name, description, is_lendable, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		item.ID,
		item.OrganizationID,
		item.CategoryID,
		item.Name,
		item.Description,
		item.IsLendable,
		item.CreatedAt,
		item.UpdatedAt,
	)
	return mapPgError(err)
}

func (r *ItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	query := `
		SELECT id, organization_id, category_id, name, description, is_lendable, created_at, updated_at
		FROM items
		WHERE id = $1
	`
	var item domain.Item
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&item.ID,
		&item.OrganizationID,
		&item.CategoryID,
		&item.Name,
		&item.Description,
		&item.IsLendable,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, mapPgError(err)
	}
	return &item, nil
}

func (r *ItemRepository) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Item, error) {
	query := `
		SELECT id, organization_id, category_id, name, description, is_lendable, created_at, updated_at
		FROM items
		WHERE organization_id = $1
		ORDER BY name ASC
	`
	rows, err := r.pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var items []*domain.Item
	for rows.Next() {
		var item domain.Item
		if err := rows.Scan(
			&item.ID,
			&item.OrganizationID,
			&item.CategoryID,
			&item.Name,
			&item.Description,
			&item.IsLendable,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		items = append(items, &item)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPgError(err)
	}
	if items == nil {
		items = []*domain.Item{}
	}
	return items, nil
}

func (r *ItemRepository) Update(ctx context.Context, item *domain.Item) error {
	query := `
		UPDATE items
		SET category_id = $2, name = $3, description = $4, is_lendable = $5, updated_at = $6
		WHERE id = $1
	`
	cmd, err := r.pool.Exec(
		ctx,
		query,
		item.ID,
		item.CategoryID,
		item.Name,
		item.Description,
		item.IsLendable,
		item.UpdatedAt,
	)
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *ItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM items WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapPgError(err)
	}
	if cmd.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
