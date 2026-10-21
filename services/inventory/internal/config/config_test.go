package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("APP_ENV", "invalid")

	if _, err := Load(":8080"); err == nil {
		t.Fatal("expected invalid environment error")
	}

	t.Setenv("APP_ENV", "local")
	t.Setenv("HTTP_ADDR", "invalid")

	if _, err := Load(":8080"); err == nil {
		t.Fatal("expected invalid address error")
	}

	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("DATABASE_URL", "postgres://test:test@localhost:5432/test")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("INTERNAL_TOKEN", "test-internal")
	t.Setenv("IDENTITY_BASE_URL", "http://localhost:8081")

	cfg, err := Load(":8080")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Environment != "local" || cfg.HTTPAddress != ":9000" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://test:test@localhost:5432/test" {
		t.Fatalf("unexpected DatabaseURL: %s", cfg.DatabaseURL)
	}
	if cfg.JWTSecret != "test-secret" {
		t.Fatalf("unexpected JWTSecret: %s", cfg.JWTSecret)
	}
	if cfg.InternalToken != "test-internal" {
		t.Fatalf("unexpected InternalToken: %s", cfg.InternalToken)
	}
	if cfg.IdentityBaseURL != "http://localhost:8081" {
		t.Fatalf("unexpected IdentityBaseURL: %s", cfg.IdentityBaseURL)
	}
}
