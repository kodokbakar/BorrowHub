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

	cfg, err := Load(":8080")
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Environment != "local" || cfg.HTTPAddress != ":9000" {
		t.Fatalf("unexpected config: %+v", cfg)
	}

	if cfg.DatabaseURL == "" || cfg.JWTSecret == "" || cfg.InternalToken == "" {
		t.Fatalf("expected non-empty defaults, got: %+v", cfg)
	}

	t.Setenv("DATABASE_URL", "postgres://custom:custom@localhost:5432/custom")
	t.Setenv("JWT_SECRET", "custom-secret-key-1234567890123456")
	t.Setenv("INTERNAL_TOKEN", "custom-internal-token")

	cfg, err = Load(":8080")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://custom:custom@localhost:5432/custom" || cfg.JWTSecret != "custom-secret-key-1234567890123456" || cfg.InternalToken != "custom-internal-token" {
		t.Fatalf("unexpected custom config: %+v", cfg)
	}
}
