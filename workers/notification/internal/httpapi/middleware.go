package httpapi

import (
	"crypto/subtle"
	"net/http"

	"borrowhub/workers/notification/internal/auth"
	"borrowhub/workers/notification/internal/domain"
	"github.com/google/uuid"
)

func AuthMiddleware(tokenManager *auth.JWTManager, internalToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check internal token bypass (for inter-service calls)
			if internalToken != "" {
				clientInternalToken := r.Header.Get("X-Internal-Token")
				if clientInternalToken != "" && subtle.ConstantTimeCompare([]byte(clientInternalToken), []byte(internalToken)) == 1 {
					targetUserHeader := r.Header.Get("X-User-ID")
					var targetUID uuid.UUID
					if targetUserHeader != "" {
						targetUID, _ = uuid.Parse(targetUserHeader)
					}
					ctx := auth.ContextWithUser(r.Context(), targetUID, "")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// 2. Check JWT Bearer token
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

func parseUUID(field, val string) (uuid.UUID, error) {
	id, err := uuid.Parse(val)
	if err != nil {
		return uuid.Nil, domain.NewValidationError("invalid UUID format", map[string]any{
			field: "must be a valid UUID",
		})
	}
	return id, nil
}
