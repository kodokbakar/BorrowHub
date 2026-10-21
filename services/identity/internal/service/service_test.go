package service

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"github.com/google/uuid"
)

type memoryUserRepository struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func newMemoryUserRepo() *memoryUserRepository {
	return &memoryUserRepository{users: make(map[uuid.UUID]*domain.User)}
}

func (r *memoryUserRepository) Create(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return domain.ErrEmailAlreadyExists
		}
	}
	copy := *u
	r.users[u.ID] = &copy
	return nil
}

func (r *memoryUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	copy := *u
	return &copy, nil
}

func (r *memoryUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			copy := *u
			return &copy, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memoryUserRepository) Update(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	copy := *u
	r.users[u.ID] = &copy
	return nil
}

type memoryOrgRepository struct {
	mu   sync.Mutex
	orgs map[uuid.UUID]*domain.Organization
	mems *memoryMembershipRepository
}

func newMemoryOrgRepo(mems *memoryMembershipRepository) *memoryOrgRepository {
	return &memoryOrgRepository{
		orgs: make(map[uuid.UUID]*domain.Organization),
		mems: mems,
	}
}

func (r *memoryOrgRepository) Create(ctx context.Context, org *domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.orgs {
		if o.Slug == org.Slug {
			return domain.ErrSlugAlreadyExists
		}
	}
	copy := *org
	r.orgs[org.ID] = &copy
	return nil
}

func (r *memoryOrgRepository) CreateWithInitialOwner(
	ctx context.Context,
	org *domain.Organization,
	ownerMembership *domain.Membership,
) error {
	if err := r.Create(ctx, org); err != nil {
		return err
	}
	return r.mems.Create(ctx, ownerMembership)
}

func (r *memoryOrgRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	org, ok := r.orgs[id]
	if !ok {
		return nil, domain.ErrOrgNotFound
	}
	copy := *org
	return &copy, nil
}

func (r *memoryOrgRepository) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.orgs {
		if o.Slug == slug {
			copy := *o
			return &copy, nil
		}
	}
	return nil, domain.ErrOrgNotFound
}

func (r *memoryOrgRepository) Update(ctx context.Context, org *domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.orgs[org.ID]; !ok {
		return domain.ErrOrgNotFound
	}
	copy := *org
	r.orgs[org.ID] = &copy
	return nil
}

func (r *memoryOrgRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []domain.UserOrganization
	for _, m := range r.mems.members {
		if m.UserID == userID {
			if o, ok := r.orgs[m.OrganizationID]; ok {
				res = append(res, domain.UserOrganization{
					Organization: *o,
					Role:         m.Role,
				})
			}
		}
	}
	return res, nil
}

type memoryMembershipRepository struct {
	mu      sync.Mutex
	members map[uuid.UUID]*domain.Membership
	users   *memoryUserRepository
}

func newMemoryMembershipRepo(users *memoryUserRepository) *memoryMembershipRepository {
	return &memoryMembershipRepository{
		members: make(map[uuid.UUID]*domain.Membership),
		users:   users,
	}
}

func (r *memoryMembershipRepository) Create(ctx context.Context, m *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.members {
		if existing.OrganizationID == m.OrganizationID && existing.UserID == m.UserID {
			return domain.ErrAlreadyMember
		}
	}
	copy := *m
	r.members[m.ID] = &copy
	return nil
}

func (r *memoryMembershipRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.members[id]
	if !ok {
		return nil, domain.ErrMembershipNotFound
	}
	copy := *m
	return &copy, nil
}

func (r *memoryMembershipRepository) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members {
		if m.OrganizationID == orgID && m.UserID == userID {
			copy := *m
			return &copy, nil
		}
	}
	return nil, domain.ErrMembershipNotFound
}

func (r *memoryMembershipRepository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]domain.MembershipDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []domain.MembershipDetail
	for _, m := range r.members {
		if m.OrganizationID == orgID {
			detail := domain.MembershipDetail{
				ID:             m.ID,
				OrganizationID: m.OrganizationID,
				UserID:         m.UserID,
				Role:           m.Role,
				CreatedAt:      m.CreatedAt,
				UpdatedAt:      m.UpdatedAt,
			}
			if u, ok := r.users.users[m.UserID]; ok {
				detail.Email = u.Email
				detail.DisplayName = u.DisplayName
			}
			res = append(res, detail)
		}
	}
	return res, nil
}

