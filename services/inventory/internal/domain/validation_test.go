package domain

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestValidateCategory(t *testing.T) {
	// Valid
	if err := ValidateCategory("Cameras", "cameras", nil); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
	if err := ValidateCategory("DSLR Cameras", "dslr-cameras-101", nil); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}

	// Invalid empty name
	if err := ValidateCategory("", "cameras", nil); err == nil {
		t.Fatal("expected error for empty name")
	}

	// Invalid long name
	if err := ValidateCategory(strings.Repeat("a", 101), "cameras", nil); err == nil {
		t.Fatal("expected error for long name")
	}

	// Invalid slug formats
	invalidSlugs := []string{"", "Cameras", "camera_dslr", "-camera", "camera-", "camera--dslr", "kamera dslr"}
	for _, s := range invalidSlugs {
		if err := ValidateCategory("Cameras", s, nil); err == nil {
			t.Fatalf("expected error for invalid slug %q", s)
		}
	}
}

func TestValidateItem(t *testing.T) {
	orgID := uuid.New()
	catID := uuid.New()

	// Valid
	if err := ValidateItem("Sony A7 III", orgID, catID, nil); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}

	// Empty name
	if err := ValidateItem("", orgID, catID, nil); err == nil {
		t.Fatal("expected error for empty name")
	}

	// Missing org
	if err := ValidateItem("Sony A7 III", uuid.Nil, catID, nil); err == nil {
		t.Fatal("expected error for nil org")
	}

	// Missing category
	if err := ValidateItem("Sony A7 III", orgID, uuid.Nil, nil); err == nil {
		t.Fatal("expected error for nil category")
	}

	// Long name
	if err := ValidateItem(strings.Repeat("x", 151), orgID, catID, nil); err == nil {
		t.Fatal("expected error for long name")
	}
}

func TestValidateUnit(t *testing.T) {
	orgID := uuid.New()
	itemID := uuid.New()

	// Valid with normalization
	barcode, serial, err := ValidateUnit("  cam-001  ", nil, StatusAvailable, ConditionNew, orgID, itemID, nil)
	if err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
	if barcode != "CAM-001" {
		t.Fatalf("expected normalized barcode 'CAM-001', got %q", barcode)
	}
	if serial != nil {
		t.Fatalf("expected nil serial, got %v", serial)
	}

	// Valid with serial
	serialInput := " SN123456 "
	barcode, serial, err = ValidateUnit("CAM-002", &serialInput, StatusAvailable, ConditionGood, orgID, itemID, nil)
	if err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}
	if serial == nil || *serial != "SN123456" {
		t.Fatalf("expected trimmed serial 'SN123456', got %v", serial)
	}

	// Invalid barcode characters
	invalidBarcodes := []string{"", "A", "AB", "BC@01", "CAM 001", "CAM#1"}
	for _, bc := range invalidBarcodes {
		if _, _, err := ValidateUnit(bc, nil, StatusAvailable, ConditionNew, orgID, itemID, nil); err == nil {
			t.Fatalf("expected error for invalid barcode %q", bc)
		}
	}

	// Invalid status
	if _, _, err := ValidateUnit("CAM-003", nil, UnitStatus("invalid"), ConditionNew, orgID, itemID, nil); err == nil {
		t.Fatal("expected error for invalid status")
	}

	// Invalid condition
	if _, _, err := ValidateUnit("CAM-003", nil, StatusAvailable, UnitCondition("invalid"), orgID, itemID, nil); err == nil {
		t.Fatal("expected error for invalid condition")
	}

	// Initial status cannot be borrowed or reserved
	for _, st := range []UnitStatus{StatusBorrowed, StatusReserved} {
		if _, _, err := ValidateUnit("CAM-003", nil, st, ConditionNew, orgID, itemID, nil); err == nil {
			t.Fatalf("expected error for initial status %s", st)
		} else {
			var valErr *ValidationError
			if !errors.As(err, &valErr) || valErr.Fields["status"] != "initial status cannot be borrowed or reserved" {
				t.Fatalf("expected 'initial status cannot be borrowed or reserved', got: %v", err)
			}
		}
	}

	// Initial status maintenance is allowed
	if _, _, err := ValidateUnit("CAM-004", nil, StatusMaintenance, ConditionNew, orgID, itemID, nil); err != nil {
		t.Fatalf("expected maintenance allowed as initial status, got: %v", err)
	}
}

func TestValidateUpdateUnitInput(t *testing.T) {
	// Valid update
	newBc := " cam-update "
	newSerial := " sn-update "
	st := StatusMaintenance
	cond := ConditionFair
	notes := "updated notes"
	reason := "scheduled checkup"
	input := UpdateUnitInput{
		Barcode:      &newBc,
		SerialNumber: &newSerial,
		Status:       &st,
		Condition:    &cond,
		Notes:        &notes,
		Reason:       &reason,
	}
	normBc, normSerial, err := ValidateUpdateUnitInput(input)
	if err != nil {
		t.Fatalf("expected valid update input, got: %v", err)
	}
	if normBc == nil || *normBc != "CAM-UPDATE" {
		t.Fatalf("expected normalized barcode 'CAM-UPDATE', got %v", normBc)
	}
	if normSerial == nil || *normSerial != "sn-update" {
		t.Fatalf("expected trimmed serial 'sn-update', got %v", normSerial)
	}

	// Reason too long (> 500 chars)
	longReason := strings.Repeat("r", 501)
	invalidInput := UpdateUnitInput{
		Reason: &longReason,
	}
	if _, _, err := ValidateUpdateUnitInput(invalidInput); err == nil {
		t.Fatal("expected error for reason exceeding 500 characters")
	} else {
		var valErr *ValidationError
		if !errors.As(err, &valErr) || valErr.Fields["reason"] != "reason cannot exceed 500 characters" {
			t.Fatalf("expected reason length validation error, got: %v", err)
		}
	}

	// Invalid status / condition
	badStatus := UnitStatus("invalid")
	badCond := UnitCondition("invalid")
	if _, _, err := ValidateUpdateUnitInput(UpdateUnitInput{Status: &badStatus, Condition: &badCond}); err == nil {
		t.Fatal("expected error for invalid status and condition in update input")
	}
}
