package domain

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type NotificationStatus string

const (
	NotificationStatusUnread NotificationStatus = "unread"
	NotificationStatusRead   NotificationStatus = "read"
)

type DeliveryChannel string

const (
	DeliveryChannelInApp DeliveryChannel = "in_app"
	DeliveryChannelEmail DeliveryChannel = "email"
)

type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
	DeliveryStatusRetrying  DeliveryStatus = "retrying"
)

const (
	EventLoanRequested  = "loan.requested"
	EventLoanApproved   = "loan.approved"
	EventLoanCheckedOut = "loan.checked_out"
	EventLoanOverdue    = "loan.overdue"
	EventLoanReturned   = "loan.returned"
)

type DomainEnvelope struct {
	EventID        string         `json:"event_id"`
	EventType      string         `json:"event_type"`
	EventVersion   int            `json:"event_version"`
	Source         string         `json:"source"`
	Subject        string         `json:"subject"`
	OrganizationID string         `json:"organization_id"`
	OccurredAt     string         `json:"occurred_at"`
	CorrelationID  string         `json:"correlation_id"`
	CausationID    string         `json:"causation_id"`
	Data           map[string]any `json:"data"`
}

type ProcessedEvent struct {
	EventID     uuid.UUID `json:"event_id"`
	EventType   string    `json:"event_type"`
	ProcessedAt time.Time `json:"processed_at"`
	Status      string    `json:"status"`
}

type Notification struct {
	ID              uuid.UUID          `json:"id"`
	OrganizationID  uuid.UUID          `json:"organization_id"`
	RecipientUserID uuid.UUID          `json:"recipient_user_id"`
	EventID         uuid.UUID          `json:"event_id"`
	EventType       string             `json:"event_type"`
	Title           string             `json:"title"`
	Message         string             `json:"message"`
	Data            map[string]any     `json:"data"`
	Status          NotificationStatus `json:"status"`
	ReadAt          *time.Time         `json:"read_at,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type Delivery struct {
	ID             uuid.UUID       `json:"id"`
	NotificationID uuid.UUID       `json:"notification_id"`
	Channel        DeliveryChannel `json:"channel"`
	Recipient      string          `json:"recipient"`
	Status         DeliveryStatus  `json:"status"`
	AttemptsCount  int             `json:"attempts_count"`
	LastError      *string         `json:"last_error,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// CanTransitionTo checks if state transition is legal.
func (d *Delivery) CanTransitionTo(target DeliveryStatus) bool {
	switch d.Status {
	case DeliveryStatusPending:
		return target == DeliveryStatusDelivered || target == DeliveryStatusRetrying || target == DeliveryStatusFailed
	case DeliveryStatusRetrying:
		return target == DeliveryStatusDelivered || target == DeliveryStatusRetrying || target == DeliveryStatusFailed
	case DeliveryStatusDelivered, DeliveryStatusFailed:
		// Terminal states cannot transition
		return false
	default:
		return false
	}
}

// MarkDelivered transitions delivery to delivered state.
func (d *Delivery) MarkDelivered(now time.Time) error {
	if !d.CanTransitionTo(DeliveryStatusDelivered) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, d.Status, DeliveryStatusDelivered)
	}
	d.Status = DeliveryStatusDelivered
	d.AttemptsCount++
	d.LastError = nil
	d.UpdatedAt = now
	return nil
}

// MarkRetrying transitions delivery to retrying state.
func (d *Delivery) MarkRetrying(errStr string, now time.Time) error {
	if !d.CanTransitionTo(DeliveryStatusRetrying) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, d.Status, DeliveryStatusRetrying)
	}
	d.Status = DeliveryStatusRetrying
	d.AttemptsCount++
	d.LastError = &errStr
	d.UpdatedAt = now
	return nil
}

// MarkFailed transitions delivery to failed terminal state.
func (d *Delivery) MarkFailed(errStr string, now time.Time) error {
	if !d.CanTransitionTo(DeliveryStatusFailed) {
		return fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStateTransition, d.Status, DeliveryStatusFailed)
	}
	d.Status = DeliveryStatusFailed
	d.AttemptsCount++
	d.LastError = &errStr
	d.UpdatedAt = now
	return nil
}
