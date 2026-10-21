package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type userContextKey uint8

const (
	userIDKey userContextKey = iota
	userEmailKey
)

func UserID(ctx context.Context) (uuid.UUID, bool) {
	val, ok := ctx.Value(userIDKey).(uuid.UUID)
	return val, ok
}

func UserEmail(ctx context.Context) (string, bool) {
	val, ok := ctx.Value(userEmailKey).(string)
	return val, ok
}

func ContextWithUser(ctx context.Context, userID uuid.UUID, email string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	return context.WithValue(ctx, userEmailKey, email)
}

func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if cookie, err := r.Cookie("access_token"); err == nil {
		return cookie.Value
	}

	return ""
}
