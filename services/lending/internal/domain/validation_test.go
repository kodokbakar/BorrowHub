package domain

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidateCreateLoanRequestInput(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	itemID := uuid.New()

	tests := []struct {
		name      string
		input     CreateLoanRequestInput
		shouldErr bool
	}{
		{
			name: "valid input",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Field research experiment",
			},
			shouldErr: false,
		},
		{
			name: "nil item_id",
			input: CreateLoanRequestInput{
				ItemID:    uuid.Nil,
				Quantity:  1,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "zero quantity",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  0,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "quantity greater than 1",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  2,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "start time after end time",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(48 * time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "start time in the past",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(-1 * time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "duration exceeds maximum 30 days",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(35 * 24 * time.Hour),
				Purpose:   "Valid purpose",
			},
			shouldErr: true,
		},
		{
			name: "empty purpose",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   "   ",
			},
			shouldErr: true,
		},
		{
			name: "purpose too long",
			input: CreateLoanRequestInput{
				ItemID:    itemID,
				Quantity:  1,
				StartTime: now.Add(time.Hour),
				EndTime:   now.Add(24 * time.Hour),
				Purpose:   strings.Repeat("a", 501),
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCreateLoanRequestInput(tt.input, now)
			if tt.shouldErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidateReturnLoanInput(t *testing.T) {
	tests := []struct {
		name      string
		input     ReturnLoanInput
		shouldErr bool
	}{
		{
			name: "valid return condition good",
			input: ReturnLoanInput{
				ReturnCondition: ReturnConditionGood,
			},
			shouldErr: false,
		},
		{
			name: "valid return condition damaged",
			input: ReturnLoanInput{
				ReturnCondition: ReturnConditionDamaged,
			},
			shouldErr: false,
		},
		{
			name: "invalid condition",
			input: ReturnLoanInput{
				ReturnCondition: ReturnCondition("broken"),
			},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReturnLoanInput(tt.input)
			if tt.shouldErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
