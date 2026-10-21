package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound               = errors.New("resource not found")
	ErrNotificationNotFound   = fmt.Errorf("%w: notification not found", ErrNotFound)
	ErrInvalidStateTransition = errors.New("invalid state transition")
	ErrAlreadyProcessed       = errors.New("event already processed")
	ErrMalformedEvent         = errors.New("malformed event")
	ErrUnauthorized           = errors.New("unauthorized")
	ErrForbidden              = errors.New("forbidden")
)

type ValidationError struct {
	Message string         `json:"message"`
	Fields  map[string]any `json:"fields,omitempty"`
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
