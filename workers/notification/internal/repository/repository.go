package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"borrowhub/workers/notification/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

func (r *Repository) RunInTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

// IsEventProcessed checks if event_id already exists in processed_events.
func (r *Repository) IsEventProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM processed_events WHERE event_id = $1)`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, eventID).Scan(&exists); err != nil {
		return false, mapPgError(err)
	}
	return exists, nil
}

// RecordProcessedEvent atomically attempts to record the event. Returns true if newly recorded, false if already processed.
func (r *Repository) RecordProcessedEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, eventType string) (bool, error) {
	query := `
		INSERT INTO processed_events (event_id, event_type, processed_at, status)
		VALUES ($1, $2, CURRENT_TIMESTAMP, 'success')
		ON CONFLICT (event_id) DO NOTHING
		RETURNING event_id;
	`
	var returnedID uuid.UUID
	var err error
	if tx != nil {
		err = tx.QueryRow(ctx, query, eventID, eventType).Scan(&returnedID)
	} else {
		err = r.pool.QueryRow(ctx, query, eventID, eventType).Scan(&returnedID)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, mapPgError(err)
	}
	return true, nil
}

func (r *Repository) CreateNotification(ctx context.Context, tx pgx.Tx, n *domain.Notification) error {
	query := `
		INSERT INTO notifications (
			id, organization_id, recipient_user_id, event_id, event_type,
			title, message, data, status, read_at, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	dataBytes, err := json.Marshal(n.Data)
	if err != nil {
		return fmt.Errorf("marshaling notification data: %w", err)
	}

	var execErr error
	if tx != nil {
		_, execErr = tx.Exec(
			ctx, query,
			n.ID, n.OrganizationID, n.RecipientUserID, n.EventID, n.EventType,
			n.Title, n.Message, dataBytes, string(n.Status), n.ReadAt, n.CreatedAt, n.UpdatedAt,
		)
	} else {
		_, execErr = r.pool.Exec(
			ctx, query,
			n.ID, n.OrganizationID, n.RecipientUserID, n.EventID, n.EventType,
			n.Title, n.Message, dataBytes, string(n.Status), n.ReadAt, n.CreatedAt, n.UpdatedAt,
		)
	}

	return mapPgError(execErr)
}

func (r *Repository) CreateDelivery(ctx context.Context, tx pgx.Tx, d *domain.Delivery) error {
	query := `
		INSERT INTO deliveries (
			id, notification_id, channel, recipient, status, attempts_count, last_error, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`
	var execErr error
	if tx != nil {
		_, execErr = tx.Exec(
			ctx, query,
			d.ID, d.NotificationID, string(d.Channel), d.Recipient, string(d.Status),
			d.AttemptsCount, d.LastError, d.CreatedAt, d.UpdatedAt,
		)
	} else {
		_, execErr = r.pool.Exec(
			ctx, query,
			d.ID, d.NotificationID, string(d.Channel), d.Recipient, string(d.Status),
			d.AttemptsCount, d.LastError, d.CreatedAt, d.UpdatedAt,
		)
	}

	return mapPgError(execErr)
}

