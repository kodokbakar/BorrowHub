package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound            = errors.New("resource not found")
	ErrLoanRequestNotFound = fmt.Errorf("%w: loan request not found", ErrNotFound)
	ErrLoanNotFound        = fmt.Errorf("%w: loan not found", ErrNotFound)
	ErrReservationNotFound = fmt.Errorf("%w: reservation not found", ErrNotFound)
	ErrTokenNotFound       = fmt.Errorf("%w: checkout token not found", ErrNotFound)

	ErrConflict            = errors.New("conflict")
	ErrReservationConflict = fmt.Errorf("%w: unit is already reserved for overlapping time range", ErrConflict)
	ErrIdempotencyConflict = fmt.Errorf("%w: conflicting request with same idempotency key", ErrConflict)
	ErrNoUnitsAvailable    = fmt.Errorf("%w: no units available for requested time period", ErrConflict)

	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden: insufficient permissions")
	ErrUnauthorizedAction = fmt.Errorf("%w: user not authorized to perform action", ErrForbidden)
	ErrInvalidTransition  = errors.New("invalid status transition")

	ErrInvalidToken     = errors.New("invalid checkout token")
	ErrTokenExpired     = errors.New("checkout token has expired")
	ErrTokenAlreadyUsed = errors.New("checkout token has already been used")
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
