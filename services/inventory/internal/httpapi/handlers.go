package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"

	"borrowhub/services/inventory/internal/auth"
	"borrowhub/services/inventory/internal/domain"
	"borrowhub/services/inventory/internal/service"
	"github.com/google/uuid"
)

type RouterDependencies struct {
	CategoryService *service.CategoryService
	ItemService     *service.ItemService
	UnitService     *service.UnitService
	InternalService *service.InternalInventoryService
	TokenManager    *auth.JWTManager
	ReadyCheck      func() bool
	InternalToken   string
}

type handler struct {
	deps RouterDependencies
}

func NewRouter(deps RouterDependencies) http.Handler {
	h := &handler{deps: deps}
	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /health/live", h.handleHealthLive)
	mux.HandleFunc("GET /health/ready", h.handleHealthReady)

	// Internal endpoints (Lending)
	mux.HandleFunc("GET /api/v1/internal/items/{itemId}/eligible-units", h.handleInternalEligibleUnits)
	mux.HandleFunc("PATCH /api/v1/internal/units/{unitId}/status", h.handleInternalUnitStatus)

	// Protected user endpoints
	authWrap := AuthMiddleware(deps.TokenManager)

	// Categories
	mux.Handle("GET /api/v1/organizations/{organizationId}/categories", authWrap(http.HandlerFunc(h.handleListCategories)))
	mux.Handle("POST /api/v1/organizations/{organizationId}/categories", authWrap(http.HandlerFunc(h.handleCreateCategory)))
	mux.Handle("PATCH /api/v1/categories/{categoryId}", authWrap(http.HandlerFunc(h.handleUpdateCategory)))
	mux.Handle("DELETE /api/v1/categories/{categoryId}", authWrap(http.HandlerFunc(h.handleDeleteCategory)))

	// Items
	mux.Handle("GET /api/v1/organizations/{organizationId}/items", authWrap(http.HandlerFunc(h.handleListItems)))
	mux.Handle("POST /api/v1/organizations/{organizationId}/items", authWrap(http.HandlerFunc(h.handleCreateItem)))
	mux.Handle("GET /api/v1/items/{itemId}", authWrap(http.HandlerFunc(h.handleGetItem)))
	mux.Handle("PATCH /api/v1/items/{itemId}", authWrap(http.HandlerFunc(h.handleUpdateItem)))
	mux.Handle("DELETE /api/v1/items/{itemId}", authWrap(http.HandlerFunc(h.handleDeleteItem)))

	// Units
	mux.Handle("POST /api/v1/items/{itemId}/units", authWrap(http.HandlerFunc(h.handleCreateUnit)))
	mux.Handle("GET /api/v1/items/{itemId}/units", authWrap(http.HandlerFunc(h.handleListUnits)))
	mux.Handle("GET /api/v1/units/{unitId}", authWrap(http.HandlerFunc(h.handleGetUnit)))
	mux.Handle("PATCH /api/v1/units/{unitId}", authWrap(http.HandlerFunc(h.handleUpdateUnit)))
	mux.Handle("GET /api/v1/units/{unitId}/condition-logs", authWrap(http.HandlerFunc(h.handleGetConditionLogs)))

	return mux
}

// Health handlers

func (h *handler) handleHealthLive(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *handler) handleHealthReady(w http.ResponseWriter, r *http.Request) {
	if h.deps.ReadyCheck != nil && !h.deps.ReadyCheck() {
		WriteError(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "service not ready", nil)
		return
	}
	WriteJSON(w, r, http.StatusOK, map[string]string{"status": "ok"})
}

// Categories handlers

type createCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
}

func (h *handler) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req createCategoryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	cat, err := h.deps.CategoryService.Create(r.Context(), orgID, userID, req.Name, req.Slug, req.Description)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, cat)
}

func (h *handler) handleListCategories(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	categories, err := h.deps.CategoryService.ListByOrg(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, categories)
}

type updateCategoryRequest struct {
	Name        *string `json:"name"`
	Slug        *string `json:"slug"`
	Description *string `json:"description"`
}

func (h *handler) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	categoryID, err := parseUUID("categoryId", r.PathValue("categoryId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req updateCategoryRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	cat, err := h.deps.CategoryService.Update(r.Context(), categoryID, userID, req.Name, req.Slug, req.Description)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, cat)
}

func (h *handler) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	categoryID, err := parseUUID("categoryId", r.PathValue("categoryId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	if err := h.deps.CategoryService.Delete(r.Context(), categoryID, userID); err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"message": "category deleted successfully",
	})
}

// Items handlers

type createItemRequest struct {
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	IsLendable  *bool     `json:"is_lendable"`
}

func (h *handler) handleCreateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req createItemRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	isLendable := true
	if req.IsLendable != nil {
		isLendable = *req.IsLendable
	}

	item, err := h.deps.ItemService.Create(
		r.Context(),
		orgID,
		userID,
		req.CategoryID,
		req.Name,
		req.Description,
		isLendable,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, item)
}

