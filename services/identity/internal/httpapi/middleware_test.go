package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimiter(t *testing.T) {
	// Create rate limiter with 1 token/sec and burst capacity of 2
	limiter := NewRateLimiter(60, 2)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	wrapped := limiter.Middleware(dummyHandler)

	// First request: should pass
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req1.RemoteAddr = "192.168.1.100:1234"
	w1 := httptest.NewRecorder()
	wrapped.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w1.Code)
	}

	// Second request: should pass (burst = 2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req2.RemoteAddr = "192.168.1.100:1234"
	w2 := httptest.NewRecorder()
	wrapped.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}

	// Third request: burst exhausted, should return 429 RATE_LIMIT_EXCEEDED
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req3.RemoteAddr = "192.168.1.100:1234"
	w3 := httptest.NewRecorder()
	wrapped.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 Too Many Requests, got %d", w3.Code)
	}
	if retryAfter := w3.Header().Get("Retry-After"); retryAfter == "" {
		t.Fatal("expected Retry-After header to be set")
	}

	var errResp ErrorEnvelope
	if err := json.Unmarshal(w3.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to decode error json: %v", err)
	}
	if errResp.Error.Code != "RATE_LIMIT_EXCEEDED" {
		t.Fatalf("expected error code RATE_LIMIT_EXCEEDED, got %s", errResp.Error.Code)
	}
	// Request from different IP should pass
	reqOther := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	reqOther.RemoteAddr = "192.168.1.200:5678"
	wOther := httptest.NewRecorder()
	wrapped.ServeHTTP(wOther, reqOther)
	if wOther.Code != http.StatusOK {
		t.Fatalf("expected 200 for different IP, got %d", wOther.Code)
	}
}

func TestRateLimiter_MaxEntriesEviction(t *testing.T) {
	rl := NewRateLimiter(60, 5)
	rl.maxEntries = 2 // Small maxEntries to test eviction

	// Add 2 entries
	if !rl.Allow("10.0.0.1") {
		t.Fatal("expected 10.0.0.1 to be allowed")
	}
	if !rl.Allow("10.0.0.2") {
		t.Fatal("expected 10.0.0.2 to be allowed")
	}

	// Adding 3rd entry should trigger eviction and keep map bounded <= 2
	if !rl.Allow("10.0.0.3") {
		t.Fatal("expected 10.0.0.3 to be allowed")
	}
	if len(rl.buckets) > 2 {
		t.Fatalf("expected buckets count <= 2, got %d", len(rl.buckets))
	}
}

func TestClientIP_PrioritizeXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	req.Header.Set("X-Real-IP", "172.16.0.1")

	ip := clientIP(req)
	if ip != "172.16.0.1" {
		t.Fatalf("expected X-Real-IP priority 172.16.0.1, got %s", ip)
	}

	// Without X-Real-IP, use X-Forwarded-For
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "192.168.1.1:1234"
	req2.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2")
	if ip2 := clientIP(req2); ip2 != "10.0.0.1" {
		t.Fatalf("expected X-Forwarded-For 10.0.0.1, got %s", ip2)
	}

	// Without headers, parse RemoteAddr host
	req3 := httptest.NewRequest(http.MethodGet, "/", nil)
	req3.RemoteAddr = "192.168.1.1:1234"
	if ip3 := clientIP(req3); ip3 != "192.168.1.1" {
		t.Fatalf("expected RemoteAddr host 192.168.1.1, got %s", ip3)
	}
}
