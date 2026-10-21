package httpapi

import (
	"net/http"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/service"
)

type RouterDependencies struct {
	LendingService *service.LendingService
	Repository     IdempotencyRepository
	TokenManager   *auth.JWTManager
	ReadyCheck     func() bool
	InternalToken  string
}

func NewRouter(deps RouterDependencies) http.Handler {
	handlers := NewHandlers(deps.LendingService)
	mux := http.NewServeMux()

	// Health checks
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, r, http.StatusOK, map[string]any{"status": "live"})
	})
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if deps.ReadyCheck != nil && !deps.ReadyCheck() {
			WriteError(w, r, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service not ready", nil)
			return
		}
		WriteJSON(w, r, http.StatusOK, map[string]any{"status": "ready"})
	})

	// Middlewares
	authWrap := AuthMiddleware(deps.TokenManager)
	bodyLimitWrap := BodyLimitMiddleware(1024 * 1024) // 1MB body limit
	idempWrap := IdempotencyMiddleware(deps.Repository)
	internalTokenWrap := InternalTokenMiddleware(deps.InternalToken)

	// Internal endpoint (Scheduler)
	mux.Handle(
		"POST /api/v1/internal/scheduler/process-overdue",
		internalTokenWrap(bodyLimitWrap(http.HandlerFunc(handlers.ProcessOverdueInternal))),
	)

	// Availability check
	mux.Handle(
		"GET /api/v1/items/{itemId}/availability",
		authWrap(http.HandlerFunc(handlers.GetItemAvailability)),
	)

	// Loan requests
	mux.Handle(
		"POST /api/v1/loan-requests",
		authWrap(bodyLimitWrap(idempWrap(http.HandlerFunc(handlers.CreateLoanRequest)))),
	)
	mux.Handle(
		"GET /api/v1/loan-requests",
		authWrap(http.HandlerFunc(handlers.ListLoanRequests)),
	)
	mux.Handle(
		"GET /api/v1/loan-requests/{requestId}",
		authWrap(http.HandlerFunc(handlers.GetLoanRequest)),
	)
	mux.Handle(
		"POST /api/v1/loan-requests/{requestId}/approve",
		authWrap(bodyLimitWrap(idempWrap(http.HandlerFunc(handlers.ApproveLoanRequest)))),
	)
	mux.Handle(
		"POST /api/v1/loan-requests/{requestId}/reject",
		authWrap(bodyLimitWrap(http.HandlerFunc(handlers.RejectLoanRequest))),
	)
	mux.Handle(
		"POST /api/v1/loan-requests/{requestId}/cancel",
		authWrap(bodyLimitWrap(http.HandlerFunc(handlers.CancelLoanRequest))),
	)

	// Loans
	mux.Handle(
		"GET /api/v1/loans",
		authWrap(http.HandlerFunc(handlers.ListLoans)),
	)
	mux.Handle(
		"GET /api/v1/loans/{loanId}",
		authWrap(http.HandlerFunc(handlers.GetLoan)),
	)
	mux.Handle(
		"POST /api/v1/loans/{loanId}/check-out",
		authWrap(bodyLimitWrap(idempWrap(http.HandlerFunc(handlers.CheckoutLoan)))),
	)
	mux.Handle(
		"POST /api/v1/loans/{loanId}/return",
		authWrap(bodyLimitWrap(idempWrap(http.HandlerFunc(handlers.ReturnLoan)))),
	)

	// Overdue loans query
	mux.Handle(
		"GET /api/v1/organizations/{organizationId}/overdue",
		authWrap(http.HandlerFunc(handlers.ListOverdueLoans)),
	)

	return mux
}
