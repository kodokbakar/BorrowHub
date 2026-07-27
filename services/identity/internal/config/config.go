package config

import (
	"fmt"
	"net"
	"os"
	"slices"
)

type Config struct {
	Environment string
	HTTPAddress string
}

func Load(defaultAddress string) (Config, error) {
	cfg := Config{
		Environment: value("APP_ENV", "local"),
		HTTPAddress: value("HTTP_ADDR", defaultAddress),
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
