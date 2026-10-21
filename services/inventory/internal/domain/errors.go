package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound         = errors.New("resource not found")
	ErrCategoryNotFound = fmt.Errorf("%w: category not found", ErrNotFound)
	ErrItemNotFound     = fmt.Errorf("%w: item not found", ErrNotFound)
	ErrUnitNotFound     = fmt.Errorf("%w: unit not found", ErrNotFound)

	ErrConflict         = errors.New("conflict")
	ErrDuplicateBarcode = fmt.Errorf("%w: barcode already exists in organization", ErrConflict)
	ErrDuplicateSerial  = fmt.Errorf("%w: serial number already exists in organization", ErrConflict)
	ErrDuplicateSlug    = fmt.Errorf("%w: category slug already exists in organization", ErrConflict)
	ErrCategoryInUse    = fmt.Errorf("%w: category has associated items and cannot be deleted", ErrConflict)
	ErrItemInUse        = fmt.Errorf("%w: item has associated units and cannot be deleted", ErrConflict)

	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden: insufficient permissions")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrStatusMismatch    = fmt.Errorf("%w: unit status mismatch", ErrConflict)
)

type ValidationError struct {
	Message string
	Fields  map[string]any
}

func (e *ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string, fields map[string]any) *ValidationError {
	return &ValidationError{
		Message: message,
		Fields:  fields,
	}
}
