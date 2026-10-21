package repository

import (
	"errors"

	"borrowhub/services/identity/internal/domain"
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
			case "users_email_key", "uq_users_email", "idx_users_email":
				return domain.ErrEmailAlreadyExists
			case "organizations_slug_key", "uq_organizations_slug", "idx_organizations_slug":
				return domain.ErrSlugAlreadyExists
			case "uq_memberships_org_user":
				return domain.ErrAlreadyMember
			default:
				return domain.ErrConflict
			}
		}
	}
	return err
}

type Repositories struct {
	Users         *UserRepository
	Organizations *OrganizationRepository
	Memberships   *MembershipRepository
	Invitations   *InvitationRepository
	Sessions      *SessionRepository
}

func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Users:         NewUserRepository(pool),
		Organizations: NewOrganizationRepository(pool),
		Memberships:   NewMembershipRepository(pool),
		Invitations:   NewInvitationRepository(pool),
		Sessions:      NewSessionRepository(pool),
	}
}
