package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
)

const InvitationTTL = 7 * 24 * time.Hour

type InviteInput struct {
	Email string      `json:"email"`
	Role  domain.Role `json:"role"`
}

type MemberService struct {
	users       UserRepository
	orgs        OrganizationRepository
	memberships MembershipRepository
	invitations InvitationRepository
}

func NewMemberService(
	users UserRepository,
	orgs OrganizationRepository,
	memberships MembershipRepository,
	invitations InvitationRepository,
) *MemberService {
	return &MemberService{
		users:       users,
		orgs:        orgs,
		memberships: memberships,
		invitations: invitations,
	}
}

func (s *MemberService) ListMembers(
	ctx context.Context,
	orgID uuid.UUID,
	userID uuid.UUID,
) ([]domain.MembershipDetail, error) {
	_, err := s.memberships.GetByOrgAndUser(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return nil, domain.ErrForbidden
		}
		return nil, err
	}

	return s.memberships.ListByOrgID(ctx, orgID)
}

func (s *MemberService) Invite(
	ctx context.Context,
	orgID uuid.UUID,
	inviterUserID uuid.UUID,
	input InviteInput,
) (*domain.Invitation, string, error) {
	mem, err := s.memberships.GetByOrgAndUser(ctx, orgID, inviterUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return nil, "", domain.ErrForbidden
		}
		return nil, "", err
	}

	if !mem.Role.CanManageMembers() {
		return nil, "", domain.ErrForbidden
	}

	normEmail, err := domain.ValidateEmail(input.Email)
	if err != nil {
		return nil, "", err
	}

	if err := domain.ValidateInvitationRole(input.Role); err != nil {
		return nil, "", err
	}

	existingUser, err := s.users.GetByEmail(ctx, normEmail)
	if err == nil && existingUser != nil {
		existingMem, err := s.memberships.GetByOrgAndUser(ctx, orgID, existingUser.ID)
		if err == nil && existingMem != nil {
			return nil, "", domain.ErrAlreadyMember
		}
	}
	pendingInv, err := s.invitations.GetPendingByOrgAndEmail(ctx, orgID, normEmail)
	if err == nil && pendingInv != nil {
		pendingInv.Status = domain.InvitationStatusRevoked
		pendingInv.UpdatedAt = time.Now()
		if err := s.invitations.Update(ctx, pendingInv); err != nil {
			return nil, "", err
		}
	} else if err != nil && !errors.Is(err, domain.ErrNotFound) && !errors.Is(err, domain.ErrInvitationNotFound) {
		return nil, "", err
	}

	rawToken, err := auth.GenerateSecureToken(32)
	if err != nil {
		return nil, "", err
	}

	tokenHash := auth.HashToken(rawToken)

	invID, err := uuid.NewV7()
	if err != nil {
		invID = uuid.New()
	}

	now := time.Now()
	inv := &domain.Invitation{
		ID:              invID,
		OrganizationID:  orgID,
		Email:           normEmail,
		Role:            input.Role,
		TokenHash:       tokenHash,
		InvitedByUserID: &inviterUserID,
		Status:          domain.InvitationStatusPending,
		ExpiresAt:       now.Add(InvitationTTL),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.invitations.Create(ctx, inv); err != nil {
		return nil, "", err
	}

	return inv, rawToken, nil
}

func (s *MemberService) AcceptInvitation(
	ctx context.Context,
	rawToken string,
	userID uuid.UUID,
) (*domain.Membership, error) {
	if strings.TrimSpace(rawToken) == "" {
		return nil, domain.ErrInvitationNotFound
	}

	tokenHash := auth.HashToken(rawToken)
	inv, err := s.invitations.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrInvitationNotFound
	}

	if inv.Status != domain.InvitationStatusPending {
		return nil, domain.ErrInvitationAlreadyUsed
	}

	now := time.Now()
	if now.After(inv.ExpiresAt) {
		inv.Status = domain.InvitationStatusExpired
		inv.UpdatedAt = now
		_ = s.invitations.Update(ctx, inv)
		return nil, domain.ErrInvitationExpired
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !strings.EqualFold(user.Email, inv.Email) {
		return nil, domain.ErrForbidden
	}

	existingMem, err := s.memberships.GetByOrgAndUser(ctx, inv.OrganizationID, userID)
	if err == nil && existingMem != nil {
		return nil, domain.ErrAlreadyMember
	}

	memberID, err := uuid.NewV7()
	if err != nil {
		memberID = uuid.New()
	}

	membership := &domain.Membership{
		ID:             memberID,
		OrganizationID: inv.OrganizationID,
		UserID:         userID,
		Role:           inv.Role,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.invitations.AcceptInvitation(ctx, inv.ID, now, membership); err != nil {
		return nil, err
	}

	return membership, nil
}

func (s *MemberService) ChangeRole(
	ctx context.Context,
	orgID uuid.UUID,
	operatorUserID uuid.UUID,
	memberID uuid.UUID,
	newRole domain.Role,
) error {
	opMem, err := s.memberships.GetByOrgAndUser(ctx, orgID, operatorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return domain.ErrForbidden
		}
		return err
	}

	if opMem.Role != domain.RoleOwner {
		return domain.ErrForbidden
	}

	if err := domain.ValidateRole(newRole); err != nil {
		return err
	}

	targetMem, err := s.memberships.GetByID(ctx, memberID)
	if err != nil {
		return err
	}

	if targetMem.OrganizationID != orgID {
		return domain.ErrMembershipNotFound
	}

	if targetMem.Role == domain.RoleOwner && newRole != domain.RoleOwner {
		count, err := s.memberships.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrCannotDemoteSoleOwner
		}
	}

	return s.memberships.UpdateRole(ctx, memberID, newRole)
}

func (s *MemberService) RemoveMember(
	ctx context.Context,
	orgID uuid.UUID,
	operatorUserID uuid.UUID,
	memberID uuid.UUID,
) error {
	opMem, err := s.memberships.GetByOrgAndUser(ctx, orgID, operatorUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrMembershipNotFound) {
			return domain.ErrForbidden
		}
		return err
	}

	if !opMem.Role.CanManageMembers() {
		return domain.ErrForbidden
	}

	targetMem, err := s.memberships.GetByID(ctx, memberID)
	if err != nil {
		return err
	}

	if targetMem.OrganizationID != orgID {
		return domain.ErrMembershipNotFound
	}

	if opMem.Role == domain.RoleAdmin && (targetMem.Role == domain.RoleOwner || targetMem.Role == domain.RoleAdmin) {
		return domain.ErrForbidden
	}

	if targetMem.Role == domain.RoleOwner {
		count, err := s.memberships.CountOwners(ctx, orgID)
		if err != nil {
			return err
		}
		if count <= 1 {
			return domain.ErrCannotRemoveSoleOwner
		}
	}

	return s.memberships.Delete(ctx, memberID)
}

func (s *MemberService) GetMemberForUser(
	ctx context.Context,
	orgID uuid.UUID,
	targetUserID uuid.UUID,
) (*domain.Membership, error) {
	return s.memberships.GetByOrgAndUser(ctx, orgID, targetUserID)
}
