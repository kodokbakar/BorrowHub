package service

import (
	"context"
	"errors"
	"time"

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

type UnitService struct {
	unitRepo UnitRepository
	itemRepo ItemRepository
	identity auth.IdentityClient
}

func NewUnitService(
	unitRepo UnitRepository,
	itemRepo ItemRepository,
	identity auth.IdentityClient,
) *UnitService {
	return &UnitService{
		unitRepo: unitRepo,
		itemRepo: itemRepo,
		identity: identity,
	}
}

func (s *UnitService) Create(
	ctx context.Context,
	itemID, userID uuid.UUID,
	barcode string,
	serialNumber *string,
	status *domain.UnitStatus,
	condition *domain.UnitCondition,
	notes *string,
) (*domain.Unit, error) {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, domain.ErrItemNotFound
	}

	mem, err := s.identity.GetMember(ctx, item.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrItemNotFound
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return nil, domain.ErrForbidden
	}

	st := domain.StatusAvailable
	if status != nil {
		st = *status
	}

	cond := domain.ConditionNew
	if condition != nil {
		cond = *condition
	}

	normBarcode, normSerial, err := domain.ValidateUnit(
		barcode,
		serialNumber,
		st,
		cond,
		item.OrganizationID,
		itemID,
		notes,
	)
	if err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	now := time.Now()

	unit := &domain.Unit{
		ID:             id,
		OrganizationID: item.OrganizationID,
		ItemID:         itemID,
		Barcode:        normBarcode,
		SerialNumber:   normSerial,
		Status:         st,
		Condition:      cond,
		Notes:          notes,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.unitRepo.Create(ctx, unit, userID); err != nil {
		return nil, err
	}

	return unit, nil
}

func (s *UnitService) GetByID(ctx context.Context, unitID, userID uuid.UUID) (*domain.Unit, error) {
	unit, err := s.unitRepo.GetByID(ctx, unitID)
	if err != nil {
		return nil, domain.ErrUnitNotFound
	}

	mem, err := s.identity.GetMember(ctx, unit.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrUnitNotFound
	}

	return unit, nil
}

func (s *UnitService) ListByItem(ctx context.Context, itemID, userID uuid.UUID) ([]*domain.Unit, error) {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, domain.ErrItemNotFound
	}

	mem, err := s.identity.GetMember(ctx, item.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrItemNotFound
	}

	return s.unitRepo.ListByItem(ctx, itemID)
}

func (s *UnitService) Update(
	ctx context.Context,
	unitID, userID uuid.UUID,
	barcode *string,
	serialNumber *string,
	status *domain.UnitStatus,
	condition *domain.UnitCondition,
	notes *string,
	reason *string,
) (*domain.Unit, error) {
	unit, err := s.unitRepo.GetByID(ctx, unitID)
	if err != nil {
		return nil, domain.ErrUnitNotFound
	}

	mem, err := s.identity.GetMember(ctx, unit.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrUnitNotFound
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return nil, domain.ErrForbidden
	}

	input := domain.UpdateUnitInput{
		Barcode:      barcode,
		SerialNumber: serialNumber,
		Status:       status,
		Condition:    condition,
		Notes:        notes,
		Reason:       reason,
	}

	normBarcode, normSerial, err := domain.ValidateUpdateUnitInput(input)
	if err != nil {
		return nil, err
	}
	input.Barcode = normBarcode
	input.SerialNumber = normSerial

	if input.Status != nil && !domain.CanAdminTransition(unit.Status, *input.Status) {
		return nil, domain.ErrInvalidTransition
	}

	updatedUnit, err := s.unitRepo.Update(ctx, unitID, input, &userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnitNotFound
		}
		return nil, err
	}

	return updatedUnit, nil
}

func (s *UnitService) GetConditionLogs(ctx context.Context, unitID, userID uuid.UUID) ([]*domain.ConditionLog, error) {
	unit, err := s.unitRepo.GetByID(ctx, unitID)
	if err != nil {
		return nil, domain.ErrUnitNotFound
	}

	mem, err := s.identity.GetMember(ctx, unit.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrUnitNotFound
	}

	return s.unitRepo.GetConditionLogs(ctx, unitID)
}
