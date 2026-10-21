package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/client"
	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/service"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// In-memory repo for HTTP handler tests
type memHTTPRepo struct {
	loanRequests map[uuid.UUID]*domain.LoanRequest
	reservations map[uuid.UUID]*domain.Reservation
	loans        map[uuid.UUID]*domain.Loan
	tokens       map[string]*domain.CheckoutToken
	tokenByID    map[uuid.UUID]*domain.CheckoutToken
	idempotency  map[string]*domain.IdempotencyRecord
}

func newMemHTTPRepo() *memHTTPRepo {
	return &memHTTPRepo{
		loanRequests: make(map[uuid.UUID]*domain.LoanRequest),
		reservations: make(map[uuid.UUID]*domain.Reservation),
		loans:        make(map[uuid.UUID]*domain.Loan),
		tokens:       make(map[string]*domain.CheckoutToken),
		tokenByID:    make(map[uuid.UUID]*domain.CheckoutToken),
		idempotency:  make(map[string]*domain.IdempotencyRecord),
	}
}

func (m *memHTTPRepo) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return fn(nil)
}

func (m *memHTTPRepo) CreateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	cp := *req
	m.loanRequests[req.ID] = &cp
	return nil
}

func (m *memHTTPRepo) GetLoanRequest(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	if r, ok := m.loanRequests[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (m *memHTTPRepo) GetLoanRequestByID(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	return m.GetLoanRequest(ctx, id)
}

func (m *memHTTPRepo) ListLoanRequests(ctx context.Context, orgID uuid.UUID, requesterID *uuid.UUID, status *domain.RequestStatus, limit, offset int) ([]domain.LoanRequest, error) {
	var list []domain.LoanRequest
	for _, r := range m.loanRequests {
		if r.OrganizationID == orgID {
			list = append(list, *r)
		}
	}
	return list, nil
}

func (m *memHTTPRepo) UpdateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	cp := *req
	m.loanRequests[req.ID] = &cp
	return nil
}

func (m *memHTTPRepo) CreateReservation(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error {
	cp := *res
	m.reservations[res.ID] = &cp
	return nil
}

func (m *memHTTPRepo) GetReservation(ctx context.Context, id uuid.UUID) (*domain.Reservation, error) {
	if r, ok := m.reservations[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memHTTPRepo) GetReservationByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Reservation, error) {
	for _, r := range m.reservations {
		if r.LoanRequestID == loanRequestID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *memHTTPRepo) UpdateReservationStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.ReservationStatus) error {
	if r, ok := m.reservations[id]; ok {
		r.Status = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *memHTTPRepo) FindOverlappingUnitIDs(ctx context.Context, unitIDs []uuid.UUID, startTime, endTime time.Time) ([]uuid.UUID, error) {
	return nil, nil
}

func (m *memHTTPRepo) CreateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	cp := *loan
	m.loans[loan.ID] = &cp
	return nil
}

func (m *memHTTPRepo) GetLoan(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	if l, ok := m.loans[id]; ok {
		cp := *l
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (m *memHTTPRepo) GetLoanByID(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	return m.GetLoan(ctx, id)
}

func (m *memHTTPRepo) GetLoanByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Loan, error) {
	for _, l := range m.loans {
		if l.LoanRequestID == loanRequestID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *memHTTPRepo) ListLoans(ctx context.Context, orgID uuid.UUID, borrowerUserID *uuid.UUID, status *domain.LoanStatus, limit, offset int) ([]domain.Loan, error) {
	var list []domain.Loan
	for _, l := range m.loans {
		if l.OrganizationID != orgID {
			continue
		}
		if borrowerUserID != nil && l.BorrowerUserID != *borrowerUserID {
			continue
		}
		if status != nil && l.Status != *status {
			continue
		}
		list = append(list, *l)
	}
	return list, nil
}

func (m *memHTTPRepo) UpdateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	cp := *loan
	m.loans[loan.ID] = &cp
	return nil
}

func (m *memHTTPRepo) ListOverdueLoans(ctx context.Context, orgID uuid.UUID) ([]domain.Loan, error) {
	var list []domain.Loan
	for _, l := range m.loans {
		if l.OrganizationID == orgID && l.Status == domain.LoanStatusOverdue {
			list = append(list, *l)
		}
	}
	return list, nil
}

func (m *memHTTPRepo) FindOverdueCheckedOutLoans(ctx context.Context, now time.Time) ([]domain.Loan, error) {
	return nil, nil
}
func (m *memHTTPRepo) TransitionLoanToOverdue(ctx context.Context, loanID uuid.UUID, now time.Time) (bool, error) {
	if l, ok := m.loans[loanID]; ok && l.Status == domain.LoanStatusCheckedOut {
		l.Status = domain.LoanStatusOverdue
		l.UpdatedAt = now
		return true, nil
	}
	return false, nil
}

func (m *memHTTPRepo) CreateCheckoutToken(ctx context.Context, tx pgx.Tx, token *domain.CheckoutToken) error {
	cp := *token
	m.tokens[token.TokenHash] = &cp
	m.tokenByID[token.ID] = &cp
	return nil
}

func (m *memHTTPRepo) GetCheckoutTokenByHash(ctx context.Context, tokenHash string) (*domain.CheckoutToken, error) {
	if t, ok := m.tokens[tokenHash]; ok {
		cp := *t
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memHTTPRepo) MarkCheckoutTokenUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID, usedAt time.Time) error {
	if t, ok := m.tokenByID[id]; ok {
		t.UsedAt = &usedAt
		return nil
	}
	return domain.ErrNotFound
}

func (m *memHTTPRepo) CreateStatusHistory(ctx context.Context, tx pgx.Tx, history *domain.StatusHistory) error {
	return nil
}

func (m *memHTTPRepo) GetStatusHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.StatusHistory, error) {
	return nil, nil
}

func (m *memHTTPRepo) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event *domain.OutboxEvent) error {
	return nil
}

func (m *memHTTPRepo) GetPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	return nil, nil
}
func (m *memHTTPRepo) ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	return nil, nil
}

