package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound           = errors.New("resource not found")
	ErrUserNotFound       = fmt.Errorf("%w: user not found", ErrNotFound)
	ErrOrgNotFound        = fmt.Errorf("%w: organization not found", ErrNotFound)
	ErrMembershipNotFound = fmt.Errorf("%w: membership not found", ErrNotFound)
	ErrInvitationNotFound = fmt.Errorf("%w: invitation not found", ErrNotFound)
	ErrSessionNotFound    = fmt.Errorf("%w: session not found", ErrNotFound)

	ErrConflict           = errors.New("conflict")
	ErrEmailAlreadyExists = fmt.Errorf("%w: email already exists", ErrConflict)
	ErrSlugAlreadyExists  = fmt.Errorf("%w: organization slug already exists", ErrConflict)
	ErrAlreadyMember      = fmt.Errorf("%w: user is already a member", ErrConflict)

	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrUnauthorized          = errors.New("unauthorized")
	ErrForbidden             = errors.New("forbidden: insufficient permissions")
	ErrCannotDemoteSoleOwner = errors.New("cannot demote the sole owner of an organization")
	ErrCannotRemoveSoleOwner = errors.New("cannot remove the sole owner of an organization")

	ErrInvitationExpired     = errors.New("invitation has expired")
	ErrInvitationAlreadyUsed = errors.New("invitation has already been accepted or revoked")
	ErrTermsNotAccepted      = errors.New("terms of service must be accepted")
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
