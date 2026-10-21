package httpapi

import (
	"errors"
	"net/http"

	"borrowhub/services/inventory/internal/domain"
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

	if errors.Is(err, domain.ErrUnauthorized) {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required", nil)
		return
	}

	if errors.Is(err, domain.ErrForbidden) {
		WriteError(w, r, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrNotFound) ||
		errors.Is(err, domain.ErrCategoryNotFound) ||
		errors.Is(err, domain.ErrItemNotFound) ||
		errors.Is(err, domain.ErrUnitNotFound) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrStatusMismatch) {
		WriteError(w, r, http.StatusConflict, "STATUS_MISMATCH", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrConflict) ||
		errors.Is(err, domain.ErrDuplicateBarcode) ||
		errors.Is(err, domain.ErrDuplicateSerial) ||
		errors.Is(err, domain.ErrDuplicateSlug) ||
		errors.Is(err, domain.ErrCategoryInUse) ||
		errors.Is(err, domain.ErrItemInUse) {
		WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error(), nil)
		return
	}

	if errors.Is(err, domain.ErrInvalidTransition) {
		WriteError(w, r, http.StatusBadRequest, "INVALID_TRANSITION", err.Error(), nil)
		return
	}

	WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error", nil)
}
