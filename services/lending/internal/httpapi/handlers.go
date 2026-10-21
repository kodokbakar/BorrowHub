package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/domain"
	"borrowhub/services/lending/internal/observability"
	"borrowhub/services/lending/internal/service"
	"github.com/google/uuid"
)

type Handlers struct {
	lendingService *service.LendingService
	logger         *slog.Logger
}

func NewHandlers(lendingService *service.LendingService, logger *slog.Logger) *Handlers {
	return &Handlers{
		lendingService: lendingService,
		logger:         logger,
	}
}

// ---------------------------------------------------------------------------
// Availability
// ---------------------------------------------------------------------------

func (h *Handlers) GetItemAvailability(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	itemIDStr := r.PathValue("itemId")
	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid itemId format", nil)
		return
	}

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		orgIDStr = r.Header.Get("X-Organization-ID")
	}
	if orgIDStr == "" {
		WriteError(w, r, http.StatusBadRequest, "MISSING_PARAM", "organization_id query param is required", nil)
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PARAM", "Invalid organization_id format", nil)
		return
	}

	startStr := r.URL.Query().Get("start_time")
	endStr := r.URL.Query().Get("end_time")
	if startStr == "" || endStr == "" {
		WriteError(w, r, http.StatusBadRequest, "MISSING_PARAM", "start_time and end_time are required", nil)
		return
	}

	startTime, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PARAM", "start_time must be RFC3339 format", nil)
		return
	}
	endTime, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PARAM", "end_time must be RFC3339 format", nil)
		return
	}

	avail, err := h.lendingService.GetItemAvailability(r.Context(), orgID, userID, itemID, startTime, endTime)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, avail)
}

// ---------------------------------------------------------------------------
// Loan Requests
// ---------------------------------------------------------------------------

