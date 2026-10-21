package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckoutToken(t *testing.T) {
	plainToken, tokenHash, err := GenerateSecureToken()
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}

	if len(plainToken) != 64 {
		t.Fatalf("expected 64 hex chars for plainToken, got %d", len(plainToken))
	}
	if len(tokenHash) != 64 {
		t.Fatalf("expected 64 hex chars for tokenHash, got %d", len(tokenHash))
	}

	// Verify hashing is deterministic
	computedHash := HashToken(plainToken)
	if computedHash != tokenHash {
		t.Fatalf("expected hash %s, got %s", tokenHash, computedHash)
	}

	// Verify token model status
	now := time.Now()
	loanID := uuid.New()
	token := CheckoutToken{
		ID:        uuid.New(),
		LoanID:    loanID,
		TokenHash: tokenHash,
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	}

	// Valid token check
	if token.UsedAt != nil {
		t.Fatal("expected unused token")
	}
	if time.Now().After(token.ExpiresAt) {
		t.Fatal("token should not be expired")
	}

	// Mark used
	used := now.Add(time.Minute)
	token.UsedAt = &used
	if token.UsedAt == nil {
		t.Fatal("expected token to be marked used")
	}
}
