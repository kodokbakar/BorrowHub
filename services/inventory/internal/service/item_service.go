package service

import (
	"context"
	"time"

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

type ItemService struct {
	itemRepo     ItemRepository
	categoryRepo CategoryRepository
	identity     auth.IdentityClient
}

func NewItemService(
	itemRepo ItemRepository,
	categoryRepo CategoryRepository,
	identity auth.IdentityClient,
) *ItemService {
	return &ItemService{
		itemRepo:     itemRepo,
		categoryRepo: categoryRepo,
		identity:     identity,
	}
}

func (s *ItemService) Create(
	ctx context.Context,
	orgID, userID, categoryID uuid.UUID,
	name string,
	description *string,
	isLendable bool,
) (*domain.Item, error) {
	mem, err := s.identity.GetMember(ctx, orgID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrForbidden
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return nil, domain.ErrForbidden
	}

	cat, err := s.categoryRepo.GetByID(ctx, categoryID)
	if err != nil || cat.OrganizationID != orgID {
		return nil, domain.ErrCategoryNotFound
	}

	if err := domain.ValidateItem(name, orgID, categoryID, description); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	now := time.Now()

	item := &domain.Item{
		ID:             id,
		OrganizationID: orgID,
		CategoryID:     categoryID,
		Name:           name,
		Description:    description,
		IsLendable:     isLendable,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.itemRepo.Create(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ItemService) GetByID(ctx context.Context, itemID, userID uuid.UUID) (*domain.Item, error) {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return nil, domain.ErrItemNotFound
	}

	mem, err := s.identity.GetMember(ctx, item.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrItemNotFound
	}

	return item, nil
}

func (s *ItemService) ListByOrg(ctx context.Context, orgID, userID uuid.UUID) ([]*domain.Item, error) {
	mem, err := s.identity.GetMember(ctx, orgID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrForbidden
	}

	return s.itemRepo.ListByOrg(ctx, orgID)
}

func (s *ItemService) Update(
	ctx context.Context,
	itemID, userID uuid.UUID,
	categoryID *uuid.UUID,
	name *string,
	description *string,
	isLendable *bool,
) (*domain.Item, error) {
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

	newCatID := item.CategoryID
	if categoryID != nil {
		cat, err := s.categoryRepo.GetByID(ctx, *categoryID)
		if err != nil || cat.OrganizationID != item.OrganizationID {
			return nil, domain.ErrCategoryNotFound
		}
		newCatID = *categoryID
	}

	newName := item.Name
	if name != nil {
		newName = *name
	}

	newDesc := item.Description
	if description != nil {
		newDesc = description
	}

	newLendable := item.IsLendable
	if isLendable != nil {
		newLendable = *isLendable
	}

	if err := domain.ValidateItem(newName, item.OrganizationID, newCatID, newDesc); err != nil {
		return nil, err
	}

	item.CategoryID = newCatID
	item.Name = newName
	item.Description = newDesc
	item.IsLendable = newLendable
	item.UpdatedAt = time.Now()

	if err := s.itemRepo.Update(ctx, item); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *ItemService) Delete(ctx context.Context, itemID, userID uuid.UUID) error {
	item, err := s.itemRepo.GetByID(ctx, itemID)
	if err != nil {
		return domain.ErrItemNotFound
	}

	mem, err := s.identity.GetMember(ctx, item.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return domain.ErrItemNotFound
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return domain.ErrForbidden
	}

	return s.itemRepo.Delete(ctx, itemID)
}
