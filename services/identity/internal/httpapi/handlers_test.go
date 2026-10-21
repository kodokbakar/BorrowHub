package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"borrowhub/services/identity/internal/auth"
	"borrowhub/services/identity/internal/domain"
	"borrowhub/services/identity/internal/service"
	"github.com/google/uuid"
)

type memUserRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*domain.User
}

func (r *memUserRepo) Create(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.users {
		if strings.EqualFold(existing.Email, u.Email) {
			return domain.ErrEmailAlreadyExists
		}
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *memUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, domain.ErrUserNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *memUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.users {
		if strings.EqualFold(u.Email, email) {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrUserNotFound
}

func (r *memUserRepo) Update(ctx context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[u.ID]; !ok {
		return domain.ErrUserNotFound
	}
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

type memOrgRepo struct {
	mu   sync.Mutex
	orgs map[uuid.UUID]*domain.Organization
	mems *memMemRepo
}

func (r *memOrgRepo) Create(ctx context.Context, org *domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.orgs {
		if o.Slug == org.Slug {
			return domain.ErrSlugAlreadyExists
		}
	}
	cp := *org
	r.orgs[org.ID] = &cp
	return nil
}

func (r *memOrgRepo) CreateWithInitialOwner(ctx context.Context, org *domain.Organization, owner *domain.Membership) error {
	if err := r.Create(ctx, org); err != nil {
		return err
	}
	return r.mems.Create(ctx, owner)
}

func (r *memOrgRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	o, ok := r.orgs[id]
	if !ok {
		return nil, domain.ErrOrgNotFound
	}
	cp := *o
	return &cp, nil
}

func (r *memOrgRepo) GetBySlug(ctx context.Context, slug string) (*domain.Organization, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range r.orgs {
		if o.Slug == slug {
			cp := *o
			return &cp, nil
		}
	}
	return nil, domain.ErrOrgNotFound
}

func (r *memOrgRepo) Update(ctx context.Context, org *domain.Organization) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.orgs[org.ID]; !ok {
		return domain.ErrOrgNotFound
	}
	cp := *org
	r.orgs[org.ID] = &cp
	return nil
}

func (r *memOrgRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]domain.UserOrganization, error) {
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

type memMemRepo struct {
	mu      sync.Mutex
	members map[uuid.UUID]*domain.Membership
	users   *memUserRepo
}

func (r *memMemRepo) Create(ctx context.Context, m *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.members {
		if existing.OrganizationID == m.OrganizationID && existing.UserID == m.UserID {
			return domain.ErrAlreadyMember
		}
	}
	cp := *m
	r.members[m.ID] = &cp
	return nil
}

func (r *memMemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.members[id]
	if !ok {
		return nil, domain.ErrMembershipNotFound
	}
	cp := *m
	return &cp, nil
}

func (r *memMemRepo) GetByOrgAndUser(ctx context.Context, orgID, userID uuid.UUID) (*domain.Membership, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, m := range r.members {
		if m.OrganizationID == orgID && m.UserID == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, domain.ErrMembershipNotFound
}

func (r *memMemRepo) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]domain.MembershipDetail, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var res []domain.MembershipDetail
	for _, m := range r.members {
		if m.OrganizationID == orgID {
			d := domain.MembershipDetail{
				ID:             m.ID,
				OrganizationID: m.OrganizationID,
				UserID:         m.UserID,
				Role:           m.Role,
				CreatedAt:      m.CreatedAt,
				UpdatedAt:      m.UpdatedAt,
			}
			if u, ok := r.users.users[m.UserID]; ok {
				d.Email = u.Email
				d.DisplayName = u.DisplayName
			}
			res = append(res, d)
		}
	}
	return res, nil
}

func (r *memMemRepo) CountOwners(ctx context.Context, orgID uuid.UUID) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	cnt := 0
	for _, m := range r.members {
		if m.OrganizationID == orgID && m.Role == domain.RoleOwner {
			cnt++
		}
	}
	return cnt, nil
}

func (r *memMemRepo) UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error {
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

func (r *memMemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.members[id]; !ok {
		return domain.ErrMembershipNotFound
	}
	delete(r.members, id)
	return nil
}

type memInvRepo struct {
	mu   sync.Mutex
	invs map[uuid.UUID]*domain.Invitation
	mems *memMemRepo
}

func (r *memInvRepo) Create(ctx context.Context, inv *domain.Invitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *inv
	r.invs[inv.ID] = &cp
	return nil
}

func (r *memInvRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invs {
		if inv.TokenHash == tokenHash {
			cp := *inv
			return &cp, nil
		}
	}
	return nil, domain.ErrInvitationNotFound
}

func (r *memInvRepo) GetPendingByOrgAndEmail(ctx context.Context, orgID uuid.UUID, email string) (*domain.Invitation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inv := range r.invs {
		if inv.OrganizationID == orgID && strings.EqualFold(inv.Email, email) && inv.Status == domain.InvitationStatusPending {
			cp := *inv
			return &cp, nil
		}
	}
	return nil, domain.ErrInvitationNotFound
}

func (r *memInvRepo) Update(ctx context.Context, inv *domain.Invitation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.invs[inv.ID]; !ok {
		return domain.ErrInvitationNotFound
	}
	cp := *inv
	r.invs[inv.ID] = &cp
	return nil
}

func (r *memInvRepo) AcceptInvitation(ctx context.Context, invitationID uuid.UUID, acceptedAt time.Time, m *domain.Membership) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	inv, ok := r.invs[invitationID]
	if !ok || inv.Status != domain.InvitationStatusPending {
		return domain.ErrInvitationAlreadyUsed
	}
	inv.Status = domain.InvitationStatusAccepted
	inv.AcceptedAt = &acceptedAt
	inv.UpdatedAt = acceptedAt
	return r.mems.Create(ctx, m)
}

type memSessRepo struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]*domain.Session
}

