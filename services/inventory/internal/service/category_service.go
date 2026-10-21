package service

import (
	"context"
	"time"

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

type CategoryService struct {
	repo     CategoryRepository
	identity auth.IdentityClient
}

func NewCategoryService(repo CategoryRepository, identity auth.IdentityClient) *CategoryService {
	return &CategoryService{
		repo:     repo,
		identity: identity,
	}
}

func (s *CategoryService) Create(
	ctx context.Context,
	orgID, userID uuid.UUID,
	name, slug string,
	description *string,
) (*domain.Category, error) {
	mem, err := s.identity.GetMember(ctx, orgID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrForbidden
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return nil, domain.ErrForbidden
	}

	if err := domain.ValidateCategory(name, slug, description); err != nil {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		id = uuid.New()
	}
	now := time.Now()

	category := &domain.Category{
		ID:             id,
		OrganizationID: orgID,
		Name:           name,
		Slug:           slug,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *CategoryService) ListByOrg(
	ctx context.Context,
	orgID, userID uuid.UUID,
) ([]*domain.Category, error) {
	mem, err := s.identity.GetMember(ctx, orgID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrForbidden
	}

	return s.repo.ListByOrg(ctx, orgID)
}

func (s *CategoryService) Update(
	ctx context.Context,
	categoryID, userID uuid.UUID,
	name, slug *string,
	description *string,
) (*domain.Category, error) {
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return nil, domain.ErrCategoryNotFound
	}

	mem, err := s.identity.GetMember(ctx, category.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return nil, domain.ErrCategoryNotFound
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return nil, domain.ErrForbidden
	}

	newName := category.Name
	if name != nil {
		newName = *name
	}

	newSlug := category.Slug
	if slug != nil {
		newSlug = *slug
	}

	newDesc := category.Description
	if description != nil {
		newDesc = description
	}

	if err := domain.ValidateCategory(newName, newSlug, newDesc); err != nil {
		return nil, err
	}

	category.Name = newName
	category.Slug = newSlug
	category.Description = newDesc
	category.UpdatedAt = time.Now()

	if err := s.repo.Update(ctx, category); err != nil {
		return nil, err
	}

	return category, nil
}

func (s *CategoryService) Delete(ctx context.Context, categoryID, userID uuid.UUID) error {
	category, err := s.repo.GetByID(ctx, categoryID)
	if err != nil {
		return domain.ErrCategoryNotFound
	}

	mem, err := s.identity.GetMember(ctx, category.OrganizationID, userID)
	if err != nil || !mem.IsMember {
		return domain.ErrCategoryNotFound
	}
	if mem.Role != "admin" && mem.Role != "owner" {
		return domain.ErrForbidden
	}

	return s.repo.Delete(ctx, categoryID)
}
