package domain

import (
	"errors"
	"testing"
)

func TestLoanStateMachine(t *testing.T) {
	tests := []struct {
		name    string
		from    LoanStatus
		to      LoanStatus
		wantErr error
	}{
		// Valid transitions
		{
			name:    "pending_checkout to checked_out",
			from:    LoanStatusPendingCheckout,
			to:      LoanStatusCheckedOut,
			wantErr: nil,
		},
		{
			name:    "pending_checkout to cancelled",
			from:    LoanStatusPendingCheckout,
			to:      LoanStatusCancelled,
			wantErr: nil,
		},
		{
			name:    "checked_out to returned",
			from:    LoanStatusCheckedOut,
			to:      LoanStatusReturned,
			wantErr: nil,
		},
		{
			name:    "checked_out to overdue",
			from:    LoanStatusCheckedOut,
			to:      LoanStatusOverdue,
			wantErr: nil,
		},
		{
			name:    "overdue to returned",
			from:    LoanStatusOverdue,
			to:      LoanStatusReturned,
			wantErr: nil,
		},

		// Invalid transitions
		{
			name:    "pending_checkout to returned fails",
			from:    LoanStatusPendingCheckout,
			to:      LoanStatusReturned,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "pending_checkout to overdue fails",
			from:    LoanStatusPendingCheckout,
			to:      LoanStatusOverdue,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "returned to checked_out fails",
			from:    LoanStatusReturned,
			to:      LoanStatusCheckedOut,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "returned to overdue fails",
			from:    LoanStatusReturned,
			to:      LoanStatusOverdue,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "overdue to checked_out fails",
			from:    LoanStatusOverdue,
			to:      LoanStatusCheckedOut,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "cancelled to checked_out fails",
			from:    LoanStatusCancelled,
			to:      LoanStatusCheckedOut,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "self transition checked_out to checked_out fails",
			from:    LoanStatusCheckedOut,
			to:      LoanStatusCheckedOut,
			wantErr: ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLoanTransition(tt.from, tt.to)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && (err == nil || !errors.Is(err, tt.wantErr)) {
				t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestReservationStateMachine(t *testing.T) {
	tests := []struct {
		name    string
		from    ReservationStatus
		to      ReservationStatus
		wantErr error
	}{
		{
			name:    "active to fulfilled",
			from:    ReservationStatusActive,
			to:      ReservationStatusFulfilled,
			wantErr: nil,
		},
		{
			name:    "active to cancelled",
			from:    ReservationStatusActive,
			to:      ReservationStatusCancelled,
			wantErr: nil,
		},
		{
			name:    "fulfilled to active fails",
			from:    ReservationStatusFulfilled,
			to:      ReservationStatusActive,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "cancelled to active fails",
			from:    ReservationStatusCancelled,
			to:      ReservationStatusActive,
			wantErr: ErrInvalidTransition,
		},
		{
			name:    "self transition fails",
			from:    ReservationStatusActive,
			to:      ReservationStatusActive,
			wantErr: ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReservationTransition(tt.from, tt.to)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && (err == nil || !errors.Is(err, tt.wantErr)) {
				t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
			}
		})
	}
}
