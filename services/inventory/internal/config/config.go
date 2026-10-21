package config

import (
	"fmt"
	"net"
	"os"
	"slices"
)

type Config struct {
	Environment     string
	HTTPAddress     string
	DatabaseURL     string
	JWTSecret       string
	InternalToken   string
	IdentityBaseURL string
}

func Load(defaultAddress string) (Config, error) {
	cfg := Config{
		Environment:     value("APP_ENV", "local"),
		HTTPAddress:     value("HTTP_ADDR", defaultAddress),
		DatabaseURL:     value("DATABASE_URL", "postgres://borrowhub_inventory:borrowhub_inventory@127.0.0.1:5432/borrowhub_inventory?sslmode=disable"),
		JWTSecret:       value("JWT_SECRET", "dev-secret-keep-at-least-32-bytes-long-for-hs256"),
		InternalToken:   value("INTERNAL_TOKEN", "dev-internal-token-secret"),
		IdentityBaseURL: value("IDENTITY_BASE_URL", "http://127.0.0.1:8081"),
	}
	if !slices.Contains(
		[]string{"local", "test", "staging", "production"},
		cfg.Environment,
	) {
		return Config{}, fmt.Errorf("invalid APP_ENV %q", cfg.Environment)
	}

	if _, err := net.ResolveTCPAddr("tcp", cfg.HTTPAddress); err != nil {
		return Config{}, fmt.Errorf("invalid HTTP_ADDR %q: %w", cfg.HTTPAddress, err)
	}

	return cfg, nil
}

func value(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
