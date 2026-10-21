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
	RabbitMQURL     string
	SMTPHost        string
	SMTPPort        string
	SMTPUser        string
	SMTPPass        string
	SMTPFrom        string
	IdentityBaseURL string
	InternalToken   string
	JWTSecret       string
}

func Load(defaultAddress string) (Config, error) {
	if defaultAddress == "" {
		defaultAddress = ":8084"
	}

	cfg := Config{
		Environment:     value("APP_ENV", "local"),
		HTTPAddress:     value("HTTP_ADDR", defaultAddress),
		DatabaseURL:     value("DATABASE_URL", "postgres://borrowhub_notification:borrowhub_notification@127.0.0.1:5432/borrowhub_notification?sslmode=disable"),
		RabbitMQURL:     value("RABBITMQ_URL", "amqp://borrowhub:borrowhub@127.0.0.1:5672/borrowhub"),
		SMTPHost:        value("SMTP_HOST", "127.0.0.1"),
		SMTPPort:        value("SMTP_PORT", "1025"),
		SMTPUser:        value("SMTP_USER", ""),
		SMTPPass:        value("SMTP_PASS", ""),
		SMTPFrom:        value("SMTP_FROM", "notifications@borrowhub.local"),
		IdentityBaseURL: value("IDENTITY_BASE_URL", "http://127.0.0.1:8081"),
		InternalToken:   value("INTERNAL_TOKEN", "dev-internal-token-secret"),
		JWTSecret:       value("JWT_SECRET", "dev-secret-keep-at-least-32-bytes-long-for-hs256"),
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
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
