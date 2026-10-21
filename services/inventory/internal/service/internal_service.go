package service

import (
	"context"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

type InternalInventoryService struct {
	unitRepo UnitRepository
	itemRepo ItemRepository
}

func NewInternalInventoryService(
	unitRepo UnitRepository,
	itemRepo ItemRepository,
) *InternalInventoryService {
	return &InternalInventoryService{
		unitRepo: unitRepo,
		itemRepo: itemRepo,
	}
}

func (s *InternalInventoryService) ListEligibleUnits(
	ctx context.Context,
	itemID uuid.UUID,
) ([]*domain.Unit, error) {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, domain.ErrItemNotFound
	}

	if !item.IsLendable {
		return []*domain.Unit{}, nil
	}

	return s.unitRepo.ListEligibleByItem(ctx, itemID)
}

func (s *InternalInventoryService) TransitionUnitStatus(
	ctx context.Context,
	unitID uuid.UUID,
	fromStatus, toStatus domain.UnitStatus,
) (*domain.Unit, error) {
	if !domain.CanLendingTransition(fromStatus, toStatus) {
		return nil, domain.ErrInvalidTransition
	}

	return s.unitRepo.TransitionStatus(ctx, unitID, fromStatus, toStatus)
}
