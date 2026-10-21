package repository

import (
	"errors"

	"borrowhub/services/inventory/internal/domain"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code == pgerrcode.UniqueViolation {
			switch pgErr.ConstraintName {
			case "uq_units_org_barcode":
				return domain.ErrDuplicateBarcode
			case "uq_units_org_serial":
				return domain.ErrDuplicateSerial
			case "uq_categories_org_slug":
				return domain.ErrDuplicateSlug
			default:
				return domain.ErrConflict
			}
		}
		if pgErr.Code == pgerrcode.ForeignKeyViolation || pgErr.Code == pgerrcode.RestrictViolation {
			switch pgErr.ConstraintName {
			case "items_category_id_fkey":
				return domain.ErrCategoryInUse
			case "units_item_id_fkey":
				return domain.ErrItemInUse
			default:
				return domain.ErrConflict
			}
		}
	}
	return err
}

type Repositories struct {
	Categories *CategoryRepository
	Items      *ItemRepository
	Units      *UnitRepository
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Categories: NewCategoryRepository(pool),
		Items:      NewItemRepository(pool),
		Units:      NewUnitRepository(pool),
	}
}
