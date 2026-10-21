package httpapi

import (
	"net/http"

	"borrowhub/services/inventory/internal/auth"
	"github.com/google/uuid"
)

func AuthMiddleware(tokenManager *auth.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := auth.ExtractToken(r)
			if tokenStr == "" {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Missing authentication token", nil)
				return
			}

			claims, err := tokenManager.ValidateAccessToken(tokenStr)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token", nil)
				return
			}

			uid, err := uuid.Parse(claims.UserID)
			if err != nil {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid user subject in token", nil)
				return
			}

			ctx := auth.ContextWithUser(r.Context(), uid, claims.Email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
