package httpapi

import (
	"net/http"

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/service"
)

type RouterDependencies struct {
	Service       *service.NotificationService
	TokenManager  *auth.JWTManager
	InternalToken string
	ReadyCheck    func() bool
}

func NewRouter(deps RouterDependencies) http.Handler {
	mux := http.NewServeMux()
	handlers := NewHandlers(deps.Service, deps.ReadyCheck)

	// Health check endpoints (public)
	mux.HandleFunc("GET /health/live", handlers.HandleHealthLive)
	mux.HandleFunc("GET /health/ready", handlers.HandleHealthReady)

	// Protected notification query endpoints
	authWrap := AuthMiddleware(deps.TokenManager, deps.InternalToken)

	mux.Handle(
		"GET /api/v1/organizations/{organizationId}/notifications",
		authWrap(http.HandlerFunc(handlers.HandleListNotifications)),
	)
	mux.Handle(
		"PATCH /api/v1/notifications/{notificationId}/read",
		authWrap(http.HandlerFunc(handlers.HandleMarkAsRead)),
	)

	// Fallback 404
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		WriteError(w, r, http.StatusNotFound, "RESOURCE_NOT_FOUND", "route not found", nil)
	})

	return RequestIDs(mux)
}
