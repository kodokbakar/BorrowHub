package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager(t *testing.T) {
	mgr := NewJWTManager("super-secret-key-that-is-long-enough-32-bytes")
	userID := uuid.New().String()
	email := "user@example.com"

	token, err := mgr.GenerateAccessToken(userID, email, time.Minute)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected error validating token: %v", err)
	}
	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("unexpected claims: %+v", claims)
	}

	// Invalid signature
	otherMgr := NewJWTManager("another-super-secret-key-that-is-long")
	if _, err := otherMgr.ValidateAccessToken(token); err == nil {
		t.Fatal("expected error with different secret")
	}

	// Expired token
	expiredToken, err := mgr.GenerateAccessToken(userID, email, -time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := mgr.ValidateAccessToken(expiredToken); err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestHTTPIdentityClient(t *testing.T) {
	var callCount atomic.Int32
	orgID := uuid.New()
	userID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		token := r.Header.Get("X-Internal-Token")
		if token != "valid-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/api/v1/internal/organizations/"+orgID.String()+"/members/"+userID.String() {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"is_member":       true,
				"role":            "admin",
				"organization_id": orgID.String(),
				"user_id":         userID.String(),
			},
		})
	}))
	defer server.Close()

	client := NewHTTPIdentityClient(server.URL, "valid-token", 100*time.Millisecond)

	// First call -> hits HTTP
	mem, err := client.GetMember(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mem.IsMember || mem.Role != "admin" {
		t.Fatalf("unexpected member info: %+v", mem)
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call, got %d", callCount.Load())
	}

	// Second call -> cached
	mem2, err := client.GetMember(context.Background(), orgID, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mem2.Role != "admin" {
		t.Fatalf("unexpected role: %s", mem2.Role)
	}
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 call (cached), got %d", callCount.Load())
	}

	// Unknown member -> 404
	unknownUser := uuid.New()
	_, err = client.GetMember(context.Background(), orgID, unknownUser)
	if err == nil {
		t.Fatal("expected error for unknown user")
	}
}

func TestHTTPIdentityClient_BoundedCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"is_member": true,
				"role":      "admin",
			},
		})
	}))
	defer server.Close()

	client := NewHTTPIdentityClient(server.URL, "token", 10*time.Minute)
	client.maxEntries = 3

	orgID := uuid.New()
	for range 5 {
		uID := uuid.New()
		_, err := client.GetMember(context.Background(), orgID, uID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	client.cacheMu.RLock()
	cacheLen := len(client.cache)
	client.cacheMu.RUnlock()

	if cacheLen > 3 {
		t.Fatalf("expected cache size <= 3, got %d", cacheLen)
	}
}