func (m *memHTTPRepo) MarkOutboxEventPublished(ctx context.Context, id int64) error {
	return nil
}

func (m *memHTTPRepo) MarkOutboxEventFailed(ctx context.Context, id int64, errStr string) error {
	return nil
}

func (m *memHTTPRepo) GetIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.IdempotencyRecord, error) {
	k := orgID.String() + ":" + key
	if rec, ok := m.idempotency[k]; ok {
		cp := *rec
		return &cp, nil
	}
	return nil, nil
}

func (m *memHTTPRepo) SaveIdempotencyKey(ctx context.Context, record *domain.IdempotencyRecord) error {
	k := record.OrganizationID.String() + ":" + record.Key
	cp := *record
	m.idempotency[k] = &cp
	return nil
}

type mockIdentClient struct {
	members map[string]*auth.MemberInfo
}

func (m *mockIdentClient) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*auth.MemberInfo, error) {
	k := orgID.String() + ":" + userID.String()
	if mem, ok := m.members[k]; ok {
		return mem, nil
	}
	return &auth.MemberInfo{IsMember: false}, nil
}

type mockInvClient struct {
	units map[uuid.UUID]*client.InventoryUnit
}

func (m *mockInvClient) GetEligibleUnits(ctx context.Context, itemID uuid.UUID) ([]client.InventoryUnit, error) {
	var list []client.InventoryUnit
	for _, u := range m.units {
		if u.ItemID == itemID && u.Status == "available" {
			list = append(list, *u)
		}
	}
	return list, nil
}

