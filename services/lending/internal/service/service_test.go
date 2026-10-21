package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/client"
	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/event"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// ---------------------------------------------------------------------------
// Mock Clients and In-Memory Repository
// ---------------------------------------------------------------------------

type mockIdentityClient struct {
	members map[string]*auth.MemberInfo
}

func newMockIdentityClient() *mockIdentityClient {
	return &mockIdentityClient{members: make(map[string]*auth.MemberInfo)}
}

func (m *mockIdentityClient) addMember(orgID, userID uuid.UUID, role string) {
	key := orgID.String() + ":" + userID.String()
	m.members[key] = &auth.MemberInfo{
		IsMember:       true,
		Role:           role,
		OrganizationID: orgID,
		UserID:         userID,
	}
}

func (m *mockIdentityClient) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*auth.MemberInfo, error) {
	key := orgID.String() + ":" + userID.String()
	mem, ok := m.members[key]
	if !ok {
		return &auth.MemberInfo{IsMember: false}, nil
	}
	return mem, nil
}

type mockInventoryClient struct {
	mu    sync.Mutex
	units map[uuid.UUID]*client.InventoryUnit
}

func newMockInventoryClient() *mockInventoryClient {
	return &mockInventoryClient{units: make(map[uuid.UUID]*client.InventoryUnit)}
}

func (m *mockInventoryClient) addUnit(unit client.InventoryUnit) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := unit
	m.units[unit.ID] = &cp
}

func (m *mockInventoryClient) GetEligibleUnits(ctx context.Context, itemID uuid.UUID) ([]client.InventoryUnit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []client.InventoryUnit
	for _, u := range m.units {
		if u.ItemID == itemID && u.Status == "available" {
			list = append(list, *u)
		}
	}
	return list, nil
}

func (m *mockInventoryClient) TransitionUnitStatus(ctx context.Context, unitID uuid.UUID, fromStatus, toStatus string) (*client.InventoryUnit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.units[unitID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if u.Status != fromStatus {
		return nil, domain.ErrConflict
	}
	u.Status = toStatus
	cp := *u
	return &cp, nil
}

type memRepository struct {
	mu           sync.Mutex
	loanRequests map[uuid.UUID]*domain.LoanRequest
	reservations map[uuid.UUID]*domain.Reservation
	loans        map[uuid.UUID]*domain.Loan
	tokens       map[string]*domain.CheckoutToken
	tokenByID    map[uuid.UUID]*domain.CheckoutToken
	history      []domain.StatusHistory
	outbox       []domain.OutboxEvent
	idempotency  map[string]*domain.IdempotencyRecord
}

func newMemRepository() *memRepository {
	return &memRepository{
		loanRequests: make(map[uuid.UUID]*domain.LoanRequest),
		reservations: make(map[uuid.UUID]*domain.Reservation),
		loans:        make(map[uuid.UUID]*domain.Loan),
		tokens:       make(map[string]*domain.CheckoutToken),
		tokenByID:    make(map[uuid.UUID]*domain.CheckoutToken),
		idempotency:  make(map[string]*domain.IdempotencyRecord),
	}
}

func (m *memRepository) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(nil)
}

func (m *memRepository) CreateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	cp := *req
	m.loanRequests[req.ID] = &cp
	return nil
}

