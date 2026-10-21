//go:build integration

package repository

import (
	"context"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresConnection(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	expectedDatabase := os.Getenv("EXPECTED_DATABASE")

	if databaseURL == "" || expectedDatabase == "" {
		t.Skip("DATABASE_URL and EXPECTED_DATABASE are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())

	var database string
	if err := conn.QueryRow(ctx, "SELECT current_database()").Scan(&database); err != nil {
		t.Fatal(err)
	}

	if database != expectedDatabase {
		t.Fatalf("connected to %q, expected %q", database, expectedDatabase)
	}
}

func setupTestRepository(t *testing.T) (*Repository, *pgxpool.Pool, func()) {
	t.Helper()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping database integration test")
	}

	// Apply migrations up
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to migrate up: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := NewPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}

	repo := NewRepository(pool)

	cleanup := func() {
		// Clean up tables
		ctxClean, cancelClean := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelClean()
		_, _ = pool.Exec(ctxClean, "TRUNCATE TABLE deliveries, notifications, processed_events CASCADE;")
		pool.Close()
	}

	return repo, pool, cleanup
}

func TestMigrationsUpDown(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping migration test")
	}

	// 1. Migrate Down
	if err := MigrateDown(databaseURL); err != nil {
		t.Fatalf("failed to migrate down: %v", err)
	}

	// 2. Migrate Up
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("failed to migrate up: %v", err)
	}

	// 3. Migrate Up again (should be no-change / idempotent)
	if err := MigrateUp(databaseURL); err != nil {
		t.Fatalf("expected idempotent migrate up, got: %v", err)
	}
}

func TestIdempotencyDeduplication(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	eventID := uuid.New()
	eventType := "loan.approved"

	// 1. First record should return true
	recorded, err := repo.RecordProcessedEvent(ctx, nil, eventID, eventType)
	if err != nil {
		t.Fatalf("record processed event: %v", err)
	}
	if !recorded {
		t.Fatal("expected first record attempt to succeed")
	}

	// 2. IsEventProcessed should return true
	exists, err := repo.IsEventProcessed(ctx, eventID)
	if err != nil {
		t.Fatalf("is event processed: %v", err)
	}
	if !exists {
		t.Fatal("expected event to exist in processed_events")
	}

	// 3. Second record should return false (duplicate)
	recordedSecond, err := repo.RecordProcessedEvent(ctx, nil, eventID, eventType)
	if err != nil {
		t.Fatalf("second record attempt error: %v", err)
	}
	if recordedSecond {
		t.Fatal("expected duplicate event record to return false")
	}

	// 4. Concurrent race test: 5 goroutines inserting same event ID
	concurrentEventID := uuid.New()
	var successCount int32
	var wg sync.WaitGroup

	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec, rErr := repo.RecordProcessedEvent(ctx, nil, concurrentEventID, eventType)
			if rErr == nil && rec {
				atomic.AddInt32(&successCount, 1)
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Fatalf("expected exactly 1 concurrent insert to succeed, got %d", successCount)
	}
}

