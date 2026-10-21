package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager(t *testing.T) {
	manager := NewJWTManager("super-secret-key-that-is-at-least-32-bytes-long!")
	userID := uuid.New().String()
	email := "user@example.com"

	tokenStr, err := manager.GenerateAccessToken(userID, email, time.Hour)
	if err != nil {
		t.Fatalf("failed generating token: %v", err)
	}

	claims, err := manager.ValidateAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("failed validating token: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("expected userID %s, got %s", userID, claims.UserID)
	}
	if claims.Email != email {
		t.Fatalf("expected email %s, got %s", email, claims.Email)
	}

	// Test invalid token
	_, err = manager.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token, got nil")
	}
}

func TestHTTPIdentityClient(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Token") != "secret-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		expectedPath := "/api/v1/internal/organizations/" + orgID.String() + "/members/" + userID.String()
		if r.URL.Path != expectedPath {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]any{
			"data": MemberInfo{
				IsMember:       true,
				Role:           "admin",
				OrganizationID: orgID,
				UserID:         userID,
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewHTTPIdentityClient(server.URL, "secret-token", time.Minute)

	member, err := client.GetMember(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("failed getting member: %v", err)
	}

	if !member.IsMember || member.Role != "admin" {
		t.Fatalf("unexpected member: %+v", member)
	}

	// Second call should hit in-memory cache
	member2, err := client.GetMember(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("failed getting member from cache: %v", err)
	}
	if member2.Role != "admin" {
		t.Fatalf("unexpected cached member: %+v", member2)
	}
}
