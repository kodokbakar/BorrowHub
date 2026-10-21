package auth

import (
	"testing"
	"time"
)

func TestPasswordHashing(t *testing.T) {
	hasher := FastPasswordHasher()

	password := "CorrectHorseBatteryStaple123!"
	encoded, err := hasher.HashPassword(password)
	if err != nil {
		t.Fatalf("unexpected hash error: %v", err)
	}

	valid, err := hasher.VerifyPassword(password, encoded)
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if !valid {
		t.Fatal("expected password to match")
	}

	invalid, err := hasher.VerifyPassword("WrongPassword", encoded)
	if err != nil {
		t.Fatalf("unexpected verify error: %v", err)
	}
	if invalid {
		t.Fatal("expected password to fail")
	}

	hasher.DummyVerify("SomePassword")

	if _, err := hasher.VerifyPassword(password, "invalid$format"); err == nil {
		t.Fatal("expected error on invalid hash format")
	}
}

func TestJWTManager(t *testing.T) {
	mgr := NewJWTManager("super-secret-key-that-is-long-enough-for-testing")
	userID := "019460fa-0000-7000-8000-000000000001"
	email := "test@example.com"

	token, err := mgr.GenerateAccessToken(userID, email, time.Minute)
	if err != nil {
		t.Fatalf("unexpected token generation error: %v", err)
	}

	claims, err := mgr.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if claims.UserID != userID || claims.Email != email {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	otherMgr := NewJWTManager("wrong-secret-key-different-for-testing")
	if _, err := otherMgr.ValidateAccessToken(token); err == nil {
		t.Fatal("expected token validation to fail with wrong secret")
	}

	expiredToken, err := mgr.GenerateAccessToken(userID, email, -time.Minute)
	if err != nil {
		t.Fatalf("unexpected token generation error: %v", err)
	}
	if _, err := mgr.ValidateAccessToken(expiredToken); err == nil {
		t.Fatal("expected expired token to fail validation")
	}
}

func TestSecureTokens(t *testing.T) {
	token1, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}
	token2, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("token generation failed: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected random tokens to be distinct")
	}

	hash1 := HashToken(token1)
	hash1Again := HashToken(token1)
	if hash1 != hash1Again {
		t.Fatal("expected hashing to be deterministic")
	}

	hash2 := HashToken(token2)
	if hash1 == hash2 {
		t.Fatal("expected distinct hashes for distinct tokens")
	}
}
