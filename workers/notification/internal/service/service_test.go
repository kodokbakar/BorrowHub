package service

import (
	"context"
	"testing"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"github.com/google/uuid"
)

type mockNotificationRepo struct {
	notifications map[uuid.UUID]*domain.Notification
}

func newMockNotificationRepo() *mockNotificationRepo {
	return &mockNotificationRepo{
		notifications: make(map[uuid.UUID]*domain.Notification),
	}
}

func (m *mockNotificationRepo) GetNotificationByID(ctx context.Context, orgID, id, recipientUserID uuid.UUID) (*domain.Notification, error) {
	n, ok := m.notifications[id]
	if !ok || n.OrganizationID != orgID || n.RecipientUserID != recipientUserID {
		return nil, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockNotificationRepo) ListNotifications(ctx context.Context, orgID, recipientUserID uuid.UUID, status *domain.NotificationStatus, limit, offset int) ([]domain.Notification, int, error) {
	var results []domain.Notification
	for _, n := range m.notifications {
		if n.OrganizationID == orgID && n.RecipientUserID == recipientUserID {
			if status != nil && *status != "" && n.Status != *status {
				continue
			}
			results = append(results, *n)
		}
	}
	return results, len(results), nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, orgID, id, recipientUserID uuid.UUID, readAt time.Time) (*domain.Notification, error) {
	n, ok := m.notifications[id]
	if !ok || n.OrganizationID != orgID || n.RecipientUserID != recipientUserID {
		return nil, domain.ErrNotificationNotFound
	}
	n.Status = domain.NotificationStatusRead
	if n.ReadAt == nil {
		n.ReadAt = &readAt
	}
	return n, nil
}

func (m *mockNotificationRepo) GetUnreadCount(ctx context.Context, orgID, recipientUserID uuid.UUID) (int, error) {
	var count int
	for _, n := range m.notifications {
		if n.OrganizationID == orgID && n.RecipientUserID == recipientUserID && n.Status == domain.NotificationStatusUnread {
			count++
		}
	}
	return count, nil
}

func (m *mockNotificationRepo) GetDeliveriesByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]domain.Delivery, error) {
	return nil, nil
}

func TestNotificationService(t *testing.T) {
	repo := newMockNotificationRepo()
	svc := NewNotificationService(repo)
	ctx := context.Background()

	orgID := uuid.New()
	userID := uuid.New()
	notifID := uuid.New()

	repo.notifications[notifID] = &domain.Notification{
		ID:              notifID,
		OrganizationID:  orgID,
		RecipientUserID: userID,
		Title:           "Test Title",
		Message:         "Test Message",
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       time.Now(),
	}

	// 1. Get Unread Count
	count, err := svc.GetUnreadCount(ctx, orgID, userID)
	if err != nil || count != 1 {
		t.Fatalf("expected count 1, got %d, err: %v", count, err)
	}

	// 2. List Notifications
	list, total, err := svc.ListNotifications(ctx, orgID, userID, nil, 10, 0)
	if err != nil || total != 1 || len(list) != 1 {
		t.Fatalf("expected 1 item, got %d (total %d), err: %v", len(list), total, err)
	}

	// 3. Mark As Read
	updated, err := svc.MarkAsRead(ctx, orgID, notifID, userID)
	if err != nil || updated.Status != domain.NotificationStatusRead || updated.ReadAt == nil {
		t.Fatalf("expected marked read with read_at, got: %+v, err: %v", updated, err)
	}

	// 4. Get Unread Count after read
	count, err = svc.GetUnreadCount(ctx, orgID, userID)
	if err != nil || count != 0 {
		t.Fatalf("expected count 0, got %d, err: %v", count, err)
	}

	// 5. Cross-tenant access
	otherOrgID := uuid.New()
	_, err = svc.GetNotification(ctx, otherOrgID, notifID, userID)
	if err != domain.ErrNotificationNotFound {
		t.Fatalf("expected ErrNotificationNotFound for other org, got: %v", err)
	}
}