func (r *memSessRepo) Create(ctx context.Context, s *domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *s
	r.sessions[s.ID] = &cp
	return nil
}

func (r *memSessRepo) GetByRefreshTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range r.sessions {
		if s.RefreshTokenHash == hash {
			cp := *s
			return &cp, nil
		}
	}
	return nil, domain.ErrSessionNotFound
}

func (r *memSessRepo) Revoke(ctx context.Context, id uuid.UUID, revokedAt time.Time) (bool, error) {
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

func (r *memSessRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID, revokedAt time.Time) error {
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

type testRouterContext struct {
	router   http.Handler
	jwtMgr   *auth.JWTManager
	userRepo *memUserRepo
	orgRepo  *memOrgRepo
	memRepo  *memMemRepo
	invRepo  *memInvRepo
	sessRepo *memSessRepo
}

func setupFullTestRouter(ready bool) testRouterContext {
	jwtMgr := auth.NewJWTManager("test-jwt-secret-at-least-32-bytes-long!")
	hasher := auth.FastPasswordHasher()

	userRepo := &memUserRepo{users: make(map[uuid.UUID]*domain.User)}
	memRepo := &memMemRepo{members: make(map[uuid.UUID]*domain.Membership), users: userRepo}
	orgRepo := &memOrgRepo{orgs: make(map[uuid.UUID]*domain.Organization), mems: memRepo}
	invRepo := &memInvRepo{invs: make(map[uuid.UUID]*domain.Invitation), mems: memRepo}
	sessRepo := &memSessRepo{sessions: make(map[uuid.UUID]*domain.Session)}

	authSvc := service.NewAuthService(userRepo, orgRepo, sessRepo, hasher, jwtMgr)
	orgSvc := service.NewOrgService(orgRepo, memRepo)
	memberSvc := service.NewMemberService(userRepo, orgRepo, memRepo, invRepo)

	router := NewRouter(RouterDependencies{
		AuthService:   authSvc,
		OrgService:    orgSvc,
		MemberService: memberSvc,
		TokenManager:  jwtMgr,
		ReadyCheck: func() bool {
			return ready
		},
		IsSecure:      false,
		InternalToken: "test-internal-token",
	})

	return testRouterContext{
		router:   RequestIDs(router),
		jwtMgr:   jwtMgr,
		userRepo: userRepo,
		orgRepo:  orgRepo,
		memRepo:  memRepo,
		invRepo:  invRepo,
		sessRepo: sessRepo,
	}
}

func setupTestRouter(ready bool) (http.Handler, *auth.JWTManager) {
	ctx := setupFullTestRouter(ready)
	return ctx.router, ctx.jwtMgr
}

func TestHealthEndpoints(t *testing.T) {
	router, _ := setupTestRouter(true)

	// Live
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for live, got %d", w.Code)
	}

	// Ready - healthy
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for ready, got %d", w.Code)
	}

	// Ready - unready
	unreadyRouter, _ := setupTestRouter(false)
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w = httptest.NewRecorder()
	unreadyRouter.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for unready, got %d", w.Code)
	}
}