func (h *handler) handleListItems(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	orgID, err := parseUUID("organizationId", r.PathValue("organizationId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	items, err := h.deps.ItemService.ListByOrg(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, items)
}

func (h *handler) handleGetItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	item, err := h.deps.ItemService.GetByID(r.Context(), itemID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, item)
}

type updateItemRequest struct {
	CategoryID  *uuid.UUID `json:"category_id"`
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	IsLendable  *bool      `json:"is_lendable"`
}

func (h *handler) handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req updateItemRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	item, err := h.deps.ItemService.Update(
		r.Context(),
		itemID,
		userID,
		req.CategoryID,
		req.Name,
		req.Description,
		req.IsLendable,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, item)
}

func (h *handler) handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	if err := h.deps.ItemService.Delete(r.Context(), itemID, userID); err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]string{
		"message": "item deleted successfully",
	})
}

// Units handlers

type createUnitRequest struct {
	Barcode      string                `json:"barcode"`
	SerialNumber *string               `json:"serial_number"`
	Status       *domain.UnitStatus    `json:"status"`
	Condition    *domain.UnitCondition `json:"condition"`
	Notes        *string               `json:"notes"`
}

func (h *handler) handleCreateUnit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req createUnitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	unit, err := h.deps.UnitService.Create(
		r.Context(),
		itemID,
		userID,
		req.Barcode,
		req.SerialNumber,
		req.Status,
		req.Condition,
		req.Notes,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, unit)
}

func (h *handler) handleListUnits(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	units, err := h.deps.UnitService.ListByItem(r.Context(), itemID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, units)
}

func (h *handler) handleGetUnit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	unitID, err := parseUUID("unitId", r.PathValue("unitId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	unit, err := h.deps.UnitService.GetByID(r.Context(), unitID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, unit)
}

type updateUnitRequest struct {
	Barcode      *string               `json:"barcode"`
	SerialNumber *string               `json:"serial_number"`
	Status       *domain.UnitStatus    `json:"status"`
	Condition    *domain.UnitCondition `json:"condition"`
	Notes        *string               `json:"notes"`
	Reason       *string               `json:"reason"`
}

func (h *handler) handleUpdateUnit(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	unitID, err := parseUUID("unitId", r.PathValue("unitId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req updateUnitRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	unit, err := h.deps.UnitService.Update(
		r.Context(),
		unitID,
		userID,
		req.Barcode,
		req.SerialNumber,
		req.Status,
		req.Condition,
		req.Notes,
		req.Reason,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, unit)
}

func (h *handler) handleGetConditionLogs(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		HandleError(w, r, domain.ErrUnauthorized)
		return
	}
	unitID, err := parseUUID("unitId", r.PathValue("unitId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	logs, err := h.deps.UnitService.GetConditionLogs(r.Context(), unitID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, logs)
}

// Internal endpoints (Lending)

func (h *handler) checkInternalToken(w http.ResponseWriter, r *http.Request) bool {
	if h.deps.InternalToken == "" {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid internal token", nil)
		return false
	}
	token := r.Header.Get("X-Internal-Token")
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(h.deps.InternalToken)) != 1 {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing or invalid internal token", nil)
		return false
	}
	return true
}

func (h *handler) handleInternalEligibleUnits(w http.ResponseWriter, r *http.Request) {
	if !h.checkInternalToken(w, r) {
		return
	}

	itemID, err := parseUUID("itemId", r.PathValue("itemId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	units, err := h.deps.InternalService.ListEligibleUnits(r.Context(), itemID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, units)
}

type internalUnitStatusRequest struct {
	FromStatus domain.UnitStatus `json:"from_status"`
	ToStatus   domain.UnitStatus `json:"to_status"`
}

func (h *handler) handleInternalUnitStatus(w http.ResponseWriter, r *http.Request) {
	if !h.checkInternalToken(w, r) {
		return
	}

	unitID, err := parseUUID("unitId", r.PathValue("unitId"))
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var req internalUnitStatusRequest
	if err := decodeJSON(w, r, &req); err != nil {
		HandleError(w, r, err)
		return
	}

	unit, err := h.deps.InternalService.TransitionUnitStatus(
		r.Context(),
		unitID,
		req.FromStatus,
		req.ToStatus,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, unit)
}

// Helpers

func parseUUID(paramName, raw string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, domain.NewValidationError("invalid identifier", map[string]any{
			paramName: "must be a valid UUID",
		})
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if r.Body == nil {
		return domain.NewValidationError("empty request body", nil)
	}
	defer r.Body.Close()

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return domain.NewValidationError("malformed or oversized json payload: "+err.Error(), nil)
	}

	if dec.Decode(&struct{}{}) != io.EOF {
		return domain.NewValidationError("unexpected trailing content", nil)
	}

	return nil
}