func (m *mockInvClient) TransitionUnitStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus string) (*client.InventoryUnit, error) {
	u, ok := m.units[unitID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u.Status = toStatus
	cp := *u
	return &cp, nil
}

func setupTestServer() (http.Handler, *auth.JWTManager, uuid.UUID, uuid.UUID, string) {
	jwtSecret := "test-secret-at-least-32-bytes-long-for-hs256!"
	jwtMgr := auth.NewJWTManager(jwtSecret)

	orgID := uuid.New()
	adminID := uuid.New()
	internalToken := "test-internal-token"

	repo := newMemHTTPRepo()
	ident := &mockIdentClient{
		members: map[string]*auth.MemberInfo{
			orgID.String() + ":" + adminID.String(): {
				IsMember:       true,
				Role:           "admin",
				OrganizationID: orgID,
				UserID:         adminID,
			},
		},
	}
	inv := &mockInvClient{
		units: make(map[uuid.UUID]*client.InventoryUnit),
	}

	lendingSvc := service.NewLendingService(repo, ident, inv)

	router := NewRouter(RouterDependencies{
		LendingService: lendingSvc,
		Repository:     nil, // Not needed for simple routing test or tested via handlers
		TokenManager:   jwtMgr,
		ReadyCheck:     func() bool { return true },
		InternalToken:  internalToken,
	})

	handler := RequestIDs(router)
	return handler, jwtMgr, orgID, adminID, internalToken
}

func TestHealthEndpoints(t *testing.T) {
	handler, _, _, _, _ := setupTestServer()

	// Live
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	// Ready
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestInternalSchedulerEndpoint(t *testing.T) {
	handler, _, _, _, internalToken := setupTestServer()

	// Missing internal token
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/scheduler/process-overdue", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}

	// Valid internal token
	req = httptest.NewRequest(http.MethodPost, "/api/v1/internal/scheduler/process-overdue", nil)
	req.Header.Set("X-Internal-Token", internalToken)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestProtectedEndpointWithoutAuth(t *testing.T) {
	handler, _, orgID, _, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-requests?organization_id="+orgID.String(), nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", rr.Code)
	}
}

func TestProtectedEndpointWithAuth(t *testing.T) {
	handler, jwtMgr, orgID, adminID, _ := setupTestServer()

	token, err := jwtMgr.GenerateAccessToken(adminID.String(), "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/loan-requests?organization_id="+orgID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data []domain.LoanRequest `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
}

func TestCreateLoanRequestValidation(t *testing.T) {
	handler, jwtMgr, orgID, adminID, _ := setupTestServer()

	token, _ := jwtMgr.GenerateAccessToken(adminID.String(), "admin@example.com", time.Hour)

	// Missing required fields
	payload := map[string]any{
		"organization_id": orgID.String(),
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
	}
}

type fullTestFixture struct {
	handler        http.Handler
	jwtMgr         *auth.JWTManager
	repo           *memHTTPRepo
	ident          *mockIdentClient
	inv            *mockInvClient
	orgID          uuid.UUID
	adminID        uuid.UUID
	memberID       uuid.UUID
	otherMemberID  uuid.UUID
	nonMemberID    uuid.UUID
	itemID         uuid.UUID
	unitID         uuid.UUID
	adminToken     string
	memberToken    string
	otherToken     string
	nonMemberToken string
}

func setupFullTestServer(t *testing.T) *fullTestFixture {
	jwtSecret := "test-secret-at-least-32-bytes-long-for-hs256!"
	jwtMgr := auth.NewJWTManager(jwtSecret)

	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	otherMemberID := uuid.New()
	nonMemberID := uuid.New()
	itemID := uuid.New()
	unitID := uuid.New()
	internalToken := "test-internal-token"

	repo := newMemHTTPRepo()
	ident := &mockIdentClient{
		members: map[string]*auth.MemberInfo{
			orgID.String() + ":" + adminID.String(): {
				IsMember:       true,
				Role:           "admin",
				OrganizationID: orgID,
				UserID:         adminID,
			},
			orgID.String() + ":" + memberID.String(): {
				IsMember:       true,
				Role:           "member",
				OrganizationID: orgID,
				UserID:         memberID,
			},
			orgID.String() + ":" + otherMemberID.String(): {
				IsMember:       true,
				Role:           "member",
				OrganizationID: orgID,
				UserID:         otherMemberID,
			},
		},
	}
	inv := &mockInvClient{
		units: map[uuid.UUID]*client.InventoryUnit{
			unitID: {
				ID:     unitID,
				ItemID: itemID,
				Status: "available",
			},
		},
	}

	lendingSvc := service.NewLendingService(repo, ident, inv)

	router := NewRouter(RouterDependencies{
		LendingService: lendingSvc,
		Repository:     repo,
		TokenManager:   jwtMgr,
		ReadyCheck:     func() bool { return true },
		InternalToken:  internalToken,
	})

	handler := RequestIDs(router)

	adminToken, err := jwtMgr.GenerateAccessToken(adminID.String(), "admin@example.com", time.Hour)
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	memberToken, err := jwtMgr.GenerateAccessToken(memberID.String(), "member@example.com", time.Hour)
	if err != nil {
		t.Fatalf("generate member token: %v", err)
	}
	otherToken, err := jwtMgr.GenerateAccessToken(otherMemberID.String(), "other@example.com", time.Hour)
	if err != nil {
		t.Fatalf("generate other token: %v", err)
	}
	nonMemberToken, err := jwtMgr.GenerateAccessToken(nonMemberID.String(), "nonmember@example.com", time.Hour)
	if err != nil {
		t.Fatalf("generate nonmember token: %v", err)
	}

	return &fullTestFixture{
		handler:        handler,
		jwtMgr:         jwtMgr,
		repo:           repo,
		ident:          ident,
		inv:            inv,
		orgID:          orgID,
		adminID:        adminID,
		memberID:       memberID,
		otherMemberID:  otherMemberID,
		nonMemberID:    nonMemberID,
		itemID:         itemID,
		unitID:         unitID,
		adminToken:     adminToken,
		memberToken:    memberToken,
		otherToken:     otherToken,
		nonMemberToken: nonMemberToken,
	}
}

func TestHTTPFullLifecycle_Create_Approve_Checkout_Return(t *testing.T) {
	f := setupFullTestServer(t)

	// 1. Create Loan Request as Member
	now := time.Now().UTC()
	createPayload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Field research",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&createResp); err != nil {
		t.Fatalf("decode create resp: %v", err)
	}
	reqID := createResp.Data.ID
	if reqID == uuid.Nil {
		t.Fatal("expected non-nil request id")
	}

	// 2. Get Loan Request as Member
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loan-requests/"+reqID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get loan request, got %d: %s", rr.Code, rr.Body.String())
	}

	var getReqResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&getReqResp); err != nil {
		t.Fatalf("decode get request resp: %v", err)
	}
	if getReqResp.Data.Status != domain.RequestStatusPending {
		t.Fatalf("expected pending status, got %s", getReqResp.Data.Status)
	}

	// 3. Approve Loan Request as Admin
	approvePayload := map[string]any{
		"review_notes": "Approved for field research",
	}
	body, _ = json.Marshal(approvePayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/approve", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for approve, got %d: %s", rr.Code, rr.Body.String())
	}

	var approveResp struct {
		Data struct {
			LoanRequest domain.LoanRequest `json:"loan_request"`
			Loan        domain.Loan        `json:"loan"`
			PlainToken  string             `json:"checkout_token"`
		} `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&approveResp); err != nil {
		t.Fatalf("decode approve resp: %v", err)
	}
	token := approveResp.Data.PlainToken
	loanID := approveResp.Data.Loan.ID
	if token == "" || loanID == uuid.Nil {
		t.Fatalf("expected valid token and loan id, got token=%q loanID=%s", token, loanID)
	}

	// 4. Get Loan as Member
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+loanID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get loan, got %d: %s", rr.Code, rr.Body.String())
	}

	var getLoanResp struct {
		Data domain.Loan `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&getLoanResp); err != nil {
		t.Fatalf("decode get loan resp: %v", err)
	}
	if getLoanResp.Data.Status != domain.LoanStatusPendingCheckout {
		t.Fatalf("expected pending_checkout status, got %s", getLoanResp.Data.Status)
	}

	// 5. Checkout Loan (admin verifies token and checks out)
	checkoutPayload := map[string]any{
		"token": token,
	}
	body, _ = json.Marshal(checkoutPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+loanID.String()+"/check-out", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for checkout, got %d: %s", rr.Code, rr.Body.String())
	}

	var checkoutResp struct {
		Data domain.Loan `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&checkoutResp); err != nil {
		t.Fatalf("decode checkout resp: %v", err)
	}
	if checkoutResp.Data.Status != domain.LoanStatusCheckedOut {
		t.Fatalf("expected checked_out status, got %s", checkoutResp.Data.Status)
	}

	// 6. Return Loan (admin processes return)
	returnPayload := map[string]any{
		"return_condition": "good",
		"return_notes":     "Equipment returned in pristine state",
	}
	body, _ = json.Marshal(returnPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loans/"+loanID.String()+"/return", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for return, got %d: %s", rr.Code, rr.Body.String())
	}

	var returnResp struct {
		Data domain.Loan `json:"data"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&returnResp); err != nil {
		t.Fatalf("decode return resp: %v", err)
	}
	if returnResp.Data.Status != domain.LoanStatusReturned {
		t.Fatalf("expected returned status, got %s", returnResp.Data.Status)
	}

	// 7. Verify Loan is returned
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans/"+loanID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for final get loan, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTPLifecycle_Reject(t *testing.T) {
	f := setupFullTestServer(t)

	// Create request
	now := time.Now().UTC()
	createPayload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Testing reject",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&createResp)
	reqID := createResp.Data.ID

	// Reject as Admin
	rejectPayload := map[string]any{
		"review_notes": "Unit needs calibration",
	}
	body, _ = json.Marshal(rejectPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/reject", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for reject, got %d: %s", rr.Code, rr.Body.String())
	}

	var rejectResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&rejectResp)
	if rejectResp.Data.Status != domain.RequestStatusRejected {
		t.Fatalf("expected rejected status, got %s", rejectResp.Data.Status)
	}

	// Verify via GET
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loan-requests/"+reqID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for get loan request, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHTTPLifecycle_Cancel(t *testing.T) {
	f := setupFullTestServer(t)

	// Create request
	now := time.Now().UTC()
	createPayload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Testing cancel",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&createResp)
	reqID := createResp.Data.ID

	// Cancel as Member
	cancelPayload := map[string]any{
		"reason": "Plans changed",
	}
	body, _ = json.Marshal(cancelPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/cancel", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cancel, got %d: %s", rr.Code, rr.Body.String())
	}

	var cancelResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&cancelResp)
	if cancelResp.Data.Status != domain.RequestStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelResp.Data.Status)
	}
}

func TestHTTP_NonMemberForbidden(t *testing.T) {
	f := setupFullTestServer(t)

	// Create request as valid member
	now := time.Now().UTC()
	createPayload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Testing auth bounds",
	}
	body, _ := json.Marshal(createPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp struct {
		Data domain.LoanRequest `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&createResp)
	reqID := createResp.Data.ID

	// Non-member calls GetLoanRequest -> 403
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loan-requests/"+reqID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.nonMemberToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-member get request, got %d", rr.Code)
	}

	// Non-member calls Approve -> 403
	approveBody, _ := json.Marshal(map[string]any{"review_notes": "hack"})
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/approve", bytes.NewReader(approveBody))
	req.Header.Set("Authorization", "Bearer "+f.nonMemberToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-member approve, got %d", rr.Code)
	}

	// Non-member calls Reject -> 403
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/reject", bytes.NewReader(approveBody))
	req.Header.Set("Authorization", "Bearer "+f.nonMemberToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-member reject, got %d", rr.Code)
	}

	// Non-member calls Cancel -> 403
	req = httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests/"+reqID.String()+"/cancel", bytes.NewReader(approveBody))
	req.Header.Set("Authorization", "Bearer "+f.nonMemberToken)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for non-member cancel, got %d", rr.Code)
	}
}

