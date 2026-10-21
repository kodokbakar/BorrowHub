//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"borrowhub/services/inventory/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConnection(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	expectedDatabase := os.Getenv("EXPECTED_DATABASE")

	if databaseURL == "" || expectedDatabase == "" {
		t.Skip("DATABASE_URL and EXPECTED_DATABASE are required for integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())

	var database string

	if err := conn.QueryRow(
		ctx,
		"SELECT current_database()",
	).Scan(&database); err != nil {
		t.Fatal(err)
	}

	if database != expectedDatabase {
		t.Fatalf(
			"connected to %q, expected %q",
			database,
			expectedDatabase,
		)
	}
}

func TestInventoryRepository_Integration(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1. Migrations Up/Down Idempotency
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to run migrate up: %v", err)
	}
	if err := MigrateDown(databaseURL); err != nil {
		t.Fatalf("failed to run migrate down: %v", err)
	}
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to re-run migrate up: %v", err)
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to connect pgxpool: %v", err)
	}
	defer pool.Close()

	repos := NewRepositories(pool)

	// Clean tables
	_, err = pool.Exec(ctx, "DELETE FROM condition_logs; DELETE FROM units; DELETE FROM items; DELETE FROM categories;")
	if err != nil {
		t.Fatalf("failed to clean tables: %v", err)
	}

	// 2. Tenant Scoped Categories
	orgA := uuid.New()
	orgB := uuid.New()
	user1 := uuid.New()
	user2 := uuid.New()
	now := time.Now().Truncate(time.Microsecond)

	catA := &domain.Category{
		ID:             uuid.New(),
		OrganizationID: orgA,
		Name:           "Cameras",
		Slug:           "cameras",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Categories.Create(ctx, catA); err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	// Duplicate slug in orgA should fail
	catADup := &domain.Category{
		ID:             uuid.New(),
		OrganizationID: orgA,
		Name:           "Cameras Duplicate",
		Slug:           "cameras",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = repos.Categories.Create(ctx, catADup)
	if !errors.Is(err, domain.ErrDuplicateSlug) {
		t.Fatalf("expected ErrDuplicateSlug, got %v", err)
	}

	// Same slug in orgB should succeed
	catB := &domain.Category{
		ID:             uuid.New(),
		OrganizationID: orgB,
		Name:           "Cameras",
		Slug:           "cameras",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Categories.Create(ctx, catB); err != nil {
		t.Fatalf("create category in orgB failed: %v", err)
	}

	// 3. Items and Foreign Key Constraints
	nonExistentCatID := uuid.New()
	invalidItem := &domain.Item{
		ID:             uuid.New(),
		OrganizationID: orgA,
		CategoryID:     nonExistentCatID,
		Name:           "Invalid Item",
		IsLendable:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = repos.Items.Create(ctx, invalidItem)
	if err == nil {
		t.Fatal("expected foreign key error creating item with invalid category")
	}

	itemA := &domain.Item{
		ID:             uuid.New(),
		OrganizationID: orgA,
		CategoryID:     catA.ID,
		Name:           "Sony FX3",
		IsLendable:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Items.Create(ctx, itemA); err != nil {
		t.Fatalf("create item failed: %v", err)
	}

	// Item in Org B
	itemB := &domain.Item{
		ID:             uuid.New(),
		OrganizationID: orgB,
		CategoryID:     catB.ID,
		Name:           "Sony FX3",
		IsLendable:     true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Items.Create(ctx, itemB); err != nil {
		t.Fatalf("create item in orgB failed: %v", err)
	}

	// Cannot delete category with items (RESTRICT)
	err = repos.Categories.Delete(ctx, catA.ID)
	if !errors.Is(err, domain.ErrCategoryInUse) {
		t.Fatalf("expected ErrCategoryInUse deleting category with items, got: %v", err)
	}

	// 4. Units & Scoped Unique Barcode / Serial
	serialA1 := "SN-99001"
	unitA1 := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ItemID:         itemA.ID,
		Barcode:        "FX3-001",
		SerialNumber:   &serialA1,
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Units.Create(ctx, unitA1, user1); err != nil {
		t.Fatalf("create unitA1 failed: %v", err)
	}

	// Duplicate barcode in Org A must fail
	unitA2DupBarcode := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ItemID:         itemA.ID,
		Barcode:        "FX3-001",
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = repos.Units.Create(ctx, unitA2DupBarcode, user1)
	if !errors.Is(err, domain.ErrDuplicateBarcode) {
		t.Fatalf("expected ErrDuplicateBarcode, got: %v", err)
	}

	// Same barcode in Org B must succeed
	unitB1 := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgB,
		ItemID:         itemB.ID,
		Barcode:        "FX3-001",
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Units.Create(ctx, unitB1, user1); err != nil {
		t.Fatalf("create unit in orgB with same barcode failed: %v", err)
	}

	// Duplicate serial number in Org A must fail
	unitA3DupSerial := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ItemID:         itemA.ID,
		Barcode:        "FX3-002",
		SerialNumber:   &serialA1,
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err = repos.Units.Create(ctx, unitA3DupSerial, user1)
	if !errors.Is(err, domain.ErrDuplicateSerial) {
		t.Fatalf("expected ErrDuplicateSerial, got: %v", err)
	}

	// Multiple units with nil serial number must succeed
	unitNilSerial1 := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ItemID:         itemA.ID,
		Barcode:        "FX3-003",
		SerialNumber:   nil,
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Units.Create(ctx, unitNilSerial1, user1); err != nil {
		t.Fatalf("create unit with nil serial 1 failed: %v", err)
	}

	unitNilSerial2 := &domain.Unit{
		ID:             uuid.New(),
		OrganizationID: orgA,
		ItemID:         itemA.ID,
		Barcode:        "FX3-004",
		SerialNumber:   nil,
		Status:         domain.StatusAvailable,
		Condition:      domain.ConditionNew,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repos.Units.Create(ctx, unitNilSerial2, user1); err != nil {
		t.Fatalf("create unit with nil serial 2 failed: %v", err)
	}

	// Cannot delete item with units (RESTRICT)
	err = repos.Items.Delete(ctx, itemA.ID)
	if !errors.Is(err, domain.ErrItemInUse) {
		t.Fatalf("expected ErrItemInUse deleting item with units, got: %v", err)
	}

	// 5. Append-only Condition Logs
	logs, err := repos.Units.GetConditionLogs(ctx, unitA1.ID)
	if err != nil {
		t.Fatalf("get condition logs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 initial log, got %d", len(logs))
	}
	if logs[0].PreviousCondition != nil || logs[0].NewCondition != domain.ConditionNew {
		t.Fatalf("unexpected initial condition log: %+v", logs[0])
	}

	// Update condition to fair
	fairCond := domain.ConditionFair
	reason1 := "Scratch on body"
	unitA1, err = repos.Units.Update(ctx, unitA1.ID, domain.UpdateUnitInput{
		Condition: &fairCond,
		Reason:    &reason1,
	}, &user1)
	if err != nil {
		t.Fatalf("update unit condition to fair failed: %v", err)
	}

	// Update condition to damaged
	dmgCond := domain.ConditionDamaged
	reason2 := "Cracked display"
	unitA1, err = repos.Units.Update(ctx, unitA1.ID, domain.UpdateUnitInput{
		Condition: &dmgCond,
		Reason:    &reason2,
	}, &user2)
	if err != nil {
		t.Fatalf("update unit condition to damaged failed: %v", err)
	}
	// Query condition logs: must have 3 records in ascending order
	logs, err = repos.Units.GetConditionLogs(ctx, unitA1.ID)
	if err != nil {
		t.Fatalf("get condition logs failed: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 condition logs, got %d", len(logs))
	}
	if logs[1].NewCondition != domain.ConditionFair || logs[1].PreviousCondition == nil || *logs[1].PreviousCondition != domain.ConditionNew {
		t.Fatalf("log 1 mismatch: %+v", logs[1])
	}
	if logs[2].NewCondition != domain.ConditionDamaged || logs[2].PreviousCondition == nil || *logs[2].PreviousCondition != domain.ConditionFair {
		t.Fatalf("log 2 mismatch: %+v", logs[2])
	}

	// 6. Compare-and-Swap Status Transition
	// unitA1 is currently available
	trans1, err := repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusAvailable, domain.StatusReserved)
	if err != nil {
		t.Fatalf("transition available -> reserved failed: %v", err)
	}
	if trans1.Status != domain.StatusReserved {
		t.Fatalf("expected reserved, got %s", trans1.Status)
	}

	// Transition with mismatching from_status
	_, err = repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusAvailable, domain.StatusReserved)
	if !errors.Is(err, domain.ErrStatusMismatch) {
		t.Fatalf("expected ErrStatusMismatch, got: %v", err)
	}

	// Illegal transition: reserved -> maintenance
	_, err = repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusReserved, domain.StatusMaintenance)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}

	// Valid transition: reserved -> borrowed
	trans2, err := repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusReserved, domain.StatusBorrowed)
	if err != nil {
		t.Fatalf("transition reserved -> borrowed failed: %v", err)
	}
	if trans2.Status != domain.StatusBorrowed {
		t.Fatalf("expected borrowed, got %s", trans2.Status)
	}

	// Valid transition: borrowed -> available
	trans3, err := repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusBorrowed, domain.StatusAvailable)
	if err != nil {
		t.Fatalf("transition borrowed -> available failed: %v", err)
	}
	if trans3.Status != domain.StatusAvailable {
		t.Fatalf("expected available, got %s", trans3.Status)
	}

	// 7. Remediation Checks:
	// Maintenance transition with reason (same condition) creates condition log
	maintStatus := domain.StatusMaintenance
	maintReason := "Scheduled maintenance calibration"
	unitA1, err = repos.Units.Update(ctx, unitA1.ID, domain.UpdateUnitInput{
		Status: &maintStatus,
		Reason: &maintReason,
	}, &user1)
	if err != nil {
		t.Fatalf("update to maintenance failed: %v", err)
	}
	if unitA1.Status != domain.StatusMaintenance {
		t.Fatalf("expected maintenance status, got %s", unitA1.Status)
	}

	logs, err = repos.Units.GetConditionLogs(ctx, unitA1.ID)
	if err != nil {
		t.Fatalf("get condition logs failed: %v", err)
	}
	lastLog := logs[len(logs)-1]
	if lastLog.Reason == nil || *lastLog.Reason != "[status: maintenance] Scheduled maintenance calibration" {
		t.Fatalf("expected maintenance log reason, got: %v", lastLog.Reason)
	}

	// Complete maintenance -> available
	availStatus := domain.StatusAvailable
	unitA1, err = repos.Units.Update(ctx, unitA1.ID, domain.UpdateUnitInput{
		Status: &availStatus,
	}, &user1)
	if err != nil {
		t.Fatalf("update to available failed: %v", err)
	}

	// Lending transitions: available -> reserved -> borrowed
	_, err = repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusAvailable, domain.StatusReserved)
	if err != nil {
		t.Fatalf("transition to reserved failed: %v", err)
	}
	_, err = repos.Units.TransitionStatus(ctx, unitA1.ID, domain.StatusReserved, domain.StatusBorrowed)
	if err != nil {
		t.Fatalf("transition to borrowed failed: %v", err)
	}

	// Admin updates notes without status: does NOT overwrite borrowed status
	newNotes := "Handed to borrower at counter B"
	unitA1, err = repos.Units.Update(ctx, unitA1.ID, domain.UpdateUnitInput{
		Notes: &newNotes,
	}, &user1)
	if err != nil {
		t.Fatalf("update notes without status failed: %v", err)
	}
	if unitA1.Status != domain.StatusBorrowed {
		t.Fatalf("expected status to remain borrowed, got %s", unitA1.Status)
	}
	if unitA1.Notes == nil || *unitA1.Notes != newNotes {
		t.Fatalf("expected notes updated to %q, got %v", newNotes, unitA1.Notes)
	}

	// Fetch from DB to be 100% sure the DB status wasn't overwritten
	unitFromDB, err := repos.Units.GetByID(ctx, unitA1.ID)
	if err != nil {
		t.Fatalf("get unit failed: %v", err)
	}
	if unitFromDB.Status != domain.StatusBorrowed {
		t.Fatalf("expected DB status to remain borrowed, got %s", unitFromDB.Status)
	}
}
