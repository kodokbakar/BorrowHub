package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

func (r Role) IsValid() bool {
	return r == RoleOwner || r == RoleAdmin || r == RoleMember
}

func (r Role) CanManageMembers() bool {
	return r == RoleOwner || r == RoleAdmin
}

const (
	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"

	OrgStatusActive    = "active"
	OrgStatusSuspended = "suspended"

	OrgVisibilityPrivate = "private"
	OrgVisibilityPublic  = "public"

	InvitationStatusPending  = "pending"
	InvitationStatusAccepted = "accepted"
	InvitationStatusRevoked  = "revoked"
	InvitationStatusExpired  = "expired"
)

type User struct {
	ID              uuid.UUID `json:"id"`
	Email           string    `json:"email"`
	PasswordHash    string    `json:"-"`
	DisplayName     string    `json:"display_name"`
	Status          string    `json:"status"`
	TermsAcceptedAt time.Time `json:"terms_accepted_at"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Organization struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Slug                string    `json:"slug"`
	Timezone            string    `json:"timezone"`
	Visibility          string    `json:"visibility"`
	MaxLoanDurationDays int       `json:"max_loan_duration_days"`
	Status              string    `json:"status"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type Membership struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Role           Role      `json:"role"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type MembershipDetail struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	UserID         uuid.UUID `json:"user_id"`
	Role           Role      `json:"role"`
	Email          string    `json:"email"`
	DisplayName    string    `json:"display_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserOrganization struct {
	Organization
	Role Role `json:"role"`
}

type Invitation struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	Email           string     `json:"email"`
	Role            Role       `json:"role"`
	TokenHash       string     `json:"-"`
	InvitedByUserID *uuid.UUID `json:"invited_by_user_id,omitempty"`
	Status          string     `json:"status"`
	ExpiresAt       time.Time  `json:"expires_at"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type Session struct {
	ID               uuid.UUID  `json:"id"`
	UserID           uuid.UUID  `json:"user_id"`
	RefreshTokenHash string     `json:"-"`
	UserAgent        string     `json:"user_agent"`
	IPAddress        string     `json:"ip_address"`
	ExpiresAt        time.Time  `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
