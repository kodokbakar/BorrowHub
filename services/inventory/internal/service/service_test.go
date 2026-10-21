package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
)

// In-memory repositories for unit testing

type memCategoryRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Category
}

func newMemCategoryRepo() *memCategoryRepo {
	return &memCategoryRepo{data: make(map[uuid.UUID]*domain.Category)}
}

func (r *memCategoryRepo) Create(ctx context.Context, c *domain.Category) error {
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

func (r *memCategoryRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Category, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *c
	return &cp, nil
}

func (r *memCategoryRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Category, error) {
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

func (r *memCategoryRepo) Update(ctx context.Context, c *domain.Category) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[c.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *c
	r.data[c.ID] = &cp
	return nil
}

func (r *memCategoryRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.data, id)
	return nil
}

type memItemRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Item
}

func newMemItemRepo() *memItemRepo {
	return &memItemRepo{data: make(map[uuid.UUID]*domain.Item)}
}

func (r *memItemRepo) Create(ctx context.Context, item *domain.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cp := *item
	r.data[item.ID] = &cp
	return nil
}

func (r *memItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Item, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *item
	return &cp, nil
}

func (r *memItemRepo) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*domain.Item, error) {
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

func (r *memItemRepo) Update(ctx context.Context, item *domain.Item) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[item.ID]; !ok {
		return domain.ErrNotFound
	}
	cp := *item
	r.data[item.ID] = &cp
	return nil
}

func (r *memItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.data[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.data, id)
	return nil
}

type memUnitRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]*domain.Unit
	logs map[uuid.UUID][]*domain.ConditionLog
}

func newMemUnitRepo() *memUnitRepo {
	return &memUnitRepo{
		data: make(map[uuid.UUID]*domain.Unit),
		logs: make(map[uuid.UUID][]*domain.ConditionLog),
	}
}

