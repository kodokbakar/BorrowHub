package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
)

type CreateOrgInput struct {
	Name                string
	Slug                string
	Timezone            string
	Visibility          string
	MaxLoanDurationDays int
}

type UpdateOrgInput struct {
	Name                *string
	Timezone            *string
	Visibility          *string
	MaxLoanDurationDays *int
}

type OrgService struct {
	orgs        OrganizationRepository
	memberships MembershipRepository
}

func NewOrgService(orgs OrganizationRepository, memberships MembershipRepository) *OrgService {
	return &OrgService{
		orgs:        orgs,
		memberships: memberships,
	}
}

func (s *OrgService) Create(
	ctx context.Context,
	userID uuid.UUID,
	input CreateOrgInput,
) (*domain.Organization, error) {
	if err := domain.ValidateOrgName(input.Name); err != nil {
		return nil, err
	}
	if err := domain.ValidateSlug(input.Slug); err != nil {
		return nil, err
	}

	tz := strings.TrimSpace(input.Timezone)
	if tz == "" {
		tz = "UTC"
	}

	vis := strings.TrimSpace(input.Visibility)
	if vis == "" {
		vis = domain.OrgVisibilityPrivate
	} else if vis != domain.OrgVisibilityPrivate && vis != domain.OrgVisibilityPublic {
		return nil, domain.NewValidationError("invalid visibility", map[string]any{
			"visibility": "must be private or public",
		})
	}

	maxDuration := input.MaxLoanDurationDays
	if maxDuration == 0 {
		maxDuration = 14
	} else if err := domain.ValidateMaxLoanDuration(maxDuration); err != nil {
		return nil, err
	}

	orgID, err := uuid.NewV7()
	if err != nil {
		orgID = uuid.New()
	}

	memberID, err := uuid.NewV7()
	if err != nil {
		memberID = uuid.New()
	}

	now := time.Now()
	org := &domain.Organization{
		ID:                  orgID,
		Name:                strings.TrimSpace(input.Name),
		Slug:                strings.ToLower(strings.TrimSpace(input.Slug)),
		Timezone:            tz,
		Visibility:          vis,
		MaxLoanDurationDays: maxDuration,
		Status:              domain.OrgStatusActive,
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	ownerMembership := &domain.Membership{
		ID:             memberID,
		OrganizationID: orgID,
		UserID:         userID,
		Role:           domain.RoleOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.orgs.CreateWithInitialOwner(ctx, org, ownerMembership); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrgService) Get(
	ctx context.Context,
	orgID uuid.UUID,
	userID uuid.UUID,
) (*domain.Organization, error) {
	_, err := s.memberships.GetByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}

	return s.orgs.GetByID(ctx, orgID)
}

func (s *OrgService) Update(
	ctx context.Context,
	orgID uuid.UUID,
	userID uuid.UUID,
	input UpdateOrgInput,
) (*domain.Organization, error) {
	mem, err := s.memberships.GetByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}

	if !mem.Role.CanManageMembers() {
		return nil, domain.ErrForbidden
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		if err := domain.ValidateOrgName(*input.Name); err != nil {
			return nil, err
		}
		org.Name = strings.TrimSpace(*input.Name)
	}

	if input.Timezone != nil {
		org.Timezone = strings.TrimSpace(*input.Timezone)
	}

	if input.Visibility != nil {
		vis := strings.TrimSpace(*input.Visibility)
		if vis != domain.OrgVisibilityPrivate && vis != domain.OrgVisibilityPublic {
			return nil, domain.NewValidationError("invalid visibility", map[string]any{
				"visibility": "must be private or public",
			})
		}
		org.Visibility = vis
	}

	if input.MaxLoanDurationDays != nil {
		if err := domain.ValidateMaxLoanDuration(*input.MaxLoanDurationDays); err != nil {
			return nil, err
		}
		org.MaxLoanDurationDays = *input.MaxLoanDurationDays
	}

	org.UpdatedAt = time.Now()
	if err := s.orgs.Update(ctx, org); err != nil {
		return nil, err
	}

	return org, nil
}

func (s *OrgService) ListUserOrgs(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error) {
	return s.orgs.ListByUserID(ctx, userID)
}
