package httpapi

import (
	"errors"
	"net/http"

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/domain"
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

	if errors.Is(err, domain.ErrForbidden) {
		WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrNotificationNotFound) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", err.Error(), nil)
		return
	}

	WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
}
