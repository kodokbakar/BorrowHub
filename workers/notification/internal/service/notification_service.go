package service

import (
	"context"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"github.com/google/uuid"
)

type NotificationRepository interface {
	GetNotificationByID(ctx context.Context, orgID, id, recipientUserID uuid.UUID) (*domain.Notification, error)
	ListNotifications(ctx context.Context, orgID, recipientUserID uuid.UUID, status *domain.NotificationStatus, limit, offset int) ([]domain.Notification, int, error)
	MarkAsRead(ctx context.Context, orgID, id, recipientUserID uuid.UUID, readAt time.Time) (*domain.Notification, error)
	GetUnreadCount(ctx context.Context, orgID, recipientUserID uuid.UUID) (int, error)
	GetDeliveriesByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]domain.Delivery, error)
}

type NotificationService struct {
	repo NotificationRepository
}

func NewNotificationService(repo NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) ListNotifications(
	ctx context.Context,
	orgID, recipientUserID uuid.UUID,
	status *domain.NotificationStatus,
	limit, offset int,
) ([]domain.Notification, int, error) {
	return s.repo.ListNotifications(ctx, orgID, recipientUserID, status, limit, offset)
}

func (s *NotificationService) MarkAsRead(
	ctx context.Context,
	orgID, id, recipientUserID uuid.UUID,
) (*domain.Notification, error) {
	return s.repo.MarkAsRead(ctx, orgID, id, recipientUserID, time.Now().UTC())
}

func (s *NotificationService) GetUnreadCount(
	ctx context.Context,
	orgID, recipientUserID uuid.UUID,
) (int, error) {
	return s.repo.GetUnreadCount(ctx, orgID, recipientUserID)
}

func (s *NotificationService) GetNotification(
	ctx context.Context,
	orgID, id, recipientUserID uuid.UUID,
) (*domain.Notification, error) {
	return s.repo.GetNotificationByID(ctx, orgID, id, recipientUserID)
}
