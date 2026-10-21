package service

import (
	"context"
	"time"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
}

type OrganizationRepository interface {
	Create(ctx context.Context, org *domain.Organization) error
	CreateWithInitialOwner(ctx context.Context, org *domain.Organization, ownerMembership *domain.Membership) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Organization, error)
	Update(ctx context.Context, org *domain.Organization) error
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error)
}

type MembershipRepository interface {
	Create(ctx context.Context, membership *domain.Membership) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error)
	GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.Membership, error)
	ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]domain.MembershipDetail, error)
	CountOwners(ctx context.Context, orgID uuid.UUID) (int, error)
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type InvitationRepository interface {
	Create(ctx context.Context, inv *domain.Invitation) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error)
	GetPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.Invitation, error)
	Update(ctx context.Context, inv *domain.Invitation) error
	AcceptInvitation(ctx context.Context, invitationID uuid.UUID, acceptedAt time.Time, membership *domain.Membership) error
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) (bool, error)
	RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	VerifyPassword(password, encodedHash string) (bool, error)
	DummyVerify(password string)
}

type TokenManager interface {
	GenerateAccessToken(userID, email string, ttl time.Duration) (string, error)
	ValidateAccessToken(tokenStr string) (*auth.Claims, error)
}