func (r *memoryMembershipRepository) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, m := range r.members {
		if m.OrganizationID == orgID && m.Role == domain.RoleOwner {
			count++
		}
	}
	return count, nil
}

func (r *memoryMembershipRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.members[id]
	if !ok {
		return domain.ErrMembershipNotFound
	}
	m.Role = role
	m.UpdatedAt = time.Now()
	return nil
}

func (r *memoryMembershipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.members[id]; !ok {
		return domain.ErrMembershipNotFound
	}
	delete(r.members, id)
	return nil
}

type memoryInvitationRepository struct {
	mu          sync.Mutex
	invitations map[uuid.UUID]*domain.Invitation
	mems        *memoryMembershipRepository
}

func newMemoryInvitationRepo(mems *memoryMembershipRepository) *memoryInvitationRepository {
	return &memoryInvitationRepository{
		invitations: make(map[uuid.UUID]*domain.Invitation),
		mems:        mems,
	}
}

func (r *memoryInvitationRepository) Create(ctx context.Context, inv *domain.Invitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *inv
	r.invitations[inv.ID] = &copy
	return nil
}

func (r *memoryInvitationRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invitations {
		if inv.TokenHash == tokenHash {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, domain.ErrInvitationNotFound
}

func (r *memoryInvitationRepository) GetPendingByOrgAndEmail(
	ctx context.Context,
	orgID uuid.UUID,
	email string,
) (*domain.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invitations {
		if inv.OrganizationID == orgID && strings.EqualFold(inv.Email, email) && inv.Status == domain.InvitationStatusPending {
			copy := *inv
			return &copy, nil
		}
	}
	return nil, domain.ErrInvitationNotFound
}

func (r *memoryInvitationRepository) Update(ctx context.Context, inv *domain.Invitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.invitations[inv.ID]; !ok {
		return domain.ErrInvitationNotFound
	}
	copy := *inv
	r.invitations[inv.ID] = &copy
	return nil
}

func (r *memoryInvitationRepository) AcceptInvitation(
	ctx context.Context,
	invitationID uuid.UUID,
	acceptedAt time.Time,
	membership *domain.Membership,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invitations[invitationID]
	if !ok || inv.Status != domain.InvitationStatusPending {
		return domain.ErrInvitationAlreadyUsed
	}
	inv.Status = domain.InvitationStatusAccepted
	inv.AcceptedAt = &acceptedAt
	inv.UpdatedAt = acceptedAt

	return r.mems.Create(ctx, membership)
}

type memorySessionRepository struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*domain.Session
}

func newMemorySessionRepo() *memorySessionRepository {
	return &memorySessionRepository{sessions: make(map[uuid.UUID]*domain.Session)}
}

func (r *memorySessionRepository) Create(ctx context.Context, s *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *s
	r.sessions[s.ID] = &copy
	return nil
}

func (r *memorySessionRepository) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if s.RefreshTokenHash == hash {
			copy := *s
			return &copy, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}

func (r *memorySessionRepository) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.sessions[id]
	if !ok || s.RevokedAt != nil {
		return false, nil
	}
	s.RevokedAt = &revokedAt
	s.UpdatedAt = revokedAt
	return true, nil
}

func (r *memorySessionRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if s.UserID == userID && s.RevokedAt == nil {
			s.RevokedAt = &revokedAt
			s.UpdatedAt = revokedAt
		}
	}
	return nil
}

func setupTestServices() (*AuthService, *OrgService, *MemberService, *memoryUserRepository) {
	hasher := auth.FastPasswordHasher()
	tokens := auth.NewJWTManager("test-secret-key-32-bytes-minimum-length!")

	userRepo := newMemoryUserRepo()
	memRepo := newMemoryMembershipRepo(userRepo)
	orgRepo := newMemoryOrgRepo(memRepo)
	invRepo := newMemoryInvitationRepo(memRepo)
	sessionRepo := newMemorySessionRepo()

	authSvc := NewAuthService(userRepo, orgRepo, sessionRepo, hasher, tokens)
	orgSvc := NewOrgService(orgRepo, memRepo)
	memberSvc := NewMemberService(userRepo, orgRepo, memRepo, invRepo)

	return authSvc, orgSvc, memberSvc, userRepo
}

