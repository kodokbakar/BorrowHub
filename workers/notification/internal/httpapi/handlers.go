package httpapi

import (
	"net/http"
	"strconv"

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/domain"
	"borrowhub/workers/notification/internal/service"
	"github.com/google/uuid"
)

type Handlers struct {
	svc        *service.NotificationService
	readyCheck func() bool
}

func NewHandlers(svc *service.NotificationService, readyCheck func() bool) *Handlers {
	if readyCheck == nil {
		readyCheck = func() bool { return true }
	}
	return &Handlers{
		svc:        svc,
		readyCheck: readyCheck,
	}
}

func (h *Handlers) HandleHealthLive(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]string{
		"status": "live",
	}, nil)
}

func (h *Handlers) HandleHealthReady(w http.ResponseWriter, r *http.Request) {
	if !h.readyCheck() {
		WriteError(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service dependencies are not ready", nil)
		return
	}
	WriteJSON(w, r, http.StatusOK, map[string]string{
		"status": "ready",
	}, nil)
}

func (h *Handlers) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity required", nil)
		return
	}

	// Parse query parameters
	var statusFilter *domain.NotificationStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := domain.NotificationStatus(s)
		statusFilter = &st
	}

	limit := 20
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}

	offset := 0
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	items, total, err := h.svc.ListNotifications(r.Context(), orgID, userID, statusFilter, limit, offset)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	meta := &Meta{
		Total:  &total,
		Limit:  &limit,
		Offset: &offset,
	}

	WriteJSON(w, r, http.StatusOK, items, meta)
}

func (h *Handlers) HandleMarkAsRead(w http.ResponseWriter, r *http.Request) {
	notifID, err := parseUUID("notificationId", r.PathValue("notificationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity required", nil)
		return
	}

	var orgID uuid.UUID
	if orgStr := r.URL.Query().Get("organization_id"); orgStr != "" {
		orgID, _ = uuid.Parse(orgStr)
	}

	updated, err := h.svc.MarkAsRead(r.Context(), orgID, notifID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, updated, nil)
}