func TestHTTP_ListLoans_MemberIsolation(t *testing.T) {
	f := setupFullTestServer(t)
	now := time.Now().UTC()

	// Create loan 1 for Member
	loan1 := domain.Loan{
		ID:                 uuid.New(),
		OrganizationID:     f.orgID,
		LoanRequestID:      uuid.New(),
		ReservationID:      uuid.New(),
		UnitID:             f.unitID,
		BorrowerUserID:     f.memberID,
		Status:             domain.LoanStatusCheckedOut,
		CheckoutTime:       &now,
		ExpectedReturnTime: now.Add(24 * time.Hour),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_ = f.repo.CreateLoan(context.Background(), nil, &loan1)

	// Create loan 2 for Other Member
	unit2ID := uuid.New()
	loan2 := domain.Loan{
		ID:                 uuid.New(),
		OrganizationID:     f.orgID,
		LoanRequestID:      uuid.New(),
		ReservationID:      uuid.New(),
		UnitID:             unit2ID,
		BorrowerUserID:     f.otherMemberID,
		Status:             domain.LoanStatusCheckedOut,
		CheckoutTime:       &now,
		ExpectedReturnTime: now.Add(24 * time.Hour),
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	_ = f.repo.CreateLoan(context.Background(), nil, &loan2)

	// Member list loans -> should only see loan 1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/loans?organization_id="+f.orgID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for member list loans, got %d: %s", rr.Code, rr.Body.String())
	}
	var memberResp struct {
		Data []domain.Loan `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&memberResp)
	if len(memberResp.Data) != 1 {
		t.Fatalf("expected 1 loan for member, got %d", len(memberResp.Data))
	}
	if memberResp.Data[0].ID != loan1.ID {
		t.Fatalf("expected loan %s, got %s", loan1.ID, memberResp.Data[0].ID)
	}

	// Other member list loans -> should only see loan 2
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans?organization_id="+f.orgID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.otherToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for other member list loans, got %d: %s", rr.Code, rr.Body.String())
	}
	var otherResp struct {
		Data []domain.Loan `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&otherResp)
	if len(otherResp.Data) != 1 {
		t.Fatalf("expected 1 loan for other member, got %d", len(otherResp.Data))
	}
	if otherResp.Data[0].ID != loan2.ID {
		t.Fatalf("expected loan %s, got %s", loan2.ID, otherResp.Data[0].ID)
	}

	// Admin list loans -> should see both loans
	req = httptest.NewRequest(http.MethodGet, "/api/v1/loans?organization_id="+f.orgID.String(), nil)
	req.Header.Set("Authorization", "Bearer "+f.adminToken)
	rr = httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin list loans, got %d: %s", rr.Code, rr.Body.String())
	}
	var adminResp struct {
		Data []domain.Loan `json:"data"`
	}
	_ = json.NewDecoder(rr.Body).Decode(&adminResp)
	if len(adminResp.Data) != 2 {
		t.Fatalf("expected 2 loans for admin, got %d", len(adminResp.Data))
	}
}