func TestNotificationsAndDeliveriesCRUD(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := uuid.New()
	userID := uuid.New()
	eventID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	notif := &domain.Notification{
		ID:              uuid.New(),
		OrganizationID:  orgID,
		RecipientUserID: userID,
		EventID:         eventID,
		EventType:       domain.EventLoanApproved,
		Title:           "Loan Approved",
		Message:         "Your loan has been approved",
		Data:            map[string]any{"loan_id": uuid.New().String()},
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Create Notification
	if err := repo.CreateNotification(ctx, nil, notif); err != nil {
		t.Fatalf("create notification: %v", err)
	}

	// Create Deliveries
	inAppDelivery := &domain.Delivery{
		ID:             uuid.New(),
		NotificationID: notif.ID,
		Channel:        domain.DeliveryChannelInApp,
		Recipient:      userID.String(),
		Status:         domain.DeliveryStatusDelivered,
		AttemptsCount:  1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateDelivery(ctx, nil, inAppDelivery); err != nil {
		t.Fatalf("create in_app delivery: %v", err)
	}

	emailDelivery := &domain.Delivery{
		ID:             uuid.New(),
		NotificationID: notif.ID,
		Channel:        domain.DeliveryChannelEmail,
		Recipient:      "user@example.com",
		Status:         domain.DeliveryStatusPending,
		AttemptsCount:  0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.CreateDelivery(ctx, nil, emailDelivery); err != nil {
		t.Fatalf("create email delivery: %v", err)
	}

	// Update Email Delivery Status
	lastErr := "smtp connect timeout"
	if err := repo.UpdateDeliveryStatus(ctx, emailDelivery.ID, domain.DeliveryStatusRetrying, 1, &lastErr); err != nil {
		t.Fatalf("update delivery status: %v", err)
	}

	deliveries, err := repo.GetDeliveriesByNotificationID(ctx, notif.ID)
	if err != nil {
		t.Fatalf("get deliveries: %v", err)
	}
	if len(deliveries) != 2 {
		t.Fatalf("expected 2 deliveries, got %d", len(deliveries))
	}

	// Check Unread Count
	unreadCount, err := repo.GetUnreadCount(ctx, orgID, userID)
	if err != nil {
		t.Fatalf("get unread count: %v", err)
	}
	if unreadCount != 1 {
		t.Fatalf("expected 1 unread notification, got %d", unreadCount)
	}

	// Mark as Read
	readTime := time.Now().UTC()
	updatedNotif, err := repo.MarkAsRead(ctx, orgID, notif.ID, userID, readTime)
	if err != nil {
		t.Fatalf("mark as read: %v", err)
	}
	if updatedNotif.Status != domain.NotificationStatusRead || updatedNotif.ReadAt == nil {
		t.Fatalf("expected notification to be read with timestamp, got %+v", updatedNotif)
	}

	// Idempotent Mark as Read should preserve original readAt
	origReadAt := *updatedNotif.ReadAt
	secondReadTime := readTime.Add(10 * time.Minute)
	secondUpdated, err := repo.MarkAsRead(ctx, orgID, notif.ID, userID, secondReadTime)
	if err != nil {
		t.Fatalf("second mark as read: %v", err)
	}
	if !secondUpdated.ReadAt.Equal(origReadAt) {
		t.Fatalf("expected read_at to be preserved, got %v vs %v", secondUpdated.ReadAt, origReadAt)
	}

	// Unread Count should now be 0
	unreadCount, err = repo.GetUnreadCount(ctx, orgID, userID)
	if err != nil {
		t.Fatalf("get unread count after read: %v", err)
	}
	if unreadCount != 0 {
		t.Fatalf("expected 0 unread notifications, got %d", unreadCount)
	}
}

func TestTenantIsolation(t *testing.T) {
	repo, _, cleanup := setupTestRepository(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgA := uuid.New()
	userA := uuid.New()

	orgB := uuid.New()
	userB := uuid.New()

	now := time.Now().UTC()

	notifA := &domain.Notification{
		ID:              uuid.New(),
		OrganizationID:  orgA,
		RecipientUserID: userA,
		EventID:         uuid.New(),
		EventType:       domain.EventLoanApproved,
		Title:           "Org A Loan",
		Message:         "Message for Org A user",
		Data:            map[string]any{"org": "A"},
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateNotification(ctx, nil, notifA); err != nil {
		t.Fatalf("create notifA: %v", err)
	}

	notifB := &domain.Notification{
		ID:              uuid.New(),
		OrganizationID:  orgB,
		RecipientUserID: userB,
		EventID:         uuid.New(),
		EventType:       domain.EventLoanApproved,
		Title:           "Org B Loan",
		Message:         "Message for Org B user",
		Data:            map[string]any{"org": "B"},
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := repo.CreateNotification(ctx, nil, notifB); err != nil {
		t.Fatalf("create notifB: %v", err)
	}

	// 1. User A lists notifications for Org A -> receives notifA only
	listA, totalA, err := repo.ListNotifications(ctx, orgA, userA, nil, 10, 0)
	if err != nil {
		t.Fatalf("list A: %v", err)
	}
	if totalA != 1 || len(listA) != 1 || listA[0].ID != notifA.ID {
		t.Fatalf("expected strictly notifA, got: %+v (total: %d)", listA, totalA)
	}

	// 2. User B in Org B tries to access notifA -> ErrNotificationNotFound
	_, err = repo.GetNotificationByID(ctx, orgB, notifA.ID, userB)
	if !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("expected ErrNotificationNotFound when Org B accesses Org A notif, got: %v", err)
	}

	// 3. User B in Org B tries to mark notifA as read -> ErrNotificationNotFound
	_, err = repo.MarkAsRead(ctx, orgB, notifA.ID, userB, time.Now().UTC())
	if !errors.Is(err, domain.ErrNotificationNotFound) {
		t.Fatalf("expected ErrNotificationNotFound when Org B marks Org A notif, got: %v", err)
	}

	// 4. Verify notifA remains unread
	checkA, err := repo.GetNotificationByID(ctx, orgA, notifA.ID, userA)
	if err != nil {
		t.Fatalf("get notifA: %v", err)
	}
	if checkA.Status != domain.NotificationStatusUnread {
		t.Fatalf("expected notifA to remain unread, got: %s", checkA.Status)
	}
}
