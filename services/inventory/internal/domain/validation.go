package domain

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	slugRegex    = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	barcodeRegex = regexp.MustCompile(`^[A-Z0-9_-]{3,64}$`)
)

func ValidateCategory(name, slug string, description *string) error {
	fields := make(map[string]any)

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		fields["name"] = "name is required"
	} else if len(trimmedName) > 100 {
		fields["name"] = "name too long (max 100 characters)"
	}

	trimmedSlug := strings.TrimSpace(slug)
	if trimmedSlug == "" {
		fields["slug"] = "slug is required"
	} else if !slugRegex.MatchString(trimmedSlug) || len(trimmedSlug) > 100 {
		fields["slug"] = "slug must be lowercase alphanumeric with single hyphens (max 100 characters)"
	}

	if description != nil && len(*description) > 1000 {
		fields["description"] = "description too long (max 1000 characters)"
	}

	if len(fields) > 0 {
		return NewValidationError("invalid category payload", fields)
	}

	return nil
}

func ValidateItem(name string, orgID, categoryID uuid.UUID, description *string) error {
	fields := make(map[string]any)

	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		fields["name"] = "name is required"
	} else if len(trimmedName) > 150 {
		fields["name"] = "name too long (max 150 characters)"
	}

	if orgID == uuid.Nil {
		fields["organization_id"] = "organization_id is required"
	}

	if categoryID == uuid.Nil {
		fields["category_id"] = "category_id is required"
	}

	if description != nil && len(*description) > 2000 {
		fields["description"] = "description too long (max 2000 characters)"
	}

	if len(fields) > 0 {
		return NewValidationError("invalid item payload", fields)
	}

	return nil
}

func NormalizeBarcode(barcode string) string {
	return strings.ToUpper(strings.TrimSpace(barcode))
}

func ValidateUnit(barcode string, serialNumber *string, status UnitStatus, condition UnitCondition, orgID, itemID uuid.UUID, notes *string) (string, *string, error) {
	fields := make(map[string]any)

	if orgID == uuid.Nil {
		fields["organization_id"] = "organization_id is required"
	}

	if itemID == uuid.Nil {
		fields["item_id"] = "item_id is required"
	}

	normalizedBarcode := NormalizeBarcode(barcode)
	if normalizedBarcode == "" {
		fields["barcode"] = "barcode is required"
	} else if !barcodeRegex.MatchString(normalizedBarcode) {
		fields["barcode"] = "barcode must be 3-64 characters alphanumeric, underscores, or hyphens"
	}

	var normSerial *string
	if serialNumber != nil {
		s := strings.TrimSpace(*serialNumber)
		if len(s) > 128 {
			fields["serial_number"] = "serial number too long (max 128 characters)"
		} else if s != "" {
			normSerial = &s
		}
	}

	if !status.IsValid() {
		fields["status"] = "invalid unit status"
	} else if status == StatusBorrowed || status == StatusReserved {
		fields["status"] = "initial status cannot be borrowed or reserved"
	} else if status != StatusAvailable && status != StatusMaintenance {
		fields["status"] = "initial status must be available or maintenance"
	}

	if !condition.IsValid() {
		fields["condition"] = "invalid unit condition"
	}

	if notes != nil && len(*notes) > 2000 {
		fields["notes"] = "notes too long (max 2000 characters)"
	}

	if len(fields) > 0 {
		return "", nil, NewValidationError("invalid unit payload", fields)
	}

	return normalizedBarcode, normSerial, nil
}

type UpdateUnitInput struct {
	Barcode      *string        `json:"barcode"`
	SerialNumber *string        `json:"serial_number"`
	Status       *UnitStatus    `json:"status"`
	Condition    *UnitCondition `json:"condition"`
	Notes        *string        `json:"notes"`
	Reason       *string        `json:"reason"`
}

func ValidateUpdateUnitInput(input UpdateUnitInput) (*string, *string, error) {
	fields := make(map[string]any)

	var normBarcode *string
	if input.Barcode != nil {
		nb := NormalizeBarcode(*input.Barcode)
		if nb == "" {
			fields["barcode"] = "barcode cannot be empty"
		} else if !barcodeRegex.MatchString(nb) {
			fields["barcode"] = "barcode must be 3-64 characters alphanumeric, underscores, or hyphens"
		} else {
			normBarcode = &nb
		}
	}

	var normSerial *string
	if input.SerialNumber != nil {
		s := strings.TrimSpace(*input.SerialNumber)
		if len(s) > 128 {
			fields["serial_number"] = "serial number too long (max 128 characters)"
		} else if s != "" {
			normSerial = &s
		}
	}

	if input.Status != nil && !input.Status.IsValid() {
		fields["status"] = "invalid unit status"
	}

	if input.Condition != nil && !input.Condition.IsValid() {
		fields["condition"] = "invalid unit condition"
	}

	if input.Notes != nil && len(*input.Notes) > 2000 {
		fields["notes"] = "notes too long (max 2000 characters)"
	}

	reason := input.Reason
	if reason != nil && len(*reason) > 500 {
		fields["reason"] = "reason cannot exceed 500 characters"
	}

	if len(fields) > 0 {
		return nil, nil, NewValidationError("invalid unit payload", fields)
	}

	return normBarcode, normSerial, nil
}