func (r *memUnitRepo) Create(ctx context.Context, u *domain.Unit, createdBy uuid.UUID) error {
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

func (r *memUnitRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Unit, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.data[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (r *memUnitRepo) ListByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
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

func (r *memUnitRepo) ListEligibleByItem(ctx context.Context, itemID uuid.UUID) ([]*domain.Unit, error) {
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

func (r *memUnitRepo) Update(
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

func (r *memUnitRepo) TransitionStatus(
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

func (r *memUnitRepo) GetConditionLogs(ctx context.Context, unitID uuid.UUID) ([]*domain.ConditionLog, error) {
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

type memIdentityClient struct {
	members map[string]*auth.MemberInfo
}

func newMemIdentityClient() *memIdentityClient {
	return &memIdentityClient{members: make(map[string]*auth.MemberInfo)}
}

func (m *memIdentityClient) setMember(orgID, userID uuid.UUID, role string) {
	key := fmt.Sprintf("%s:%s", orgID, userID)
	m.members[key] = &auth.MemberInfo{
		IsMember:       true,
		Role:           role,
		OrganizationID: orgID,
		UserID:         userID,
	}
}

func (m *memIdentityClient) GetMember(ctx context.Context, orgID, userID uuid.UUID) (*auth.MemberInfo, error) {
	key := fmt.Sprintf("%s:%s", orgID, userID)
	if mem, ok := m.members[key]; ok {
		return mem, nil
	}
	return nil, domain.ErrNotFound
}

// Tests

func TestCategoryService(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()

	identity := newMemIdentityClient()
	identity.setMember(orgID, adminID, "admin")
	identity.setMember(orgID, memberID, "member")

	repo := newMemCategoryRepo()
	svc := NewCategoryService(repo, identity)

	// Admin create -> success
	desc := "Cameras category"
	cat, err := svc.Create(ctx, orgID, adminID, "Cameras", "cameras", &desc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cat.Name != "Cameras" || cat.Slug != "cameras" {
		t.Fatalf("unexpected category: %+v", cat)
	}

	// Member create -> forbidden
	if _, err := svc.Create(ctx, orgID, memberID, "Lenses", "lenses", nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for member, got: %v", err)
	}

	// Outsider create -> forbidden
	if _, err := svc.Create(ctx, orgID, outsiderID, "Lenses", "lenses", nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for outsider, got: %v", err)
	}

	// Member list -> success
	list, err := svc.ListByOrg(ctx, orgID, memberID)
	if err != nil {
		t.Fatalf("unexpected error listing: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 category, got %d", len(list))
	}

	// Outsider list -> forbidden
	if _, err := svc.ListByOrg(ctx, orgID, outsiderID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for outsider listing, got: %v", err)
	}

	// Admin update -> success
	newName := "Digital Cameras"
	updated, err := svc.Update(ctx, cat.ID, adminID, &newName, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error updating: %v", err)
	}
	if updated.Name != "Digital Cameras" {
		t.Fatalf("expected updated name, got %s", updated.Name)
	}

	// Member update -> forbidden
	if _, err := svc.Update(ctx, cat.ID, memberID, &newName, nil, nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for member update, got: %v", err)
	}

	// Outsider update -> not found (prevents enumeration)
	if _, err := svc.Update(ctx, cat.ID, outsiderID, &newName, nil, nil); !errors.Is(err, domain.ErrCategoryNotFound) {
		t.Fatalf("expected ErrCategoryNotFound for outsider update, got: %v", err)
	}

	// Admin delete -> success
	if err := svc.Delete(ctx, cat.ID, adminID); err != nil {
		t.Fatalf("unexpected error deleting: %v", err)
	}
}

func TestItemService(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()

	identity := newMemIdentityClient()
	identity.setMember(orgID, adminID, "admin")
	identity.setMember(orgID, memberID, "member")

	catRepo := newMemCategoryRepo()
	itemRepo := newMemItemRepo()
	catSvc := NewCategoryService(catRepo, identity)
	cat, err := catSvc.Create(ctx, orgID, adminID, "Cameras", "cameras", nil)
	if err != nil {
		t.Fatal(err)
	}

	itemSvc := NewItemService(itemRepo, catRepo, identity)

	// Admin create item -> success
	desc := "Sony mirrorless"
	item, err := itemSvc.Create(ctx, orgID, adminID, cat.ID, "Sony A7 IV", &desc, true)
	if err != nil {
		t.Fatalf("unexpected error creating item: %v", err)
	}
	if item.Name != "Sony A7 IV" || !item.IsLendable {
		t.Fatalf("unexpected item: %+v", item)
	}

	// Member create -> forbidden
	if _, err := itemSvc.Create(ctx, orgID, memberID, cat.ID, "Sony A7S III", nil, true); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}

	// Member get item -> success
	got, err := itemSvc.GetByID(ctx, item.ID, memberID)
	if err != nil {
		t.Fatalf("unexpected error getting item: %v", err)
	}
	if got.ID != item.ID {
		t.Fatalf("item mismatch: %v vs %v", got.ID, item.ID)
	}

	// Outsider get item -> not found
	if _, err := itemSvc.GetByID(ctx, item.ID, outsiderID); !errors.Is(err, domain.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound for outsider, got: %v", err)
	}
}

func TestUnitServiceAndInternal(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()
	outsiderID := uuid.New()

	identity := newMemIdentityClient()
	identity.setMember(orgID, adminID, "admin")
	identity.setMember(orgID, memberID, "member")

	catRepo := newMemCategoryRepo()
	itemRepo := newMemItemRepo()
	unitRepo := newMemUnitRepo()

	catSvc := NewCategoryService(catRepo, identity)
	cat, _ := catSvc.Create(ctx, orgID, adminID, "Cameras", "cameras", nil)

	itemSvc := NewItemService(itemRepo, catRepo, identity)
	item, _ := itemSvc.Create(ctx, orgID, adminID, cat.ID, "Sony A7 IV", nil, true)

	unitSvc := NewUnitService(unitRepo, itemRepo, identity)
	internalSvc := NewInternalInventoryService(unitRepo, itemRepo)

	// Admin create unit -> success
	serial := "SN-1001"
	unit, err := unitSvc.Create(ctx, item.ID, adminID, "CAM-001", &serial, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error creating unit: %v", err)
	}
	if unit.Barcode != "CAM-001" || unit.Status != domain.StatusAvailable || unit.Condition != domain.ConditionNew {
		t.Fatalf("unexpected unit properties: %+v", unit)
	}

	// Member create unit -> forbidden
	if _, err := unitSvc.Create(ctx, item.ID, memberID, "CAM-002", nil, nil, nil, nil); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for member, got: %v", err)
	}

	// Check condition logs initially has 1 entry
	logs, err := unitSvc.GetConditionLogs(ctx, unit.ID, memberID)
	if err != nil {
		t.Fatalf("unexpected error getting logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 initial condition log, got %d", len(logs))
	}

	// Admin update condition and status to maintenance
	maintStatus := domain.StatusMaintenance
	fairCond := domain.ConditionFair
	reason := "Lens mount loose"
	updated, err := unitSvc.Update(ctx, unit.ID, adminID, nil, nil, &maintStatus, &fairCond, nil, &reason)
	if err != nil {
		t.Fatalf("unexpected error updating unit: %v", err)
	}
	if updated.Status != domain.StatusMaintenance || updated.Condition != domain.ConditionFair {
		t.Fatalf("unexpected updated unit: %+v", updated)
	}

	// Verify condition log appended
	logs, err = unitSvc.GetConditionLogs(ctx, unit.ID, memberID)
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 condition logs after update, got %d", len(logs))
	}
	if logs[1].NewCondition != domain.ConditionFair {
		t.Fatalf("expected new condition 'fair', got %s", logs[1].NewCondition)
	}

	// Internal list eligible units: currently in maintenance, so should return 0 eligible
	eligible, err := internalSvc.ListEligibleUnits(ctx, item.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(eligible) != 0 {
		t.Fatalf("expected 0 eligible units during maintenance, got %d", len(eligible))
	}

	// Admin completes maintenance -> back to available
	availStatus := domain.StatusAvailable
	unit, err = unitSvc.Update(ctx, unit.ID, adminID, nil, nil, &availStatus, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Now internal list eligible units returns 1
	eligible, err = internalSvc.ListEligibleUnits(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(eligible) != 1 {
		t.Fatalf("expected 1 eligible unit, got %d", len(eligible))
	}

	// Lending internal transitions: available -> reserved
	transitioned, err := internalSvc.TransitionUnitStatus(ctx, unit.ID, domain.StatusAvailable, domain.StatusReserved)
	if err != nil {
		t.Fatalf("unexpected error transitioning status: %v", err)
	}
	if transitioned.Status != domain.StatusReserved {
		t.Fatalf("expected reserved status, got %s", transitioned.Status)
	}

	// Invalid transition: reserved -> maintenance directly
	if _, err := internalSvc.TransitionUnitStatus(ctx, unit.ID, domain.StatusReserved, domain.StatusMaintenance); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got %v", err)
	}

	// Status mismatch: trying to transition from available when it's reserved
	if _, err := internalSvc.TransitionUnitStatus(ctx, unit.ID, domain.StatusAvailable, domain.StatusReserved); !errors.Is(err, domain.ErrStatusMismatch) {
		t.Fatalf("expected ErrStatusMismatch, got %v", err)
	}

	// Outsider accessing unit -> not found
	if _, err := unitSvc.GetByID(ctx, unit.ID, outsiderID); !errors.Is(err, domain.ErrUnitNotFound) {
		t.Fatalf("expected ErrUnitNotFound for outsider, got: %v", err)
	}
}

func TestUnitService_RemediationRegressions(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	adminID := uuid.New()
	memberID := uuid.New()

	identity := newMemIdentityClient()
	identity.setMember(orgID, adminID, "admin")
	identity.setMember(orgID, memberID, "member")

	catRepo := newMemCategoryRepo()
	itemRepo := newMemItemRepo()
	unitRepo := newMemUnitRepo()

	catSvc := NewCategoryService(catRepo, identity)
	cat, _ := catSvc.Create(ctx, orgID, adminID, "Cameras", "cameras", nil)

	itemSvc := NewItemService(itemRepo, catRepo, identity)
	item, _ := itemSvc.Create(ctx, orgID, adminID, cat.ID, "Sony A7 IV", nil, true)

	unitSvc := NewUnitService(unitRepo, itemRepo, identity)
	internalSvc := NewInternalInventoryService(unitRepo, itemRepo)

	// Regression 1: Attempt creating unit with status 'borrowed' -> validation error
	borrowedStatus := domain.StatusBorrowed
	_, err := unitSvc.Create(ctx, item.ID, adminID, "CAM-ERR-1", nil, &borrowedStatus, nil, nil)
	if err == nil {
		t.Fatal("expected error creating unit with initial status borrowed")
	}
	var valErr *domain.ValidationError
	if !errors.As(err, &valErr) || valErr.Fields["status"] != "initial status cannot be borrowed or reserved" {
		t.Fatalf("expected 'initial status cannot be borrowed or reserved', got: %v", err)
	}

	reservedStatus := domain.StatusReserved
	_, err = unitSvc.Create(ctx, item.ID, adminID, "CAM-ERR-2", nil, &reservedStatus, nil, nil)
	if err == nil {
		t.Fatal("expected error creating unit with initial status reserved")
	}
	if !errors.As(err, &valErr) || valErr.Fields["status"] != "initial status cannot be borrowed or reserved" {
		t.Fatalf("expected 'initial status cannot be borrowed or reserved', got: %v", err)
	}

	// Create valid unit with status available and condition good
	goodCond := domain.ConditionGood
	validUnit, err := unitSvc.Create(ctx, item.ID, adminID, "CAM-GOOD-1", nil, nil, &goodCond, nil)
	if err != nil {
		t.Fatalf("unexpected error creating unit: %v", err)
	}

	// Regression 2: Transitioning to maintenance with reason (same condition) -> condition log created
	maintStatus := domain.StatusMaintenance
	maintReason := "Sensor cleaning"
	updatedMaint, err := unitSvc.Update(ctx, validUnit.ID, adminID, nil, nil, &maintStatus, nil, nil, &maintReason)
	if err != nil {
		t.Fatalf("unexpected error updating unit to maintenance: %v", err)
	}
	if updatedMaint.Status != domain.StatusMaintenance || updatedMaint.Condition != domain.ConditionGood {
		t.Fatalf("unexpected unit status/condition: %+v", updatedMaint)
	}

	logs, err := unitSvc.GetConditionLogs(ctx, validUnit.ID, adminID)
	if err != nil {
		t.Fatalf("unexpected error getting condition logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 condition logs, got %d", len(logs))
	}
	maintLog := logs[1]
	if maintLog.PreviousCondition == nil || *maintLog.PreviousCondition != domain.ConditionGood {
		t.Fatalf("expected previous condition good, got: %v", maintLog.PreviousCondition)
	}
	if maintLog.NewCondition != domain.ConditionGood {
		t.Fatalf("expected new condition good, got: %s", maintLog.NewCondition)
	}
	expectedReason := "[status: maintenance] Sensor cleaning"
	if maintLog.Reason == nil || *maintLog.Reason != expectedReason {
		t.Fatalf("expected reason %q, got %v", expectedReason, maintLog.Reason)
	}

	// Transition back to available so lending can borrow it
	availStatus := domain.StatusAvailable
	_, err = unitSvc.Update(ctx, validUnit.ID, adminID, nil, nil, &availStatus, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error setting available: %v", err)
	}

	// Lending transitions: available -> reserved -> borrowed
	_, err = internalSvc.TransitionUnitStatus(ctx, validUnit.ID, domain.StatusAvailable, domain.StatusReserved)
	if err != nil {
		t.Fatalf("unexpected error transitioning to reserved: %v", err)
	}
	_, err = internalSvc.TransitionUnitStatus(ctx, validUnit.ID, domain.StatusReserved, domain.StatusBorrowed)
	if err != nil {
		t.Fatalf("unexpected error transitioning to borrowed: %v", err)
	}

	// Regression 3: Updating notes without status does not overwrite concurrent status
	newNotes := "Stored in secure locker 4"
	updatedNotesUnit, err := unitSvc.Update(ctx, validUnit.ID, adminID, nil, nil, nil, nil, &newNotes, nil)
	if err != nil {
		t.Fatalf("unexpected error updating notes: %v", err)
	}
	if updatedNotesUnit.Status != domain.StatusBorrowed {
		t.Fatalf("expected status to remain borrowed, got %s", updatedNotesUnit.Status)
	}
	if updatedNotesUnit.Notes == nil || *updatedNotesUnit.Notes != newNotes {
		t.Fatalf("expected notes %q, got %v", newNotes, updatedNotesUnit.Notes)
	}

	// Fetch unit from repo directly to verify status in repository was not overwritten
	unitFromRepo, err := unitRepo.GetByID(ctx, validUnit.ID)
	if err != nil {
		t.Fatalf("unexpected error fetching unit from repo: %v", err)
	}
	if unitFromRepo.Status != domain.StatusBorrowed {
		t.Fatalf("expected repo status to remain borrowed, got %s", unitFromRepo.Status)
	}
}
