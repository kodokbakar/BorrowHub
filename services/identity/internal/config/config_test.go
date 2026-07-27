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
}
