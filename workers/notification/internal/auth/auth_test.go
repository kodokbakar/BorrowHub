package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager(t *testing.T) {
	mgr := NewJWTManager("test-secret-key-at-least-32-bytes-long")
	userID := uuid.New().String()
	email := "user@example.com"

	// 1. Generate & Validate valid token
	token, err := mgr.GenerateAccessToken(userID, email, time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("validate token: %v", err)
	}
	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// 2. Expired token
	expToken, err := mgr.GenerateAccessToken(userID, email, -time.Minute)
	if err != nil {
		t.Fatalf("generate expired token: %v", err)
	}
	_, err = mgr.ValidateAccessToken(expToken)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for expired token, got: %v", err)
	}

	// 3. Invalid signature
	otherMgr := NewJWTManager("other-secret-key-at-least-32-bytes-long")
	_, err = otherMgr.ValidateAccessToken(token)
	if err != ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken for mismatched secret, got: %v", err)
	}

	// 4. Extract token from HTTP header
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	extracted := ExtractToken(req)
	if extracted != token {
		t.Fatalf("expected extracted %s, got %s", token, extracted)
	}
}