func TestHTTP_CreateLoanRequest_MultiUnitValidation(t *testing.T) {
	f := setupFullTestServer(t)
	now := time.Now().UTC()

	payload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        2,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Multi unit test",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request, got %d: %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "quantity must be 1 (multi-unit loans currently not supported)") {
		t.Fatalf("expected error message to contain multi-unit notice, got: %s", rr.Body.String())
	}
}
func TestHTTP_IdempotencyKey_BodyOrgExtraction(t *testing.T) {
	f := setupFullTestServer(t)
	now := time.Now().UTC()

	payload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Idempotency test via body org extraction",
	}
	body, _ := json.Marshal(payload)

	idempKey := "test-idemp-" + uuid.New().String()

	// First request: no org header/query; org is inside JSON body
	req := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.memberToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempKey)
	rr := httptest.NewRecorder()
	f.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on first request, got %d: %s", rr.Code, rr.Body.String())
	}
	firstBody := rr.Body.String()

	// Verify in repo that the record was stored under f.orgID, NOT uuid.Nil
	recKey := f.orgID.String() + ":" + idempKey
	rec, ok := f.repo.idempotency[recKey]
	if !ok || rec == nil {
		t.Fatalf("expected idempotency record stored under key %s, but not found", recKey)
	}
	if rec.OrganizationID != f.orgID {
		t.Fatalf("expected record organization_id to be %s, got %s", f.orgID, rec.OrganizationID)
	}

	// Re-send identical request: must return cached response
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(body))
	req2.Header.Set("Authorization", "Bearer "+f.memberToken)
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Idempotency-Key", idempKey)
	rr2 := httptest.NewRecorder()
	f.handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created on replay, got %d", rr2.Code)
	}
	if rr2.Body.String() != firstBody {
		t.Fatalf("expected replayed response body to match original")
	}

	// Conflicting request with same idempotency key but different purpose
	conflictPayload := map[string]any{
		"organization_id": f.orgID.String(),
		"item_id":         f.itemID.String(),
		"quantity":        1,
		"start_time":      now.Add(time.Hour).Format(time.RFC3339Nano),
		"end_time":        now.Add(24 * time.Hour).Format(time.RFC3339Nano),
		"purpose":         "Different conflicting purpose",
	}
	conflictBody, _ := json.Marshal(conflictPayload)
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/loan-requests", bytes.NewReader(conflictBody))
	req3.Header.Set("Authorization", "Bearer "+f.memberToken)
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Idempotency-Key", idempKey)
	rr3 := httptest.NewRecorder()
	f.handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for conflicting body, got %d: %s", rr3.Code, rr3.Body.String())
	}
}
