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

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"borrowhub/services/inventory/internal/service"
	"github.com/google/uuid"
)

// In-memory repositories for handler testing

type mockCategoryRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Category
}

func newMockCategoryRepo() *mockCategoryRepo {
	return &mockCategoryRepo{data: make(map[uuid.UUID]*domain.Category)}
}

func (r *mockCategoryRepo) Create(ctx context.Context, c *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.data {
		if existing.OrganizationID == c.OrganizationID && existing.Slug == c.Slug {
			return domain.ErrDuplicateSlug
		}
	}
	cp := *c
	r.data[c.ID] = &cp
	return nil
}

func (r *mockCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *mockCategoryRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*domain.Category
	for _, c := range r.data {
		if c.OrganizationID == orgID {
			cp := *c
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (r *mockCategoryRepo) Update(ctx context.Context, c *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[c.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *c
	r.data[c.ID] = &cp
	return nil
}

func (r *mockCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.data, id)
	return nil
}

type mockItemRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Item
}

func newMockItemRepo() *mockItemRepo {
	return &mockItemRepo{data: make(map[uuid.UUID]*domain.Item)}
}

func (r *mockItemRepo) Create(ctx context.Context, item *domain.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *item
	r.data[item.ID] = &cp
	return nil
}

func (r *mockItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *item
	return &cp, nil
}

func (r *mockItemRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*domain.Item
	for _, item := range r.data {
		if item.OrganizationID == orgID {
			cp := *item
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (r *mockItemRepo) Update(ctx context.Context, item *domain.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[item.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *item
	r.data[item.ID] = &cp
	return nil
}

func (r *mockItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.data, id)
	return nil
}

type mockUnitRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Unit
	logs map[uuid.UUID][]*domain.ConditionLog
}

func newMockUnitRepo() *mockUnitRepo {
	return &mockUnitRepo{
		data: make(map[uuid.UUID]*domain.Unit),
		logs: make(map[uuid.UUID][]*domain.ConditionLog),
	}
}

func (r *mockUnitRepo) Create(ctx context.Context, u *domain.Unit, createdBy uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.data {
		if existing.OrganizationID == u.OrganizationID && existing.Barcode == u.Barcode {
			return domain.ErrDuplicateBarcode
		}
	}
	cp := *u
	r.data[u.ID] = &cp
	r.logs[u.ID] = append(r.logs[u.ID], &domain.ConditionLog{
		ID:           uuid.New(),
		UnitID:       u.ID,
		NewCondition: u.Condition,
		ChangedBy:    createdBy,
		CreatedAt:    time.Now(),
	})
	return nil
}

func (r *mockUnitRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *mockUnitRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*domain.Unit
	for _, u := range r.data {
		if u.ItemID == itemID {
			cp := *u
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (r *mockUnitRepo) ListEligibleByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var list []*domain.Unit
	for _, u := range r.data {
		if u.ItemID == itemID && u.Status == domain.StatusAvailable {
			cp := *u
			list = append(list, &cp)
		}
	}
	return list, nil
}

func (r *mockUnitRepo) Update(
	ctx context.Context,
	unitID uuid.UUID,
	input domain.UpdateUnitInput,
	changedBy *uuid.UUID,
) (*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.data[unitID]
	if !ok {
		return nil, domain.ErrNotFound
	}

	currentStatus := u.Status
	currentCondition := u.Condition

	newStatus := currentStatus
	statusChanged := false
	if input.Status != nil {
		if !domain.CanAdminTransition(currentStatus, *input.Status) {
			return nil, domain.ErrInvalidTransition
		}
		newStatus = *input.Status
		statusChanged = (newStatus != currentStatus)
	}

	newCondition := currentCondition
	conditionChanged := false
	if input.Condition != nil {
		newCondition = *input.Condition
		conditionChanged = (newCondition != currentCondition)
	}

	if input.Barcode != nil {
		u.Barcode = domain.NormalizeBarcode(*input.Barcode)
	}
	if input.SerialNumber != nil {
		s := strings.TrimSpace(*input.SerialNumber)
		if s == "" {
			u.SerialNumber = nil
		} else {
			u.SerialNumber = &s
		}
	}
	if input.Notes != nil {
		u.Notes = input.Notes
	}

	u.Status = newStatus
	u.Condition = newCondition
	u.UpdatedAt = time.Now()

	reasonProvided := input.Reason != nil && *input.Reason != ""
	if (conditionChanged || statusChanged || reasonProvided) && changedBy != nil {
		prevCond := currentCondition
		var logReason *string
		if statusChanged {
			if reasonProvided {
				rStr := fmt.Sprintf("[status: %s] %s", newStatus, *input.Reason)
				logReason = &rStr
			} else {
				rStr := fmt.Sprintf("[status: %s]", newStatus)
				logReason = &rStr
			}
		} else if reasonProvided {
			logReason = input.Reason
		}

		r.logs[u.ID] = append(r.logs[u.ID], &domain.ConditionLog{
			ID:                uuid.New(),
			UnitID:            u.ID,
			PreviousCondition: &prevCond,
			NewCondition:      u.Condition,
			Reason:            logReason,
			ChangedBy:         *changedBy,
			CreatedAt:         time.Now(),
		})
	}

	cp := *u
	return &cp, nil
}

func (r *mockUnitRepo) TransitionStatus(
	ctx context.Context,
	unitID uuid.UUID,
	fromStatus, toStatus domain.UnitStatus,
) (*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.data[unitID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if u.Status != fromStatus {
		return nil, domain.ErrStatusMismatch
	}
	if !domain.CanTransition(fromStatus, toStatus) {
		return nil, domain.ErrInvalidTransition
	}
	u.Status = toStatus
	cp := *u
	return &cp, nil
}

func (r *mockUnitRepo) GetConditionLogs(ctx context.Context, unitID uuid.UUID) ([]*domain.ConditionLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	logs := r.logs[unitID]
	var res []*domain.ConditionLog
	for _, l := range logs {
		cp := *l
		res = append(res, &cp)
	}
	return res, nil
}

type mockIdentityClient struct {
	members map[string]*auth.MemberInfo
}

func newMockIdentityClient() *mockIdentityClient {
	return &mockIdentityClient{members: make(map[string]*auth.MemberInfo)}
}

func (m *mockIdentityClient) setMember(orgID, userID uuid.UUID, role string) {
	key := fmt.Sprintf("%s:%s", orgID, userID)
	m.members[key] = &auth.MemberInfo{
		IsMember:       true,
		Role:           role,
		OrganizationID: orgID,
		UserID:         userID,
	}
}

func (m *mockIdentityClient) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*auth.MemberInfo, error) {
	key := fmt.Sprintf("%s:%s", orgID, userID)
	if mem, ok := m.members[key]; ok {
		return mem, nil
	}
	return nil, domain.ErrNotFound
}

func setupTestRouter() (http.Handler, *auth.JWTManager, *mockIdentityClient, *mockCategoryRepo, *mockItemRepo, *mockUnitRepo) {
	jwtSecret := "test-secret-at-least-32-bytes-long"
	jwtMgr := auth.NewJWTManager(jwtSecret)
	identity := newMockIdentityClient()

	catRepo := newMockCategoryRepo()
	itemRepo := newMockItemRepo()
	unitRepo := newMockUnitRepo()

	catSvc := service.NewCategoryService(catRepo, identity)
	itemSvc := service.NewItemService(itemRepo, catRepo, identity)
	unitSvc := service.NewUnitService(unitRepo, itemRepo, identity)
	internalSvc := service.NewInternalInventoryService(unitRepo, itemRepo)

	router := NewRouter(RouterDependencies{
		CategoryService: catSvc,
		ItemService:     itemSvc,
		UnitService:     unitSvc,
		InternalService: internalSvc,
		TokenManager:    jwtMgr,
		ReadyCheck:      func() bool { return true },
		InternalToken:   "secret-internal-token",
	})

	return router, jwtMgr, identity, catRepo, itemRepo, unitRepo
}

func TestAuthAndInternalEndpoints(t *testing.T) {
	router, jwtMgr, _, _, itemRepo, unitRepo := setupTestRouter()

	orgID := uuid.New()
	catID := uuid.New()
	itemID := uuid.New()
	unitID := uuid.New()

	itemRepo.Create(context.Background(), &domain.Item{
		ID:             itemID,
		OrganizationID: orgID,
		CategoryID:     catID,
		Name:           "Sony Camera",
		IsLendable:     true,
	})

	unitRepo.Create(context.Background(), &domain.Unit{
		ID:             unitID,
		OrganizationID: orgID,
		ItemID:         itemID,
		Barcode:        "SONY-001",
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
	}, uuid.New())

	// 1. Missing JWT token on protected endpoint
	req := httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgID.String()+"/categories", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. Invalid JWT token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgID.String()+"/categories", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for invalid token, got %d", rec.Code)
	}

	// 3. Internal eligible units without internal token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/internal/items/"+itemID.String()+"/eligible-units", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for internal endpoint without token, got %d", rec.Code)
	}

	// 4. Internal eligible units with invalid token
	req = httptest.NewRequest(http.MethodGet, "/api/v1/internal/items/"+itemID.String()+"/eligible-units", nil)
	req.Header.Set("X-Internal-Token", "wrong-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with wrong internal token, got %d", rec.Code)
	}

	// 5. Internal eligible units with valid token -> 200 OK
	req = httptest.NewRequest(http.MethodGet, "/api/v1/internal/items/"+itemID.String()+"/eligible-units", nil)
	req.Header.Set("X-Internal-Token", "secret-internal-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK with valid internal token, got %d: %s", rec.Code, rec.Body.String())
	}

	// 6. Internal status transition: available -> reserved
	statusBody := `{"from_status":"available","to_status":"reserved"}`
	req = httptest.NewRequest(http.MethodPatch, "/api/v1/internal/units/"+unitID.String()+"/status", strings.NewReader(statusBody))
	req.Header.Set("X-Internal-Token", "secret-internal-token")
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for internal status transition, got %d: %s", rec.Code, rec.Body.String())
	}

	_ = jwtMgr
}

func TestTenantAndRoleEnforcement(t *testing.T) {
	router, jwtMgr, identity, _, _, _ := setupTestRouter()

	orgA := uuid.New()
	orgB := uuid.New()
	adminA := uuid.New()
	memberA := uuid.New()

	identity.setMember(orgA, adminA, "admin")
	identity.setMember(orgA, memberA, "member")

	tokenAdminA, _ := jwtMgr.GenerateAccessToken(adminA.String(), "admin@a.com", time.Hour)
	tokenMemberA, _ := jwtMgr.GenerateAccessToken(memberA.String(), "member@a.com", time.Hour)

	// 1. MemberA tries to create category in OrgA -> 403 Forbidden
	bodyCat := `{"name":"Lenses","slug":"lenses"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgA.String()+"/categories", strings.NewReader(bodyCat))
	req.Header.Set("Authorization", "Bearer "+tokenMemberA)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for member creating category, got %d: %s", rec.Code, rec.Body.String())
	}

	// 2. AdminA creates category in OrgA -> 201 Created
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgA.String()+"/categories", strings.NewReader(bodyCat))
	req.Header.Set("Authorization", "Bearer "+tokenAdminA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for admin creating category, got %d: %s", rec.Code, rec.Body.String())
	}

	var catResp struct {
		Data domain.Category `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&catResp)
	catID := catResp.Data.ID

	// 3. MemberA reads categories in OrgA -> 200 OK
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgA.String()+"/categories", nil)
	req.Header.Set("Authorization", "Bearer "+tokenMemberA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for member listing categories, got %d", rec.Code)
	}

	// 4. AdminA accesses OrgB (cross-org) -> 403 Forbidden
	req = httptest.NewRequest(http.MethodGet, "/api/v1/organizations/"+orgB.String()+"/categories", nil)
	req.Header.Set("Authorization", "Bearer "+tokenAdminA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for cross-org access, got %d", rec.Code)
	}

	// 5. AdminA creates item in OrgA -> 201 Created
	bodyItem := fmt.Sprintf(`{"category_id":"%s","name":"Canon 50mm","is_lendable":true}`, catID)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgA.String()+"/items", strings.NewReader(bodyItem))
	req.Header.Set("Authorization", "Bearer "+tokenAdminA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for admin creating item, got %d: %s", rec.Code, rec.Body.String())
	}

	var itemResp struct {
		Data domain.Item `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&itemResp)
	itemID := itemResp.Data.ID

	// 6. AdminA creates unit in OrgA -> 201 Created
	bodyUnit := `{"barcode":"CANON-50-01","condition":"new"}`
	req = httptest.NewRequest(http.MethodPost, "/api/v1/items/"+itemID.String()+"/units", strings.NewReader(bodyUnit))
	req.Header.Set("Authorization", "Bearer "+tokenAdminA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for unit, got %d: %s", rec.Code, rec.Body.String())
	}

	var unitResp struct {
		Data domain.Unit `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&unitResp)
	unitID := unitResp.Data.ID

	// 7. MemberA gets condition logs -> 200 OK
	req = httptest.NewRequest(http.MethodGet, "/api/v1/units/"+unitID.String()+"/condition-logs", nil)
	req.Header.Set("Authorization", "Bearer "+tokenMemberA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for condition logs, got %d", rec.Code)
	}

	// 8. Outsider (user not in OrgA) tries to access item of OrgA -> 404 Not Found (enumeration prevention)
	outsiderID := uuid.New()
	tokenOutsider, _ := jwtMgr.GenerateAccessToken(outsiderID.String(), "outsider@other.com", time.Hour)
	req = httptest.NewRequest(http.MethodGet, "/api/v1/items/"+itemID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+tokenOutsider)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 Not Found for outsider accessing item, got %d", rec.Code)
	}

	// 9. Oversized payload (> 1MB) -> 400 Validation error
	largeData := strings.Repeat("A", 1024*1024+10)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/organizations/"+orgA.String()+"/categories", bytes.NewReader([]byte(largeData)))
	req.Header.Set("Authorization", "Bearer "+tokenAdminA)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request for oversized payload, got %d", rec.Code)
	}
}
