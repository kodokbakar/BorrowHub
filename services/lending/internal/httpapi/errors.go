package httpapi

import (
	"errors"
	"net/http"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/domain"
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

	if errors.Is(err, auth.ErrInvalidToken) || errors.Is(err, domain.ErrUnauthorized) {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required or invalid token", nil)
		return
	}

	if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrUnauthorizedAction) {
		WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrLoanRequestNotFound) ||
		errors.Is(err, domain.ErrLoanNotFound) ||
		errors.Is(err, domain.ErrReservationNotFound) ||
		errors.Is(err, domain.ErrTokenNotFound) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrReservationConflict) ||
		errors.Is(err, domain.ErrNoUnitsAvailable) ||
		errors.Is(err, domain.ErrIdempotencyConflict) ||
		errors.Is(err, domain.ErrConflict) {
		WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrInvalidTransition) {
		WriteError(w, r, http.StatusBadRequest, "INVALID_TRANSITION", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrInvalidToken) ||
		errors.Is(err, domain.ErrTokenExpired) ||
		errors.Is(err, domain.ErrTokenAlreadyUsed) {
		WriteError(w, r, http.StatusBadRequest, "INVALID_CHECKOUT_TOKEN", err.Error(), nil)
		return
	}

	WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
}
