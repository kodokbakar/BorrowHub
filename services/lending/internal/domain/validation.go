package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	MaxLoanDuration = 30 * 24 * time.Hour
	PastGracePeriod = 5 * time.Minute
)

func DoReservationsOverlap(start1, end1, start2, end2 time.Time) bool {
	return start1.Before(end2) && start2.Before(end1)
}

func GenerateSecureToken() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("generating random token: %w", err)
	}
	plainToken := hex.EncodeToString(bytes)
	tokenHash := HashToken(plainToken)
	return plainToken, tokenHash, nil
}

func HashToken(plainToken string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(plainToken)))
	return hex.EncodeToString(sum[:])
}

type CreateLoanRequestInput struct {
	ItemID    uuid.UUID `json:"item_id"`
	Quantity  int       `json:"quantity"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Purpose   string    `json:"purpose"`
	Notes     *string   `json:"notes,omitempty"`
}

func ValidateCreateLoanRequestInput(input CreateLoanRequestInput, now time.Time) error {
	fields := make(map[string]any)

	if input.ItemID == uuid.Nil {
		fields["item_id"] = "item_id is required"
	}

	if input.Quantity <= 0 {
		fields["quantity"] = "quantity must be greater than 0"
	} else if input.Quantity > 1 {
		fields["quantity"] = "quantity must be 1 (multi-unit loans currently not supported)"
	}

	if input.StartTime.IsZero() {
		fields["start_time"] = "start_time is required"
	}

	if input.EndTime.IsZero() {
		fields["end_time"] = "end_time is required"
	}

	if !input.StartTime.IsZero() && !input.EndTime.IsZero() {
		if !input.StartTime.Before(input.EndTime) {
			fields["end_time"] = "end_time must be after start_time"
		} else {
			if input.StartTime.Before(now.Add(-PastGracePeriod)) {
				fields["start_time"] = "start_time cannot be in the past"
			}
			if input.EndTime.Sub(input.StartTime) > MaxLoanDuration {
				fields["end_time"] = "loan duration cannot exceed 30 days"
			}
		}
	}

	purpose := strings.TrimSpace(input.Purpose)
	if purpose == "" {
		fields["purpose"] = "purpose is required"
	} else if len(purpose) > 500 {
		fields["purpose"] = "purpose cannot exceed 500 characters"
	}

	if input.Notes != nil && len(*input.Notes) > 1000 {
		fields["notes"] = "notes cannot exceed 1000 characters"
	}

	if len(fields) > 0 {
		return NewValidationError("validation failed for loan request", fields)
	}

	return nil
}

type ReturnLoanInput struct {
	ReturnCondition ReturnCondition `json:"return_condition"`
	ReturnNotes     *string         `json:"return_notes,omitempty"`
}

func ValidateReturnLoanInput(input ReturnLoanInput) error {
	fields := make(map[string]any)

	if !input.ReturnCondition.IsValid() {
		fields["return_condition"] = "valid return_condition is required (new, good, fair, poor, damaged)"
	}

	if input.ReturnNotes != nil && len(*input.ReturnNotes) > 1000 {
		fields["return_notes"] = "return_notes cannot exceed 1000 characters"
	}

	if len(fields) > 0 {
		return NewValidationError("validation failed for loan return", fields)
	}

	return nil
}