func TestAuthAndOrgHTTPFlow(t *testing.T) {
	router, _ := setupTestRouter(true)

	// 1. Register Owner
	registerBody := `{"email":"owner@borrowhub.test","password":"Password123!","display_name":"Alice","terms_accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(registerBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 created, got %d: %s", w.Code, w.Body.String())
	}

	var regResp struct {
		Data struct {
			User struct {
				ID    string `json:"id"`
				Email string `json:"email"`
			} `json:"user"`
			Tokens struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &regResp); err != nil {
		t.Fatal(err)
	}

	ownerToken := regResp.Data.Tokens.AccessToken
	ownerID := regResp.Data.User.ID

	// 2. Register Second User
	memberRegBody := `{"email":"bob@borrowhub.test","password":"Password123!","display_name":"Bob","terms_accepted":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(memberRegBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for second user, got %d", w.Code)
	}
	var bobRegResp struct {
		Data struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &bobRegResp); err != nil {
		t.Fatal(err)
	}
	bobToken := bobRegResp.Data.Tokens.AccessToken
	bobID := bobRegResp.Data.User.ID

	// 3. GET /api/v1/me (without token -> 401)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 unauthorized, got %d", w.Code)
	}

	// 4. GET /api/v1/me (with token -> 200)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 5. PATCH /api/v1/me (update display name)
	patchMeBody := `{"display_name":"Alice Updated"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", strings.NewReader(patchMeBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK on update me, got %d: %s", w.Code, w.Body.String())
	}

	// 6. POST /api/v1/organizations (create org)
	createOrgBody := `{"name":"Makerspace","slug":"makerspace","timezone":"UTC","visibility":"private","max_loan_duration_days":14}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(createOrgBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on create org, got %d: %s", w.Code, w.Body.String())
	}

	var orgResp struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &orgResp); err != nil {
		t.Fatal(err)
	}
	orgID := orgResp.Data.ID

	// 7. GET /api/v1/organizations
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for list orgs, got %d", w.Code)
	}

	// 8. GET /api/v1/organizations/{organizationId}
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for get org, got %d", w.Code)
	}

	// 9. PATCH /api/v1/organizations/{organizationId}
	patchOrgBody := `{"name":"Makerspace HQ"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+orgID, strings.NewReader(patchOrgBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for patch org, got %d: %s", w.Code, w.Body.String())
	}

	// 10. POST /api/v1/organizations/{organizationId}/invitations
	invBody := `{"email":"bob@borrowhub.test","role":"member"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID+"/invitations", strings.NewReader(invBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for invitation, got %d: %s", w.Code, w.Body.String())
	}

	var invResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &invResp); err != nil {
		t.Fatal(err)
	}
	invToken := invResp.Data.Token

	// 11. POST /api/v1/invitations/{token}/accept (Bob accepts)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/"+invToken+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+bobToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on accept invitation, got %d: %s", w.Code, w.Body.String())
	}

	var acceptResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &acceptResp); err != nil {
		t.Fatal(err)
	}
	bobMemberID := acceptResp.Data.ID

	// 12. GET /api/v1/organizations/{organizationId}/members
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for list members, got %d", w.Code)
	}

	// 13. PATCH /api/v1/organizations/{organizationId}/members/{memberId} (Promote Bob to Admin)
	roleBody := `{"role":"admin"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+orgID+"/members/"+bobMemberID, strings.NewReader(roleBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for change role, got %d: %s", w.Code, w.Body.String())
	}

	// 14. GET /api/v1/internal/organizations/{organizationId}/members/{userId}
	// Unauthenticated / missing internal token -> 401
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/internal/organizations/%s/members/%s", orgID, bobID), nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for internal member check without token, got %d", w.Code)
	}

	// Invalid internal token -> 401
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/internal/organizations/%s/members/%s", orgID, bobID), nil)
	req.Header.Set("X-Internal-Token", "wrong-internal-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for internal member check with wrong token, got %d", w.Code)
	}

	// Valid internal token -> 200
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/internal/organizations/%s/members/%s", orgID, bobID), nil)
	req.Header.Set("X-Internal-Token", "test-internal-token")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for internal member check with token, got %d: %s", w.Code, w.Body.String())
	}
	var internalResp struct {
		Data struct {
			IsMember bool   `json:"is_member"`
			Role     string `json:"role"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &internalResp); err != nil {
		t.Fatal(err)
	}
	if !internalResp.Data.IsMember || internalResp.Data.Role != "admin" {
		t.Fatalf("unexpected internal response: %+v", internalResp)
	}

	// 15. DELETE /api/v1/organizations/{organizationId}/members/{memberId} (Remove Bob)
	req = httptest.NewRequest(http.MethodDelete, "/api/v1/organizations/"+orgID+"/members/"+bobMemberID, nil)
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for remove member, got %d: %s", w.Code, w.Body.String())
	}

	// 16. Refresh & Logout
	refreshBody := fmt.Sprintf(`{"refresh_token":%q}`, regResp.Data.Tokens.RefreshToken)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(refreshBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for refresh, got %d: %s", w.Code, w.Body.String())
	}

	var newRefreshResp struct {
		Data struct {
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &newRefreshResp)

	logoutBody := fmt.Sprintf(`{"refresh_token":%q}`, newRefreshResp.Data.RefreshToken)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewBufferString(logoutBody))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for logout, got %d", w.Code)
	}

	_ = ownerID
}