func setupAllTestServices() (*AuthService, *OrgService, *MemberService, *memoryUserRepository, *memoryInvitationRepository, *memorySessionRepository) {
	hasher := auth.FastPasswordHasher()
	tokens := auth.NewJWTManager("test-secret-key-32-bytes-minimum-length!")

	userRepo := newMemoryUserRepo()
	memRepo := newMemoryMembershipRepo(userRepo)
	orgRepo := newMemoryOrgRepo(memRepo)
	invRepo := newMemoryInvitationRepo(memRepo)
	sessionRepo := newMemorySessionRepo()

	authSvc := NewAuthService(userRepo, orgRepo, sessionRepo, hasher, tokens)
	orgSvc := NewOrgService(orgRepo, memRepo)
	memberSvc := NewMemberService(userRepo, orgRepo, memRepo, invRepo)

	return authSvc, orgSvc, memberSvc, userRepo, invRepo, sessionRepo
}

func TestAuthService_RegisterAndLogin(t *testing.T) {
	authSvc, _, _, _ := setupTestServices()
	ctx := context.Background()

	// 1. Missing terms
	_, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "user@example.com",
		Password:      "SecurePassword123!",
		DisplayName:   "User One",
		TermsAccepted: false,
	})
	if !errors.Is(err, domain.ErrTermsNotAccepted) {
		t.Fatalf("expected ErrTermsNotAccepted, got: %v", err)
	}

	// 2. Successful Register
	user, authRes, err := authSvc.Register(ctx, RegisterInput{
		Email:         "User@Example.Com",
		Password:      "SecurePassword123!",
		DisplayName:   "User One",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatalf("unexpected register error: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected lowercase email, got: %s", user.Email)
	}
	if authRes.AccessToken == "" || authRes.RefreshToken == "" {
		t.Fatal("expected non-empty tokens")
	}

	// 3. Duplicate Register
	_, _, err = authSvc.Register(ctx, RegisterInput{
		Email:         "user@example.com",
		Password:      "AnotherPassword123!",
		DisplayName:   "User Two",
		TermsAccepted: true,
	})
	if !errors.Is(err, domain.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got: %v", err)
	}

	// 4. Successful Login
	loggedUser, loggedAuth, err := authSvc.Login(ctx, LoginInput{
		Email:    "USER@example.com",
		Password: "SecurePassword123!",
	})
	if err != nil {
		t.Fatalf("unexpected login error: %v", err)
	}
	if loggedUser.ID != user.ID {
		t.Fatalf("expected matching user ID")
	}
	if loggedAuth.AccessToken == "" {
		t.Fatal("expected access token")
	}

	// 5. Login wrong password
	_, _, err = authSvc.Login(ctx, LoginInput{
		Email:    "user@example.com",
		Password: "WrongPassword!",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got: %v", err)
	}

	// 6. Login non-existent email (dummy verification)
	_, _, err = authSvc.Login(ctx, LoginInput{
		Email:    "unknown@example.com",
		Password: "SomePassword!",
	})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for unknown user, got: %v", err)
	}

	// 7. Refresh token
	refreshed, err := authSvc.Refresh(ctx, loggedAuth.RefreshToken, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("expected tokens on refresh")
	}

	// 8. Reusing old refresh token must fail
	_, err = authSvc.Refresh(ctx, loggedAuth.RefreshToken, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on revoked refresh token, got: %v", err)
	}

	// 9. Logout
	if err := authSvc.Logout(ctx, refreshed.RefreshToken); err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}
	_, err = authSvc.Refresh(ctx, refreshed.RefreshToken, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized after logout, got: %v", err)
	}
}