func (m *memRepository) GetLoanRequest(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	if req, ok := m.loanRequests[id]; ok {
		cp := *req
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (m *memRepository) GetLoanRequestByID(ctx context.Context, id uuid.UUID) (*domain.LoanRequest, error) {
	return m.GetLoanRequest(ctx, id)
}

func (m *memRepository) ListLoanRequests(ctx context.Context, orgID uuid.UUID, requesterID *uuid.UUID, status *domain.RequestStatus, limit, offset int) ([]domain.LoanRequest, error) {
	var list []domain.LoanRequest
	for _, req := range m.loanRequests {
		if req.OrganizationID != orgID {
			continue
		}
		if requesterID != nil && req.RequesterUserID != *requesterID {
			continue
		}
		if status != nil && req.Status != *status {
			continue
		}
		list = append(list, *req)
	}
	return list, nil
}

func (m *memRepository) UpdateLoanRequest(ctx context.Context, tx pgx.Tx, req *domain.LoanRequest) error {
	cp := *req
	m.loanRequests[req.ID] = &cp
	return nil
}

func (m *memRepository) CreateReservation(ctx context.Context, tx pgx.Tx, res *domain.Reservation) error {
	for _, existing := range m.reservations {
		if existing.UnitID == res.UnitID && existing.Status == domain.ReservationStatusActive {
			if domain.DoReservationsOverlap(existing.StartTime, existing.EndTime, res.StartTime, res.EndTime) {
				return domain.ErrReservationConflict
			}
		}
	}
	cp := *res
	m.reservations[res.ID] = &cp
	return nil
}

func (m *memRepository) GetReservation(ctx context.Context, id uuid.UUID) (*domain.Reservation, error) {
	if res, ok := m.reservations[id]; ok {
		cp := *res
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memRepository) GetReservationByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Reservation, error) {
	for _, r := range m.reservations {
		if r.LoanRequestID == loanRequestID {
			cp := *r
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *memRepository) UpdateReservationStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.ReservationStatus) error {
	if res, ok := m.reservations[id]; ok {
		res.Status = status
		return nil
	}
	return domain.ErrNotFound
}

func (m *memRepository) FindOverlappingUnitIDs(ctx context.Context, unitIDs []uuid.UUID, startTime, endTime time.Time) ([]uuid.UUID, error) {
	var overlapping []uuid.UUID
	targetMap := make(map[uuid.UUID]bool)
	for _, id := range unitIDs {
		targetMap[id] = true
	}

	for _, r := range m.reservations {
		if targetMap[r.UnitID] && r.Status == domain.ReservationStatusActive {
			if domain.DoReservationsOverlap(r.StartTime, r.EndTime, startTime, endTime) {
				overlapping = append(overlapping, r.UnitID)
			}
		}
	}
	return overlapping, nil
}

func (m *memRepository) CreateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	cp := *loan
	m.loans[loan.ID] = &cp
	return nil
}

func (m *memRepository) GetLoan(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	if loan, ok := m.loans[id]; ok {
		cp := *loan
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}
func (m *memRepository) GetLoanByID(ctx context.Context, id uuid.UUID) (*domain.Loan, error) {
	return m.GetLoan(ctx, id)
}

func (m *memRepository) GetLoanByLoanRequestID(ctx context.Context, loanRequestID uuid.UUID) (*domain.Loan, error) {
	for _, l := range m.loans {
		if l.LoanRequestID == loanRequestID {
			cp := *l
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *memRepository) ListLoans(ctx context.Context, orgID uuid.UUID, borrowerUserID *uuid.UUID, status *domain.LoanStatus, limit, offset int) ([]domain.Loan, error) {
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

func (m *memRepository) UpdateLoan(ctx context.Context, tx pgx.Tx, loan *domain.Loan) error {
	cp := *loan
	m.loans[loan.ID] = &cp
	return nil
}

func (m *memRepository) ListOverdueLoans(ctx context.Context, orgID uuid.UUID) ([]domain.Loan, error) {
	var list []domain.Loan
	for _, l := range m.loans {
		if l.OrganizationID == orgID && l.Status == domain.LoanStatusOverdue {
			list = append(list, *l)
		}
	}
	return list, nil
}

func (m *memRepository) FindOverdueCheckedOutLoans(ctx context.Context, now time.Time) ([]domain.Loan, error) {
	var list []domain.Loan
	for _, l := range m.loans {
		if l.Status == domain.LoanStatusCheckedOut && l.ExpectedReturnTime.Before(now) {
			list = append(list, *l)
		}
	}
	return list, nil
}
func (m *memRepository) TransitionLoanToOverdue(ctx context.Context, loanID uuid.UUID, now time.Time) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	loan, ok := m.loans[loanID]
	if !ok || loan.Status != domain.LoanStatusCheckedOut {
		return false, nil
	}
	loan.Status = domain.LoanStatusOverdue
	loan.UpdatedAt = now
	corrID := uuid.New().String()
	m.history = append(m.history, domain.StatusHistory{
		ID:              uuid.New(),
		EntityType:      "loan",
		EntityID:        loan.ID,
		FromStatus:      string(domain.LoanStatusCheckedOut),
		ToStatus:        string(domain.LoanStatusOverdue),
		ChangedByUserID: uuid.Nil,
		CorrelationID:   corrID,
		CreatedAt:       now,
	})
	ev := event.BuildOutboxEvent(
		event.EventLoanOverdue,
		loan.ID.String(),
		loan.OrganizationID,
		corrID,
		loan.ID.String(),
		map[string]any{
			"loan_id":              loan.ID.String(),
			"unit_id":              loan.UnitID.String(),
			"borrower_user_id":     loan.BorrowerUserID.String(),
			"expected_return_time": loan.ExpectedReturnTime.Format(time.RFC3339Nano),
		},
	)
	m.outbox = append(m.outbox, ev)
	return true, nil
}

func (m *memRepository) CreateCheckoutToken(ctx context.Context, tx pgx.Tx, token *domain.CheckoutToken) error {
	cp := *token
	m.tokens[token.TokenHash] = &cp
	m.tokenByID[token.ID] = &cp
	return nil
}

func (m *memRepository) GetCheckoutTokenByHash(ctx context.Context, tokenHash string) (*domain.CheckoutToken, error) {
	if tok, ok := m.tokens[tokenHash]; ok {
		cp := *tok
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (m *memRepository) MarkCheckoutTokenUsed(ctx context.Context, tx pgx.Tx, id uuid.UUID, usedAt time.Time) error {
	tok, ok := m.tokenByID[id]
	if !ok {
		return domain.ErrNotFound
	}
	if tok.UsedAt != nil {
		return domain.ErrTokenAlreadyUsed
	}
	tok.UsedAt = &usedAt
	return nil
}

func (m *memRepository) CreateStatusHistory(ctx context.Context, tx pgx.Tx, history *domain.StatusHistory) error {
	m.history = append(m.history, *history)
	return nil
}

func (m *memRepository) GetStatusHistory(ctx context.Context, entityType string, entityID uuid.UUID) ([]domain.StatusHistory, error) {
	var list []domain.StatusHistory
	for _, h := range m.history {
		if h.EntityType == entityType && h.EntityID == entityID {
			list = append(list, h)
		}
	}
	return list, nil
}

func (m *memRepository) CreateOutboxEvent(ctx context.Context, tx pgx.Tx, event *domain.OutboxEvent) error {
	m.outbox = append(m.outbox, *event)
	return nil
}

func (m *memRepository) ClaimPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	var list []domain.OutboxEvent
	count := 0
	for i := range m.outbox {
		if m.outbox[i].Status == "pending" {
			m.outbox[i].Status = "in_flight"
			list = append(list, m.outbox[i])
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}
	return list, nil
}

func (m *memRepository) GetPendingOutboxEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	return m.ClaimPendingOutboxEvents(ctx, limit)
}

func (m *memRepository) MarkOutboxEventPublished(ctx context.Context, id int64) error {
	for i := range m.outbox {
		if m.outbox[i].ID == id {
			m.outbox[i].Status = "published"
		}
	}
	return nil
}

func (m *memRepository) MarkOutboxEventFailed(ctx context.Context, id int64, errStr string) error {
	for i := range m.outbox {
		if m.outbox[i].ID == id {
			m.outbox[i].RetryCount++
			m.outbox[i].LastError = &errStr
		}
	}
	return nil
}

func (m *memRepository) GetIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.IdempotencyRecord, error) {
	k := orgID.String() + ":" + key
	if rec, ok := m.idempotency[k]; ok {
		cp := *rec
		return &cp, nil
	}
	return nil, nil
}

func (m *memRepository) SaveIdempotencyKey(ctx context.Context, record *domain.IdempotencyRecord) error {
	k := record.OrganizationID.String() + ":" + record.Key
	cp := *record
	m.idempotency[k] = &cp
	return nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func setupTestService() (*LendingService, *mockIdentityClient, *mockInventoryClient, *memRepository, uuid.UUID, uuid.UUID, uuid.UUID) {
	repo := newMemRepository()
	ident := newMockIdentityClient()
	inv := newMockInventoryClient()

	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	ident.addMember(orgID, adminID, "admin")
	ident.addMember(orgID, memberID, "member")

	svc := NewLendingService(repo, ident, inv)
	return svc, ident, inv, repo, orgID, adminID, memberID
}

func TestAvailabilityCalculation(t *testing.T) {
	svc, _, inv, repo, orgID, _, memberID := setupTestService()
	itemID := uuid.New()
	unit1 := uuid.New()
	unit2 := uuid.New()

	inv.addUnit(client.InventoryUnit{ID: unit1, ItemID: itemID, Status: "available"})
	inv.addUnit(client.InventoryUnit{ID: unit2, ItemID: itemID, Status: "available"})

	now := time.Now().UTC()
	start := now.Add(2 * time.Hour)
	end := now.Add(4 * time.Hour)

	// Initial: 2 units available
	avail, err := svc.GetItemAvailability(context.Background(), orgID, memberID, itemID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avail.TotalUnits != 2 || avail.AvailableUnits != 2 || !avail.IsAvailable {
		t.Fatalf("expected 2 available, got %+v", avail)
	}

	// Add overlapping reservation for unit1
	_ = repo.CreateReservation(context.Background(), nil, &domain.Reservation{
		ID:             uuid.New(),
		OrganizationID: orgID,
		LoanRequestID:  uuid.New(),
		UnitID:         unit1,
		StartTime:      start.Add(-time.Hour),
		EndTime:        start.Add(time.Hour), // overlaps start
		Status:         domain.ReservationStatusActive,
	})

	// Now only 1 unit available
	avail, err = svc.GetItemAvailability(context.Background(), orgID, memberID, itemID, start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if avail.TotalUnits != 2 || avail.AvailableUnits != 1 || !avail.IsAvailable {
		t.Fatalf("expected 1 available, got %+v", avail)
	}
}

func TestCreateLoanRequestWorkflow(t *testing.T) {
	svc, _, inv, repo, orgID, _, memberID := setupTestService()
	itemID := uuid.New()
	unitID := uuid.New()
	inv.addUnit(client.InventoryUnit{ID: unitID, ItemID: itemID, Status: "available"})

	now := time.Now().UTC()
	input := domain.CreateLoanRequestInput{
		ItemID:    itemID,
		Quantity:  1,
		StartTime: now.Add(2 * time.Hour),
		EndTime:   now.Add(6 * time.Hour),
		Purpose:   "Field photography test",
	}

	req, err := svc.CreateLoanRequest(context.Background(), orgID, memberID, input, "corr-1")
	if err != nil {
		t.Fatalf("failed creating loan request: %v", err)
	}

	if req.Status != domain.RequestStatusPending {
		t.Fatalf("expected status pending, got %s", req.Status)
	}
	if len(repo.outbox) != 1 || repo.outbox[0].EventType != "loan.requested" {
		t.Fatalf("expected 1 loan.requested outbox event, got %d", len(repo.outbox))
	}
}

func TestApproveAndCheckoutWorkflow(t *testing.T) {
	svc, _, inv, repo, orgID, adminID, memberID := setupTestService()
	itemID := uuid.New()
	unitID := uuid.New()
	inv.addUnit(client.InventoryUnit{ID: unitID, ItemID: itemID, Status: "available"})

	now := time.Now().UTC()
	input := domain.CreateLoanRequestInput{
		ItemID:    itemID,
		Quantity:  1,
		StartTime: now.Add(time.Hour),
		EndTime:   now.Add(5 * time.Hour),
		Purpose:   "Drone inspection",
	}

	req, err := svc.CreateLoanRequest(context.Background(), orgID, memberID, input, "corr-1")
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	// Member cannot approve
	_, err = svc.ApproveLoanRequest(context.Background(), orgID, memberID, "member", req.ID, nil, "corr-2")
	if !errors.Is(err, domain.ErrUnauthorizedAction) {
		t.Fatalf("expected ErrUnauthorizedAction, got %v", err)
	}

	// Admin approves
	notes := "Approved for research"
	result, err := svc.ApproveLoanRequest(context.Background(), orgID, adminID, "admin", req.ID, &notes, "corr-3")
	if err != nil {
		t.Fatalf("admin approval failed: %v", err)
	}

	if result.LoanRequest.Status != domain.RequestStatusApproved {
		t.Fatalf("expected approved status, got %s", result.LoanRequest.Status)
	}
	if result.PlainToken == "" {
		t.Fatal("expected plain token to be returned")
	}
	if inv.units[unitID].Status != "reserved" {
		t.Fatalf("expected inventory status reserved, got %s", inv.units[unitID].Status)
	}

	// Member checkout should fail
	_, err = svc.CheckoutLoan(context.Background(), orgID, memberID, "member", result.Loan.ID, result.PlainToken, "corr-4")
	if !errors.Is(err, domain.ErrUnauthorizedAction) {
		t.Fatalf("expected ErrUnauthorizedAction, got %v", err)
	}

	// Admin checkout with invalid token should fail
	_, err = svc.CheckoutLoan(context.Background(), orgID, adminID, "admin", result.Loan.ID, "bad-token", "corr-4")
	if !errors.Is(err, domain.ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}

	// Admin checkout with valid token succeeds
	checkedOut, err := svc.CheckoutLoan(context.Background(), orgID, adminID, "admin", result.Loan.ID, result.PlainToken, "corr-5")
	if err != nil {
		t.Fatalf("checkout failed: %v", err)
	}
	if checkedOut.Status != domain.LoanStatusCheckedOut {
		t.Fatalf("expected checked_out, got %s", checkedOut.Status)
	}
	if inv.units[unitID].Status != "borrowed" {
		t.Fatalf("expected unit status borrowed, got %s", inv.units[unitID].Status)
	}

	// Second checkout fails (single-use token)
	_, err = svc.CheckoutLoan(context.Background(), orgID, adminID, "admin", result.Loan.ID, result.PlainToken, "corr-6")
	if err == nil {
		t.Fatal("expected error on second checkout, got nil")
	}

	// Return equipment with condition damaged -> transitions unit to maintenance
	retNotes := "Scratched lens"
	returned, err := svc.ReturnLoan(context.Background(), orgID, adminID, "admin", checkedOut.ID, domain.ReturnLoanInput{
		ReturnCondition: domain.ReturnConditionDamaged,
		ReturnNotes:     &retNotes,
	}, "corr-7")
	if err != nil {
		t.Fatalf("return failed: %v", err)
	}
	if returned.Status != domain.LoanStatusReturned {
		t.Fatalf("expected returned, got %s", returned.Status)
	}
	if inv.units[unitID].Status != "maintenance" {
		t.Fatalf("expected inventory status maintenance, got %s", inv.units[unitID].Status)
	}
	if repo.reservations[result.Loan.ReservationID].Status != domain.ReservationStatusFulfilled {
		t.Fatalf("expected reservation fulfilled, got %s", repo.reservations[result.Loan.ReservationID].Status)
	}
}

func TestCancelWorkflows(t *testing.T) {
	svc, _, inv, repo, orgID, adminID, memberID := setupTestService()
	itemID := uuid.New()
	unitID := uuid.New()
	inv.addUnit(client.InventoryUnit{ID: unitID, ItemID: itemID, Status: "available"})

	now := time.Now().UTC()
	req, err := svc.CreateLoanRequest(context.Background(), orgID, memberID, domain.CreateLoanRequestInput{
		ItemID:    itemID,
		Quantity:  1,
		StartTime: now.Add(time.Hour),
		EndTime:   now.Add(3 * time.Hour),
		Purpose:   "Lab experiment",
	}, "corr-1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Cancel pending request by requester
	cancelled, err := svc.CancelLoanRequest(context.Background(), orgID, memberID, "member", req.ID, nil, "corr-2")
	if err != nil {
		t.Fatalf("cancel pending: %v", err)
	}
	if cancelled.Status != domain.RequestStatusCancelled {
		t.Fatalf("expected cancelled, got %s", cancelled.Status)
	}

	// Create second request and approve it
	req2, _ := svc.CreateLoanRequest(context.Background(), orgID, memberID, domain.CreateLoanRequestInput{
		ItemID:    itemID,
		Quantity:  1,
		StartTime: now.Add(4 * time.Hour),
		EndTime:   now.Add(8 * time.Hour),
		Purpose:   "Field test 2",
	}, "corr-3")

	appr, err := svc.ApproveLoanRequest(context.Background(), orgID, adminID, "admin", req2.ID, nil, "corr-4")
	if err != nil {
		t.Fatalf("approve: %v", err)
	}

	// Member cannot cancel approved request
	_, err = svc.CancelLoanRequest(context.Background(), orgID, memberID, "member", req2.ID, nil, "corr-5")
	if !errors.Is(err, domain.ErrUnauthorizedAction) {
		t.Fatalf("expected ErrUnauthorizedAction, got %v", err)
	}

	// Admin cancels approved request
	reason := "Schedule conflict"
	cancelledAppr, err := svc.CancelLoanRequest(context.Background(), orgID, adminID, "admin", req2.ID, &reason, "corr-6")
	if err != nil {
		t.Fatalf("admin cancel approved failed: %v", err)
	}
	if cancelledAppr.Status != domain.RequestStatusCancelled {
		t.Fatalf("expected cancelled status, got %s", cancelledAppr.Status)
	}
	if inv.units[unitID].Status != "available" {
		t.Fatalf("expected reverted inventory status available, got %s", inv.units[unitID].Status)
	}
	if repo.reservations[appr.Loan.ReservationID].Status != domain.ReservationStatusCancelled {
		t.Fatalf("expected cancelled reservation, got %s", repo.reservations[appr.Loan.ReservationID].Status)
	}
}

func TestProcessOverdue(t *testing.T) {
	svc, _, _, repo, orgID, _, memberID := setupTestService()
	past := time.Now().UTC().Add(-2 * time.Hour)

	loan := domain.Loan{
		ID:                 uuid.New(),
		OrganizationID:     orgID,
		LoanRequestID:      uuid.New(),
		ReservationID:      uuid.New(),
		UnitID:             uuid.New(),
		BorrowerUserID:     memberID,
		Status:             domain.LoanStatusCheckedOut,
		ExpectedReturnTime: past,
		CreatedAt:          past.Add(-24 * time.Hour),
		UpdatedAt:          past.Add(-24 * time.Hour),
	}
	_ = repo.CreateLoan(context.Background(), nil, &loan)

	processed, err := svc.ProcessOverdue(context.Background())
	if err != nil {
		t.Fatalf("process overdue: %v", err)
	}
	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}
	if repo.loans[loan.ID].Status != domain.LoanStatusOverdue {
		t.Fatalf("expected overdue status, got %s", repo.loans[loan.ID].Status)
	}

	// Immediate second run should process 0 (idempotent)
	secondRun, err := svc.ProcessOverdue(context.Background())
	if err != nil {
		t.Fatalf("second process overdue: %v", err)
	}
	if secondRun != 0 {
		t.Fatalf("expected 0 processed on second run, got %d", secondRun)
	}
}
