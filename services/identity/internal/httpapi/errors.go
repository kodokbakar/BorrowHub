package httpapi

import (
	"errors"
	"net/http"

	"borrowhub/services/identity/internal/domain"
)

func HandleError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	var valErr *domain.ValidationError
	if errors.As(err, &valErr) {
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_FAILED", valErr.Message, valErr.Fields)
		return
	}

	if errors.Is(err, domain.ErrInvalidCredentials) {
		WriteError(w, r, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password", nil)
		return
	}

	if errors.Is(err, domain.ErrUnauthorized) {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	if errors.Is(err, domain.ErrForbidden) ||
		errors.Is(err, domain.ErrCannotDemoteSoleOwner) ||
		errors.Is(err, domain.ErrCannotRemoveSoleOwner) {
		WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrUserNotFound) ||
		errors.Is(err, domain.ErrOrgNotFound) ||
		errors.Is(err, domain.ErrMembershipNotFound) ||
		errors.Is(err, domain.ErrInvitationNotFound) ||
		errors.Is(err, domain.ErrSessionNotFound) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrConflict) ||
		errors.Is(err, domain.ErrEmailAlreadyExists) ||
		errors.Is(err, domain.ErrSlugAlreadyExists) ||
		errors.Is(err, domain.ErrAlreadyMember) ||
		errors.Is(err, domain.ErrInvitationAlreadyUsed) {
		WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrInvitationExpired) {
		WriteError(w, r, http.StatusGone, "INVITATION_EXPIRED", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrTermsNotAccepted) {
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), map[string]any{
			"terms_accepted": "must be true",
		})
		return
	}

	WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
}