func TestHTTP_ExpiredInvitation(t *testing.T) {
	tc := setupFullTestRouter(true)

	// 1. Register Owner
	regBody := `{"email":"owner_exp@borrowhub.test","password":"Password123!","display_name":"OwnerExp","terms_accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regBody))
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var regResp struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &regResp)
	ownerToken := regResp.Data.Tokens.AccessToken

	// 2. Create Org
	orgBody := `{"name":"Exp Org","slug":"exp-org"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(orgBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var orgResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &orgResp)
	orgID := orgResp.Data.ID

	// 3. Invite Invitee
	inviteBody := `{"email":"invitee_exp@borrowhub.test","role":"member"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgID+"/invitations", strings.NewReader(inviteBody))
	req.Header.Set("Authorization", "Bearer "+ownerToken)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for invite, got %d", w.Code)
	}
	var invResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &invResp)
	rawToken := invResp.Data.Token

	// 4. Manually expire the invitation in repository
	tc.invRepo.mu.Lock()
	for _, inv := range tc.invRepo.invs {
		if inv.Email == "invitee_exp@borrowhub.test" {
			inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
		}
	}
	tc.invRepo.mu.Unlock()

	// 5. Register Invitee
	inviteeRegBody := `{"email":"invitee_exp@borrowhub.test","password":"Password123!","display_name":"InviteeExp","terms_accepted":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(inviteeRegBody))
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var inviteeRegResp struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &inviteeRegResp)
	inviteeToken := inviteeRegResp.Data.Tokens.AccessToken

	// 6. Invitee accepts expired invitation -> must return 410 Gone with INVITATION_EXPIRED
	req = httptest.NewRequest(http.MethodPost, "/api/v1/invitations/"+rawToken+"/accept", nil)
	req.Header.Set("Authorization", "Bearer "+inviteeToken)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusGone {
		t.Fatalf("expected 410 Gone for expired invitation, got %d: %s", w.Code, w.Body.String())
	}
	var errResp ErrorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &errResp); err != nil {
		t.Fatal(err)
	}
	if errResp.Error.Code != "INVITATION_EXPIRED" {
		t.Fatalf("expected error code INVITATION_EXPIRED, got %s", errResp.Error.Code)
	}
}