func (r *Repository) UpdateDeliveryStatus(
	ctx context.Context,
	id uuid.UUID,
	status domain.DeliveryStatus,
	attemptsCount int,
	lastError *string,
) error {
	query := `
		UPDATE deliveries
		SET status = $2, attempts_count = $3, last_error = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.pool.Exec(ctx, query, id, string(status), attemptsCount, lastError)
	return mapPgError(err)
}

func (r *Repository) GetNotificationByID(ctx context.Context, orgID, id, recipientUserID uuid.UUID) (*domain.Notification, error) {
	var query string
	var err error
	var n domain.Notification
	var rawData []byte
	var statusStr string

	if orgID != uuid.Nil {
		query = `
			SELECT id, organization_id, recipient_user_id, event_id, event_type,
			       title, message, data, status, read_at, created_at, updated_at
			FROM notifications
			WHERE id = $1 AND organization_id = $2 AND recipient_user_id = $3
		`
		err = r.pool.QueryRow(ctx, query, id, orgID, recipientUserID).Scan(
			&n.ID, &n.OrganizationID, &n.RecipientUserID, &n.EventID, &n.EventType,
			&n.Title, &n.Message, &rawData, &statusStr, &n.ReadAt, &n.CreatedAt, &n.UpdatedAt,
		)
	} else {
		query = `
			SELECT id, organization_id, recipient_user_id, event_id, event_type,
			       title, message, data, status, read_at, created_at, updated_at
			FROM notifications
			WHERE id = $1 AND recipient_user_id = $2
		`
		err = r.pool.QueryRow(ctx, query, id, recipientUserID).Scan(
			&n.ID, &n.OrganizationID, &n.RecipientUserID, &n.EventID, &n.EventType,
			&n.Title, &n.Message, &rawData, &statusStr, &n.ReadAt, &n.CreatedAt, &n.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, mapPgError(err)
	}

	n.Status = domain.NotificationStatus(statusStr)
	if len(rawData) > 0 {
		_ = json.Unmarshal(rawData, &n.Data)
	}
	return &n, nil
}

func (r *Repository) ListNotifications(
	ctx context.Context,
	orgID, recipientUserID uuid.UUID,
	status *domain.NotificationStatus,
	limit, offset int,
) ([]domain.Notification, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	countQuery := `
		SELECT COUNT(*)
		FROM notifications
		WHERE organization_id = $1 AND recipient_user_id = $2
	`
	listQuery := `
		SELECT id, organization_id, recipient_user_id, event_id, event_type,
		       title, message, data, status, read_at, created_at, updated_at
		FROM notifications
		WHERE organization_id = $1 AND recipient_user_id = $2
	`
	var total int
	var rows pgx.Rows
	var err error

	if status != nil && *status != "" {
		countQuery += " AND status = $3"
		if err := r.pool.QueryRow(ctx, countQuery, orgID, recipientUserID, string(*status)).Scan(&total); err != nil {
			return nil, 0, mapPgError(err)
		}

		listQuery += " AND status = $3 ORDER BY created_at DESC LIMIT $4 OFFSET $5"
		rows, err = r.pool.Query(ctx, listQuery, orgID, recipientUserID, string(*status), limit, offset)
	} else {
		if err := r.pool.QueryRow(ctx, countQuery, orgID, recipientUserID).Scan(&total); err != nil {
			return nil, 0, mapPgError(err)
		}

		listQuery += " ORDER BY created_at DESC LIMIT $3 OFFSET $4"
		rows, err = r.pool.Query(ctx, listQuery, orgID, recipientUserID, limit, offset)
	}

	if err != nil {
		return nil, 0, mapPgError(err)
	}
	defer rows.Close()

	items := make([]domain.Notification, 0, limit)
	for rows.Next() {
		var n domain.Notification
		var rawData []byte
		var statusStr string

		if err := rows.Scan(
			&n.ID, &n.OrganizationID, &n.RecipientUserID, &n.EventID, &n.EventType,
			&n.Title, &n.Message, &rawData, &statusStr, &n.ReadAt, &n.CreatedAt, &n.UpdatedAt,
		); err != nil {
			return nil, 0, mapPgError(err)
		}

		n.Status = domain.NotificationStatus(statusStr)
		if len(rawData) > 0 {
			_ = json.Unmarshal(rawData, &n.Data)
		}
		items = append(items, n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, mapPgError(err)
	}

	return items, total, nil
}

func (r *Repository) MarkAsRead(
	ctx context.Context,
	orgID, id, recipientUserID uuid.UUID,
	readAt time.Time,
) (*domain.Notification, error) {
	var query string
	var err error
	var n domain.Notification
	var rawData []byte
	var statusStr string

	if orgID != uuid.Nil {
		query = `
			UPDATE notifications
			SET status = 'read', read_at = COALESCE(read_at, $4), updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND organization_id = $2 AND recipient_user_id = $3
			RETURNING id, organization_id, recipient_user_id, event_id, event_type,
			          title, message, data, status, read_at, created_at, updated_at;
		`
		err = r.pool.QueryRow(ctx, query, id, orgID, recipientUserID, readAt).Scan(
			&n.ID, &n.OrganizationID, &n.RecipientUserID, &n.EventID, &n.EventType,
			&n.Title, &n.Message, &rawData, &statusStr, &n.ReadAt, &n.CreatedAt, &n.UpdatedAt,
		)
	} else {
		query = `
			UPDATE notifications
			SET status = 'read', read_at = COALESCE(read_at, $3), updated_at = CURRENT_TIMESTAMP
			WHERE id = $1 AND recipient_user_id = $2
			RETURNING id, organization_id, recipient_user_id, event_id, event_type,
			          title, message, data, status, read_at, created_at, updated_at;
		`
		err = r.pool.QueryRow(ctx, query, id, recipientUserID, readAt).Scan(
			&n.ID, &n.OrganizationID, &n.RecipientUserID, &n.EventID, &n.EventType,
			&n.Title, &n.Message, &rawData, &statusStr, &n.ReadAt, &n.CreatedAt, &n.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotificationNotFound
		}
		return nil, mapPgError(err)
	}

	n.Status = domain.NotificationStatus(statusStr)
	if len(rawData) > 0 {
		_ = json.Unmarshal(rawData, &n.Data)
	}
	return &n, nil
}

func (r *Repository) GetUnreadCount(ctx context.Context, orgID, recipientUserID uuid.UUID) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM notifications
		WHERE organization_id = $1 AND recipient_user_id = $2 AND status = 'unread'
	`
	var count int
	if err := r.pool.QueryRow(ctx, query, orgID, recipientUserID).Scan(&count); err != nil {
		return 0, mapPgError(err)
	}
	return count, nil
}

func (r *Repository) GetDeliveriesByNotificationID(ctx context.Context, notificationID uuid.UUID) ([]domain.Delivery, error) {
	query := `
		SELECT id, notification_id, channel, recipient, status, attempts_count, last_error, created_at, updated_at
		FROM deliveries
		WHERE notification_id = $1
		ORDER BY created_at ASC
	`
	rows, err := r.pool.Query(ctx, query, notificationID)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()

	var deliveries []domain.Delivery
	for rows.Next() {
		var d domain.Delivery
		var channelStr, statusStr string
		if err := rows.Scan(
			&d.ID, &d.NotificationID, &channelStr, &d.Recipient, &statusStr,
			&d.AttemptsCount, &d.LastError, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, mapPgError(err)
		}
		d.Channel = domain.DeliveryChannel(channelStr)
		d.Status = domain.DeliveryStatus(statusStr)
		deliveries = append(deliveries, d)
	}
	return deliveries, rows.Err()
}