type CreateLoanRequestPayload struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	ItemID         uuid.UUID `json:"item_id"`
	Quantity       int       `json:"quantity"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	Purpose        string    `json:"purpose"`
	Notes          *string   `json:"notes,omitempty"`
}

func (h *Handlers) CreateLoanRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	var payload CreateLoanRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteError(w, r, http.StatusBadRequest, "MALFORMED_JSON", "Invalid request body JSON", nil)
		return
	}

	if payload.OrganizationID == uuid.Nil {
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "organization_id is required", map[string]any{
			"organization_id": "organization_id is required",
		})
		return
	}

	input := domain.CreateLoanRequestInput{
		ItemID:    payload.ItemID,
		Quantity:  payload.Quantity,
		StartTime: payload.StartTime,
		EndTime:   payload.EndTime,
		Purpose:   payload.Purpose,
		Notes:     payload.Notes,
	}

	corrID := CorrelationID(r.Context())
	req, err := h.lendingService.CreateLoanRequest(r.Context(), payload.OrganizationID, userID, input, corrID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusCreated, req)
}

func (h *Handlers) ListLoanRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		orgIDStr = r.Header.Get("X-Organization-ID")
	}
	if orgIDStr == "" {
		WriteError(w, r, http.StatusBadRequest, "MISSING_PARAM", "organization_id query param is required", nil)
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PARAM", "Invalid organization_id format", nil)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var requesterFilter *uuid.UUID
	reqUserStr := r.URL.Query().Get("requester_user_id")
	if reqUserStr != "" {
		uid, err := uuid.Parse(reqUserStr)
		if err == nil {
			requesterFilter = &uid
		}
	}

	var statusFilter *domain.RequestStatus
	statusStr := r.URL.Query().Get("status")
	if statusStr != "" {
		st := domain.RequestStatus(statusStr)
		statusFilter = &st
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if parsed, err := strconv.Atoi(oStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	requests, err := h.lendingService.ListLoanRequests(r.Context(), orgID, userID, member.Role, requesterFilter, statusFilter, limit, offset)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, requests)
}

func (h *Handlers) GetLoanRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	reqIDStr := r.PathValue("requestId")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid requestId format", nil)
		return
	}

	rawReq, err := h.lendingService.GetLoanRequestByID(r.Context(), reqID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawReq.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	req, err := h.lendingService.GetLoanRequest(r.Context(), rawReq.OrganizationID, userID, member.Role, reqID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, req)
}

type ReviewLoanRequestPayload struct {
	ReviewNotes *string `json:"review_notes,omitempty"`
}

func (h *Handlers) ApproveLoanRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	reqIDStr := r.PathValue("requestId")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid requestId format", nil)
		return
	}

	var payload ReviewLoanRequestPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)

	// Fetch request to get organization ID
	rawReq, err := h.lendingService.GetLoanRequestByID(r.Context(), reqID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawReq.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	corrID := CorrelationID(r.Context())
	result, err := h.lendingService.ApproveLoanRequest(
		r.Context(),
		rawReq.OrganizationID,
		userID,
		member.Role,
		reqID,
		payload.ReviewNotes,
		corrID,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	observability.LogAudit(r.Context(), h.logger, "loan.approved",
		"organization_id", rawReq.OrganizationID.String(),
		"reviewer_user_id", userID.String(),
		"loan_request_id", reqID.String(),
	)

	WriteJSON(w, r, http.StatusOK, result)
}

func (h *Handlers) RejectLoanRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	reqIDStr := r.PathValue("requestId")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid requestId format", nil)
		return
	}

	var payload ReviewLoanRequestPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)

	rawReq, err := h.lendingService.GetLoanRequestByID(r.Context(), reqID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawReq.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	corrID := CorrelationID(r.Context())
	result, err := h.lendingService.RejectLoanRequest(
		r.Context(),
		rawReq.OrganizationID,
		userID,
		member.Role,
		reqID,
		payload.ReviewNotes,
		corrID,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	observability.LogAudit(r.Context(), h.logger, "loan.rejected",
		"organization_id", rawReq.OrganizationID.String(),
		"reviewer_user_id", userID.String(),
		"loan_request_id", reqID.String(),
	)

	WriteJSON(w, r, http.StatusOK, result)
}

type CancelLoanRequestPayload struct {
	Reason *string `json:"reason,omitempty"`
}

func (h *Handlers) CancelLoanRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	reqIDStr := r.PathValue("requestId")
	reqID, err := uuid.Parse(reqIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid requestId format", nil)
		return
	}

	var payload CancelLoanRequestPayload
	_ = json.NewDecoder(r.Body).Decode(&payload)

	rawReq, err := h.lendingService.GetLoanRequestByID(r.Context(), reqID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawReq.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	corrID := CorrelationID(r.Context())
	result, err := h.lendingService.CancelLoanRequest(
		r.Context(),
		rawReq.OrganizationID,
		userID,
		member.Role,
		reqID,
		payload.Reason,
		corrID,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Loans
// ---------------------------------------------------------------------------

func (h *Handlers) ListLoans(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	orgIDStr := r.URL.Query().Get("organization_id")
	if orgIDStr == "" {
		orgIDStr = r.Header.Get("X-Organization-ID")
	}
	if orgIDStr == "" {
		WriteError(w, r, http.StatusBadRequest, "MISSING_PARAM", "organization_id query param is required", nil)
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PARAM", "Invalid organization_id format", nil)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	var statusFilter *domain.LoanStatus
	if s := r.URL.Query().Get("status"); s != "" {
		ls := domain.LoanStatus(s)
		statusFilter = &ls
	}

	limit := 50
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if parsed, err := strconv.Atoi(lStr); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if parsed, err := strconv.Atoi(oStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	loans, err := h.lendingService.ListLoans(r.Context(), orgID, userID, member.Role, statusFilter, limit, offset)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, loans)
}

func (h *Handlers) GetLoan(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	loanIDStr := r.PathValue("loanId")
	loanID, err := uuid.Parse(loanIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid loanId format", nil)
		return
	}

	rawLoan, err := h.lendingService.GetLoanByID(r.Context(), loanID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawLoan.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	loan, err := h.lendingService.GetLoan(r.Context(), rawLoan.OrganizationID, userID, member.Role, loanID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, loan)
}

type CheckoutLoanPayload struct {
	Token string `json:"token"`
}

func (h *Handlers) CheckoutLoan(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	loanIDStr := r.PathValue("loanId")
	loanID, err := uuid.Parse(loanIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid loanId format", nil)
		return
	}

	var payload CheckoutLoanPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteError(w, r, http.StatusBadRequest, "MALFORMED_JSON", "Invalid request body JSON", nil)
		return
	}
	if payload.Token == "" {
		WriteError(w, r, http.StatusBadRequest, "VALIDATION_FAILED", "token is required", map[string]any{
			"token": "token is required",
		})
		return
	}

	rawLoan, err := h.lendingService.GetLoanByID(r.Context(), loanID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawLoan.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	corrID := CorrelationID(r.Context())
	loan, err := h.lendingService.CheckoutLoan(
		r.Context(),
		rawLoan.OrganizationID,
		userID,
		member.Role,
		loanID,
		payload.Token,
		corrID,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	observability.LogAudit(r.Context(), h.logger, "loan.checkout",
		"organization_id", rawLoan.OrganizationID.String(),
		"processor_user_id", userID.String(),
		"loan_id", loanID.String(),
	)

	WriteJSON(w, r, http.StatusOK, loan)
}

type ReturnLoanPayload struct {
	ReturnCondition domain.ReturnCondition `json:"return_condition"`
	ReturnNotes     *string                `json:"return_notes,omitempty"`
}

func (h *Handlers) ReturnLoan(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	loanIDStr := r.PathValue("loanId")
	loanID, err := uuid.Parse(loanIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid loanId format", nil)
		return
	}

	var payload ReturnLoanPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		WriteError(w, r, http.StatusBadRequest, "MALFORMED_JSON", "Invalid request body JSON", nil)
		return
	}

	rawLoan, err := h.lendingService.GetLoanByID(r.Context(), loanID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), rawLoan.OrganizationID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	input := domain.ReturnLoanInput{
		ReturnCondition: payload.ReturnCondition,
		ReturnNotes:     payload.ReturnNotes,
	}

	corrID := CorrelationID(r.Context())
	loan, err := h.lendingService.ReturnLoan(
		r.Context(),
		rawLoan.OrganizationID,
		userID,
		member.Role,
		loanID,
		input,
		corrID,
	)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	observability.LogAudit(r.Context(), h.logger, "loan.returned",
		"organization_id", rawLoan.OrganizationID.String(),
		"processor_user_id", userID.String(),
		"loan_id", loanID.String(),
	)

	WriteJSON(w, r, http.StatusOK, loan)
}

func (h *Handlers) ListOverdueLoans(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserID(r.Context())
	if !ok {
		WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "User identity not found", nil)
		return
	}

	orgIDStr := r.PathValue("organizationId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		WriteError(w, r, http.StatusBadRequest, "INVALID_PATH_PARAM", "Invalid organizationId format", nil)
		return
	}

	member, err := h.lendingService.CheckUserMembership(r.Context(), orgID, userID)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	loans, err := h.lendingService.ListOverdueLoans(r.Context(), orgID, userID, member.Role)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, loans)
}

func (h *Handlers) ProcessOverdueInternal(w http.ResponseWriter, r *http.Request) {
	count, err := h.lendingService.ProcessOverdue(r.Context())
	if err != nil {
		HandleError(w, r, err)
		return
	}

	WriteJSON(w, r, http.StatusOK, map[string]any{
		"processed_count": count,
		"processed_at":    time.Now().UTC().Format(time.RFC3339Nano),
	})
}