func TestHTTP_TenantIsolation_CrossOrg(t *testing.T) {
	tc := setupFullTestRouter(true)

	// Register User A
	regA := `{"email":"user_a@borrowhub.test","password":"Password123!","display_name":"Alice","terms_accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regA))
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	var respA struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &respA)
	tokenA := respA.Data.Tokens.AccessToken

	// Register User B
	regB := `{"email":"user_b@borrowhub.test","password":"Password123!","display_name":"Bob","terms_accepted":true}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regB))
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	var respB struct {
		Data struct {
			Tokens struct {
				AccessToken string `json:"access_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &respB)
	tokenB := respB.Data.Tokens.AccessToken

	// User B creates Org B
	createOrgB := `{"name":"Org B","slug":"org-b"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations", strings.NewReader(createOrgB))
	req.Header.Set("Authorization", "Bearer "+tokenB)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	var orgBResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &orgBResp)
	orgBID := orgBResp.Data.ID

	// User A tries to GET Org B
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgBID, nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for User A accessing Org B, got %d: %s", w.Code, w.Body.String())
	}

	// User A tries to PATCH Org B
	patchBody := `{"name":"Compromised Org"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/organizations/"+orgBID, strings.NewReader(patchBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for User A updating Org B, got %d: %s", w.Code, w.Body.String())
	}

	// User A tries to list Org B members
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgBID+"/members", nil)
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for User A listing Org B members, got %d: %s", w.Code, w.Body.String())
	}

	// User A tries to invite to Org B
	invBody := `{"email":"eve@borrowhub.test","role":"member"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgBID+"/invitations", strings.NewReader(invBody))
	req.Header.Set("Authorization", "Bearer "+tokenA)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for User A inviting to Org B, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_PasswordChangeRevokesSessions(t *testing.T) {
	tc := setupFullTestRouter(true)

	// 1. Register User
	regBody := `{"email":"pwd_user@borrowhub.test","password":"OldPassword123!","display_name":"PwdUser","terms_accepted":true}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(regBody))
	w := httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	var regResp struct {
		Data struct {
			Tokens struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
			} `json:"tokens"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &regResp)
	token := regResp.Data.Tokens.AccessToken
	oldRefreshToken := regResp.Data.Tokens.RefreshToken

	// 2. Change password via PATCH /api/v1/me
	updateMeBody := `{"current_password":"OldPassword123!","new_password":"NewPassword123!"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/me", strings.NewReader(updateMeBody))
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for update password, got %d: %s", w.Code, w.Body.String())
	}

	// 3. Attempt to refresh with old refresh token -> must fail with 401
	refreshBody := fmt.Sprintf(`{"refresh_token":%q}`, oldRefreshToken)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(refreshBody))
	w = httptest.NewRecorder()
	tc.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for old refresh token after password change, got %d: %s", w.Code, w.Body.String())
	}
}

func TestHTTP_DecodeJSON_MaxBytesLimit(t *testing.T) {
	router, _ := setupTestRouter(true)

	// Large payload exceeding 1MB
	largeEmail := strings.Repeat("a", 1024*1024+10)
	body := fmt.Sprintf(`{"email":%q,"password":"Password123!","display_name":"Big","terms_accepted":true}`, largeEmail)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", strings.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for payload > 1MB, got %d", w.Code)
	}
}
