package httpapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"borrowhub/services/lending/internal/auth"
	"borrowhub/services/lending/internal/domain"
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

func InternalTokenMiddleware(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Internal-Token")
			if token == "" || token != expectedToken {
				WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or missing internal service token", nil)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func BodyLimitMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
			next.ServeHTTP(w, r)
		})
	}
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	if rec.statusCode == 0 {
		rec.statusCode = http.StatusOK
	}
	return rec.body.Write(b)
}

type IdempotencyRepository interface {
	GetIdempotencyKey(ctx context.Context, orgID uuid.UUID, key string) (*domain.IdempotencyRecord, error)
	SaveIdempotencyKey(ctx context.Context, record *domain.IdempotencyRecord) error
}

func IdempotencyMiddleware(repo IdempotencyRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("Idempotency-Key")
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				WriteError(w, r, http.StatusBadRequest, "BAD_REQUEST", "Failed reading request body", nil)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			hash := sha256.Sum256(bodyBytes)
			reqHash := hex.EncodeToString(hash[:])

			userID, _ := auth.UserID(r.Context())

			// We need orgID to check key. Extract from header, query, URL path, or body:
			orgIDStr := r.Header.Get("X-Organization-ID")
			if orgIDStr == "" {
				orgIDStr = r.URL.Query().Get("organization_id")
			}
			if orgIDStr == "" {
				orgIDStr = r.PathValue("organizationId")
			}
			if orgIDStr == "" {
				orgIDStr = r.PathValue("organization_id")
			}
			if orgIDStr == "" {
				orgIDStr = r.PathValue("orgId")
			}
			var orgID uuid.UUID
			if orgIDStr != "" {
				orgID, _ = uuid.Parse(orgIDStr)
			}

			if orgID == uuid.Nil && len(bodyBytes) > 0 {
				var bodyObj struct {
					OrganizationID *uuid.UUID `json:"organization_id"`
				}
				if err := json.Unmarshal(bodyBytes, &bodyObj); err == nil && bodyObj.OrganizationID != nil {
					orgID = *bodyObj.OrganizationID
				}
			}
			// Check DB
			rec, err := repo.GetIdempotencyKey(r.Context(), orgID, key)
			if err != nil {
				HandleError(w, r, err)
				return
			}
			if rec != nil {
				if rec.RequestHash != reqHash || rec.RequestPath != r.URL.Path {
					WriteError(w, r, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Conflicting request for idempotency key", nil)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(rec.ResponseCode)
				_, _ = w.Write(rec.ResponseBody)
				return
			}

			// Capture response
			recorder := &responseRecorder{
				ResponseWriter: w,
			}
			next.ServeHTTP(recorder, r)

			if recorder.statusCode >= 200 && recorder.statusCode < 300 {
				_ = repo.SaveIdempotencyKey(r.Context(), &domain.IdempotencyRecord{
					Key:            key,
					OrganizationID: orgID,
					UserID:         userID,
					RequestPath:    r.URL.Path,
					RequestHash:    reqHash,
					ResponseCode:   recorder.statusCode,
					ResponseBody:   recorder.body.Bytes(),
					CreatedAt:      time.Now().UTC(),
					ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
				})
			}

			// Write recorder to actual response writer
			for k, v := range recorder.Header() {
				w.Header()[k] = v
			}
			if recorder.statusCode > 0 {
				w.WriteHeader(recorder.statusCode)
			}
			_, _ = w.Write(recorder.body.Bytes())
		})
	}
}
