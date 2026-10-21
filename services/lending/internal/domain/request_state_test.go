package domain

import (
	"errors"
	"testing"
)

func TestRequestStateMachine(t *testing.T) {
	tests := []struct {
		name        string
		from        RequestStatus
		to          RequestStatus
		role        string
		isRequester bool
		wantErr     error
	}{
		// Valid transitions
		{
			name:        "pending to approved by admin",
			from:        RequestStatusPending,
			to:          RequestStatusApproved,
			role:        "admin",
			isRequester: false,
			wantErr:     nil,
		},
		{
			name:        "pending to approved by owner",
			from:        RequestStatusPending,
			to:          RequestStatusApproved,
			role:        "owner",
			isRequester: false,
			wantErr:     nil,
		},
		{
			name:        "pending to rejected by admin",
			from:        RequestStatusPending,
			to:          RequestStatusRejected,
			role:        "admin",
			isRequester: false,
			wantErr:     nil,
		},
		{
			name:        "pending to cancelled by requester",
			from:        RequestStatusPending,
			to:          RequestStatusCancelled,
			role:        "member",
			isRequester: true,
			wantErr:     nil,
		},
		{
			name:        "pending to cancelled by admin",
			from:        RequestStatusPending,
			to:          RequestStatusCancelled,
			role:        "admin",
			isRequester: false,
			wantErr:     nil,
		},
		{
			name:        "approved to cancelled by admin",
			from:        RequestStatusApproved,
			to:          RequestStatusCancelled,
			role:        "admin",
			isRequester: false,
			wantErr:     nil,
		},

		// Unauthorized transitions
		{
			name:        "pending to approved by member fails",
			from:        RequestStatusPending,
			to:          RequestStatusApproved,
			role:        "member",
			isRequester: false,
			wantErr:     ErrUnauthorizedAction,
		},
		{
			name:        "pending to rejected by member fails",
			from:        RequestStatusPending,
			to:          RequestStatusRejected,
			role:        "member",
			isRequester: false,
			wantErr:     ErrUnauthorizedAction,
		},
		{
			name:        "pending to cancelled by other member fails",
			from:        RequestStatusPending,
			to:          RequestStatusCancelled,
			role:        "member",
			isRequester: false,
			wantErr:     ErrUnauthorizedAction,
		},
		{
			name:        "approved to cancelled by requester fails",
			from:        RequestStatusApproved,
			to:          RequestStatusCancelled,
			role:        "member",
			isRequester: true,
			wantErr:     ErrUnauthorizedAction,
		},

		// Invalid transitions
		{
			name:        "self transition pending to pending fails",
			from:        RequestStatusPending,
			to:          RequestStatusPending,
			role:        "admin",
			isRequester: false,
			wantErr:     ErrInvalidTransition,
		},
		{
			name:        "approved to rejected fails",
			from:        RequestStatusApproved,
			to:          RequestStatusRejected,
			role:        "admin",
			isRequester: false,
			wantErr:     ErrInvalidTransition,
		},
		{
			name:        "rejected to approved fails",
			from:        RequestStatusRejected,
			to:          RequestStatusApproved,
			role:        "admin",
			isRequester: false,
			wantErr:     ErrInvalidTransition,
		},
		{
			name:        "cancelled to approved fails",
			from:        RequestStatusCancelled,
			to:          RequestStatusApproved,
			role:        "admin",
			isRequester: false,
			wantErr:     ErrInvalidTransition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequestTransition(tt.from, tt.to, tt.role, tt.isRequester)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != nil && (err == nil || !errors.Is(err, tt.wantErr)) {
				t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
			}
		})
	}
}
