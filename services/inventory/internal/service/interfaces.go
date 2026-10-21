package service

import (
	"context"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *domain.Category) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Category, error)
	Update(ctx context.Context, c *domain.Category) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ItemRepository interface {
	Create(ctx context.Context, item *domain.Item) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Item, error)
	Update(ctx context.Context, item *domain.Item) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type UnitRepository interface {
	Create(ctx context.Context, u *domain.Unit, createdBy uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error)
	ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error)
	ListEligibleByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error)
	Update(ctx context.Context, unitID uuid.UUID, input domain.UpdateUnitInput, changedBy *uuid.UUID) (*domain.Unit, error)
	TransitionStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus domain.UnitStatus) (*domain.Unit, error)
	GetConditionLogs(ctx context.Context, unitID uuid.UUID) ([]*domain.ConditionLog, error)
}
