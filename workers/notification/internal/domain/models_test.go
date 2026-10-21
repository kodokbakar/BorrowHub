package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDeliveryStateMachine(t *testing.T) {
	now := time.Now().UTC()

	// 1. Initial State: pending, attempts = 0
	d := Delivery{
		ID:             uuid.New(),
		NotificationID: uuid.New(),
		Channel:        DeliveryChannelEmail,
		Recipient:      "user@example.com",
		Status:         DeliveryStatusPending,
		AttemptsCount:  0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if d.Status != DeliveryStatusPending || d.AttemptsCount != 0 {
		t.Fatalf("unexpected initial state: %+v", d)
	}

	// 2. Pending -> Retrying
	err := d.MarkRetrying("transient error", now.Add(time.Second))
	if err != nil {
		t.Fatalf("expected retrying transition, got err: %v", err)
	}
	if d.Status != DeliveryStatusRetrying || d.AttemptsCount != 1 || *d.LastError != "transient error" {
		t.Fatalf("unexpected retrying state: %+v", d)
	}

	// 3. Retrying -> Retrying again
	err = d.MarkRetrying("second error", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("expected second retrying transition, got err: %v", err)
	}
	if d.Status != DeliveryStatusRetrying || d.AttemptsCount != 2 || *d.LastError != "second error" {
		t.Fatalf("unexpected retrying state 2: %+v", d)
	}

	// 4. Retrying -> Delivered
	err = d.MarkDelivered(now.Add(3 * time.Second))
	if err != nil {
		t.Fatalf("expected delivered transition, got err: %v", err)
	}
	if d.Status != DeliveryStatusDelivered || d.AttemptsCount != 3 || d.LastError != nil {
		t.Fatalf("unexpected delivered state: %+v", d)
	}

	// 5. Terminal State Guard: Delivered cannot transition
	err = d.MarkFailed("error after delivered", now.Add(4*time.Second))
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got: %v", err)
	}

	err = d.MarkRetrying("retry after delivered", now.Add(4*time.Second))
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got: %v", err)
	}

	// 6. Terminal State Guard: Failed cannot transition
	dFailed := Delivery{
		ID:             uuid.New(),
		NotificationID: uuid.New(),
		Channel:        DeliveryChannelEmail,
		Recipient:      "user@example.com",
		Status:         DeliveryStatusPending,
		AttemptsCount:  0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := dFailed.MarkFailed("fatal error", now.Add(time.Second)); err != nil {
		t.Fatalf("expected failed transition, got: %v", err)
	}
	if dFailed.Status != DeliveryStatusFailed || dFailed.AttemptsCount != 1 {
		t.Fatalf("unexpected failed state: %+v", dFailed)
	}

	err = dFailed.MarkDelivered(now.Add(2 * time.Second))
	if !errors.Is(err, ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition on terminal failed, got: %v", err)
	}
}