func TestOrgAndMemberService_LifecycleAndSoleOwner(t *testing.T) {
	authSvc, orgSvc, memberSvc, userRepo := setupTestServices()
	ctx := context.Background()

	// Create 2 users
	owner, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "owner@example.com",
		Password:      "OwnerPass123!",
		DisplayName:   "Owner User",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	memberUser, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "member@example.com",
		Password:      "MemberPass123!",
		DisplayName:   "Member User",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Create Organization
	org, err := orgSvc.Create(ctx, owner.ID, CreateOrgInput{
		Name: "Acme Labs",
		Slug: "acme-labs",
	})
	if err != nil {
		t.Fatalf("unexpected create org error: %v", err)
	}

	// Verify owner's org list
	orgs, err := orgSvc.ListUserOrgs(ctx, owner.ID)
	if err != nil || len(orgs) != 1 || orgs[0].Role != domain.RoleOwner {
		t.Fatalf("expected 1 owner org, got: %+v", orgs)
	}

	// Non-member cannot access org
	_, err = orgSvc.Get(ctx, org.ID, memberUser.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-member, got: %v", err)
	}

	// Owner invites memberUser
	inv, token, err := memberSvc.Invite(ctx, org.ID, owner.ID, InviteInput{
		Email: "member@example.com",
		Role:  domain.RoleMember,
	})
	if err != nil {
		t.Fatalf("unexpected invite error: %v", err)
	}
	if inv.Status != domain.InvitationStatusPending || token == "" {
		t.Fatal("invalid invitation state")
	}

	// Non-member accepts invitation
	membership, err := memberSvc.AcceptInvitation(ctx, token, memberUser.ID)
	if err != nil {
		t.Fatalf("unexpected accept invitation error: %v", err)
	}
	if membership.Role != domain.RoleMember {
		t.Fatalf("expected RoleMember, got %s", membership.Role)
	}

	// Accepting already accepted token must fail
	_, err = memberSvc.AcceptInvitation(ctx, token, memberUser.ID)
	if !errors.Is(err, domain.ErrInvitationAlreadyUsed) {
		t.Fatalf("expected ErrInvitationAlreadyUsed, got: %v", err)
	}

	// List members
	members, err := memberSvc.ListMembers(ctx, org.ID, owner.ID)
	if err != nil || len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}

	// Find owner membership ID
	var ownerMembershipID uuid.UUID
	for _, m := range members {
		if m.UserID == owner.ID {
			ownerMembershipID = m.ID
		}
	}

	// Sole owner cannot be demoted!
	err = memberSvc.ChangeRole(ctx, org.ID, owner.ID, ownerMembershipID, domain.RoleAdmin)
	if !errors.Is(err, domain.ErrCannotDemoteSoleOwner) {
		t.Fatalf("expected ErrCannotDemoteSoleOwner, got: %v", err)
	}

	// Sole owner cannot be removed!
	err = memberSvc.RemoveMember(ctx, org.ID, owner.ID, ownerMembershipID)
	if !errors.Is(err, domain.ErrCannotRemoveSoleOwner) {
		t.Fatalf("expected ErrCannotRemoveSoleOwner, got: %v", err)
	}

	// Promote memberUser to Owner
	err = memberSvc.ChangeRole(ctx, org.ID, owner.ID, membership.ID, domain.RoleOwner)
	if err != nil {
		t.Fatalf("unexpected promote to owner error: %v", err)
	}

	// Now there are 2 owners, original owner CAN be demoted to admin
	err = memberSvc.ChangeRole(ctx, org.ID, owner.ID, ownerMembershipID, domain.RoleAdmin)
	if err != nil {
		t.Fatalf("expected demotion to succeed with 2 owners, got: %v", err)
	}

	// Demoted owner (now admin) cannot remove remaining owner!
	err = memberSvc.RemoveMember(ctx, org.ID, owner.ID, membership.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden when admin tries to remove owner, got: %v", err)
	}

	_ = userRepo
}

