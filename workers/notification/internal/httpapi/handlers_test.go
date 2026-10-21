package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/domain"
	"borrowhub/workers/notification/internal/service"
	"github.com/google/uuid"
)

type mockRepo struct {
	notifications map[uuid.UUID]*domain.Notification
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		notifications: make(map[uuid.UUID]*domain.Notification),
	}
}

func (m *mockRepo) GetNotificationByID(ctx context.Context, orgID, id, recipientUserID uuid.UUID) (*domain.Notification, error) {
	n, ok := m.notifications[id]
	if !ok || n.RecipientUserID != recipientUserID {
		return nil, domain.ErrNotificationNotFound
	}
	if orgID != uuid.Nil && n.OrganizationID != orgID {
		return nil, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (m *mockRepo) ListNotifications(ctx context.Context, orgID, recipientUserID uuid.UUID, status *domain.NotificationStatus, limit, offset int) ([]domain.Notification, int, error) {
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

func (m *mockRepo) MarkAsRead(ctx context.Context, orgID, id, recipientUserID uuid.UUID, readAt time.Time) (*domain.Notification, error) {
	n, ok := m.notifications[id]
	if !ok || n.RecipientUserID != recipientUserID {
		return nil, domain.ErrNotificationNotFound
	}
	if orgID != uuid.Nil && n.OrganizationID != orgID {
		return nil, domain.ErrNotificationNotFound
	}
	n.Status = domain.NotificationStatusRead
	if n.ReadAt == nil {
		n.ReadAt = &readAt
	}
	return n, nil
}

func (m *mockRepo) GetUnreadCount(ctx context.Context, orgID, recipientUserID uuid.UUID) (int, error) {
	var count int
	for _, n := range m.notifications {
		if n.OrganizationID == orgID && n.RecipientUserID == recipientUserID && n.Status == domain.NotificationStatusUnread {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) GetDeliveriesByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]domain.Delivery, error) {
	return nil, nil
}

func setupTestServer() (http.Handler, *mockRepo, *auth.JWTManager, string) {
	repo := newMockRepo()
	svc := service.NewNotificationService(repo)
	jwtMgr := auth.NewJWTManager("test-jwt-secret-at-least-32-bytes-long")
	internalToken := "test-internal-token"

	isReady := true
	router := NewRouter(RouterDependencies{
		Service:       svc,
		TokenManager:  jwtMgr,
		InternalToken: internalToken,
		ReadyCheck:    func() bool { return isReady },
	})

	return router, repo, jwtMgr, internalToken
}

func TestHealthEndpoints(t *testing.T) {
	router, _, _, _ := setupTestServer()

	// 1. GET /health/live
	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for live, got %d", rr.Code)
	}

	// 2. GET /health/ready
	req = httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for ready, got %d", rr.Code)
	}
}

func TestListNotifications(t *testing.T) {
	router, repo, jwtMgr, internalToken := setupTestServer()

	orgID := uuid.New()
	userID := uuid.New()
	notifID := uuid.New()

	repo.notifications[notifID] = &domain.Notification{
		ID:              notifID,
		OrganizationID:  orgID,
		RecipientUserID: userID,
		Title:           "Your Loan Request",
		Message:         "Loan approved",
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       time.Now(),
	}

	// 1. Unauthenticated -> 401
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/notifications", orgID), nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rr.Code)
	}

	// 2. Authenticated with JWT -> 200
	token, _ := jwtMgr.GenerateAccessToken(userID.String(), "user@example.com", time.Hour)
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/notifications", orgID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with JWT, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data []domain.Notification `json:"data"`
		Meta Meta                  `json:"meta"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != notifID {
		t.Fatalf("unexpected notifications list: %+v", resp.Data)
	}
	if *resp.Meta.Total != 1 {
		t.Fatalf("expected total 1, got %d", *resp.Meta.Total)
	}

	// 3. Internal token access with X-User-ID -> 200
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/notifications", orgID), nil)
	req.Header.Set("X-Internal-Token", internalToken)
	req.Header.Set("X-User-ID", userID.String())
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with internal token, got %d", rr.Code)
	}
}

func TestMarkAsRead(t *testing.T) {
	router, repo, jwtMgr, _ := setupTestServer()

	orgID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()
	notifID := uuid.New()

	repo.notifications[notifID] = &domain.Notification{
		ID:              notifID,
		OrganizationID:  orgID,
		RecipientUserID: userID,
		Title:           "Your Loan Request",
		Message:         "Loan approved",
		Status:          domain.NotificationStatusUnread,
		CreatedAt:       time.Now(),
	}

	token, _ := jwtMgr.GenerateAccessToken(userID.String(), "user@example.com", time.Hour)
	otherToken, _ := jwtMgr.GenerateAccessToken(otherUserID.String(), "other@example.com", time.Hour)

	// 1. Other user tries to mark as read -> 404 (isolated)
	req := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", notifID), nil)
	req.Header.Set("Authorization", "Bearer "+otherToken)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for other user, got %d", rr.Code)
	}

	// 2. Owner marks as read -> 200
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/notifications/%s/read", notifID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for owner, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Data domain.Notification `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal mark read response: %v", err)
	}
	if resp.Data.Status != domain.NotificationStatusRead || resp.Data.ReadAt == nil {
		t.Fatalf("expected notification to be marked read, got: %+v", resp.Data)
	}
}