func TestMemberService_InvitationExpiration(t *testing.T) {
	authSvc, orgSvc, memberSvc, _, invRepo, _ := setupAllTestServices()
	ctx := context.Background()

	owner, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "owner@test.com",
		Password:      "Password123!",
		DisplayName:   "Owner",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	invitee, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "invitee@test.com",
		Password:      "Password123!",
		DisplayName:   "Invitee",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	org, err := orgSvc.Create(ctx, owner.ID, CreateOrgInput{
		Name: "Test Org",
		Slug: "test-org",
	})
	if err != nil {
		t.Fatal(err)
	}

	inv, token, err := memberSvc.Invite(ctx, org.ID, owner.ID, InviteInput{
		Email: "invitee@test.com",
		Role:  domain.RoleMember,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Expire the invitation
	inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if err := invRepo.Update(ctx, inv); err != nil {
		t.Fatal(err)
	}

	// Attempt to accept expired invitation -> ErrInvitationExpired
	_, err = memberSvc.AcceptInvitation(ctx, token, invitee.ID)
	if !errors.Is(err, domain.ErrInvitationExpired) {
		t.Fatalf("expected ErrInvitationExpired, got: %v", err)
	}

	// Verify status updated to expired
	updatedInv, err := invRepo.GetByTokenHash(ctx, inv.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if updatedInv.Status != domain.InvitationStatusExpired {
		t.Fatalf("expected status expired, got: %s", updatedInv.Status)
	}
}

func TestMemberService_ReInviteRevokesPrevious(t *testing.T) {
	authSvc, orgSvc, memberSvc, _, invRepo, _ := setupAllTestServices()
	ctx := context.Background()

	owner, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "owner@test.com",
		Password:      "Password123!",
		DisplayName:   "Owner",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	invitee, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "invitee@test.com",
		Password:      "Password123!",
		DisplayName:   "Invitee",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	org, err := orgSvc.Create(ctx, owner.ID, CreateOrgInput{
		Name: "Test Org",
		Slug: "test-org",
	})
	if err != nil {
		t.Fatal(err)
	}

	// First invitation
	inv1, token1, err := memberSvc.Invite(ctx, org.ID, owner.ID, InviteInput{
		Email: "invitee@test.com",
		Role:  domain.RoleMember,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Re-invite same email
	_, token2, err := memberSvc.Invite(ctx, org.ID, owner.ID, InviteInput{
		Email: "invitee@test.com",
		Role:  domain.RoleAdmin,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Previous invitation must be revoked
	updatedInv1, err := invRepo.GetByTokenHash(ctx, inv1.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if updatedInv1.Status != domain.InvitationStatusRevoked {
		t.Fatalf("expected status revoked for first invitation, got: %s", updatedInv1.Status)
	}

	// Accepting token1 fails
	_, err = memberSvc.AcceptInvitation(ctx, token1, invitee.ID)
	if !errors.Is(err, domain.ErrInvitationAlreadyUsed) {
		t.Fatalf("expected ErrInvitationAlreadyUsed for revoked invitation, got: %v", err)
	}

	// Accepting token2 succeeds
	mem, err := memberSvc.AcceptInvitation(ctx, token2, invitee.ID)
	if err != nil {
		t.Fatalf("unexpected error accepting second invitation: %v", err)
	}
	if mem.Role != domain.RoleAdmin {
		t.Fatalf("expected RoleAdmin, got: %s", mem.Role)
	}
}

func TestTenantIsolation_CrossOrgAccess(t *testing.T) {
	authSvc, orgSvc, memberSvc, _, _, _ := setupAllTestServices()
	ctx := context.Background()

	userA, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "usera@test.com",
		Password:      "Password123!",
		DisplayName:   "User A",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	userB, _, err := authSvc.Register(ctx, RegisterInput{
		Email:         "userb@test.com",
		Password:      "Password123!",
		DisplayName:   "User B",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	orgA, err := orgSvc.Create(ctx, userA.ID, CreateOrgInput{Name: "Org A", Slug: "org-a"})
	if err != nil {
		t.Fatal(err)
	}

	orgB, err := orgSvc.Create(ctx, userB.ID, CreateOrgInput{Name: "Org B", Slug: "org-b"})
	if err != nil {
		t.Fatal(err)
	}

	// User A attempts to access Org B resources -> all must return ErrForbidden
	_, err = orgSvc.Get(ctx, orgB.ID, userA.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B Get, got: %v", err)
	}

	newName := "Hacked Name"
	_, err = orgSvc.Update(ctx, orgB.ID, userA.ID, UpdateOrgInput{Name: &newName})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B Update, got: %v", err)
	}

	_, err = memberSvc.ListMembers(ctx, orgB.ID, userA.ID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B ListMembers, got: %v", err)
	}

	_, _, err = memberSvc.Invite(ctx, orgB.ID, userA.ID, InviteInput{Email: "someone@test.com", Role: domain.RoleMember})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B Invite, got: %v", err)
	}

	dummyMemberID := uuid.New()
	err = memberSvc.ChangeRole(ctx, orgB.ID, userA.ID, dummyMemberID, domain.RoleAdmin)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B ChangeRole, got: %v", err)
	}

	err = memberSvc.RemoveMember(ctx, orgB.ID, userA.ID, dummyMemberID)
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for Org B RemoveMember, got: %v", err)
	}

	// Org A accessing Org A should succeed
	gotOrgA, err := orgSvc.Get(ctx, orgA.ID, userA.ID)
	if err != nil || gotOrgA.ID != orgA.ID {
		t.Fatalf("expected success for Org A Get, got: %v", err)
	}
}

func TestAuthService_PasswordChangeRevokesSessions(t *testing.T) {
	authSvc, _, _, _, _, _ := setupAllTestServices()
	ctx := context.Background()

	user, authRes, err := authSvc.Register(ctx, RegisterInput{
		Email:         "pwdchange@test.com",
		Password:      "OldPassword123!",
		DisplayName:   "Pwd User",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// Update password
	currPwd := "OldPassword123!"
	newPwd := "NewPassword123!"
	_, err = authSvc.UpdateMe(ctx, user.ID, UpdateMeInput{
		CurrentPassword: &currPwd,
		NewPassword:     &newPwd,
	})
	if err != nil {
		t.Fatalf("unexpected error on UpdateMe: %v", err)
	}

	// Using old refresh token must be rejected
	_, err = authSvc.Refresh(ctx, authRes.RefreshToken, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized for refresh token after password change, got: %v", err)
	}

	// Login with new password works
	_, newAuthRes, err := authSvc.Login(ctx, LoginInput{
		Email:    "pwdchange@test.com",
		Password: "NewPassword123!",
	})
	if err != nil {
		t.Fatalf("login with new password failed: %v", err)
	}

	// Refresh with new token works
	_, err = authSvc.Refresh(ctx, newAuthRes.RefreshToken, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("refresh with new token failed: %v", err)
	}
}

func TestAuthService_RefreshTokenRotationAndReuse(t *testing.T) {
	authSvc, _, _, _, _, _ := setupAllTestServices()
	ctx := context.Background()

	_, authRes, err := authSvc.Register(ctx, RegisterInput{
		Email:         "reuse@test.com",
		Password:      "Password123!",
		DisplayName:   "Reuse User",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	rt1 := authRes.RefreshToken

	// First refresh rotates rt1 -> rt2
	refreshed, err := authSvc.Refresh(ctx, rt1, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("first refresh failed: %v", err)
	}
	rt2 := refreshed.RefreshToken

	// Replay rt1 -> token reuse detected, should return ErrUnauthorized and revoke all
	_, err = authSvc.Refresh(ctx, rt1, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on reused rt1, got: %v", err)
	}

	// Subsequent refresh with rt2 must also fail because token reuse revoked all sessions
	_, err = authSvc.Refresh(ctx, rt2, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on rt2 after reuse revocation, got: %v", err)
	}
}

func TestAuthService_RefreshTokenNaturalExpiration(t *testing.T) {
	authSvc, _, _, _, _, sessRepo := setupAllTestServices()
	ctx := context.Background()

	_, authRes1, err := authSvc.Register(ctx, RegisterInput{
		Email:         "expire@test.com",
		Password:      "Password123!",
		DisplayName:   "Expire User",
		TermsAccepted: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, authRes2, err := authSvc.Login(ctx, LoginInput{
		Email:    "expire@test.com",
		Password: "Password123!",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Manually expire session 1
	h1 := auth.HashToken(authRes1.RefreshToken)
	s1, err := sessRepo.GetByRefreshTokenHash(ctx, h1)
	if err != nil {
		t.Fatal(err)
	}
	sessRepo.mu.Lock()
	sessRepo.sessions[s1.ID].ExpiresAt = time.Now().Add(-1 * time.Hour)
	sessRepo.mu.Unlock()

	// Refreshing expired session 1 should fail with ErrUnauthorized
	_, err = authSvc.Refresh(ctx, authRes1.RefreshToken, "agent", "127.0.0.1")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized on expired token, got: %v", err)
	}

	// Session 2 should NOT be revoked and should successfully refresh
	refreshed2, err := authSvc.Refresh(ctx, authRes2.RefreshToken, "agent", "127.0.0.1")
	if err != nil {
		t.Fatalf("expected session 2 refresh to succeed after session 1 natural expiration, got: %v", err)
	}
	if refreshed2.AccessToken == "" {
		t.Fatal("expected new access token")
	}
}
