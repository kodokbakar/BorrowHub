package config

import (
	"testing"
)

func TestConfigLoadDefaults(t *testing.T) {
	cfg, err := Load(":8084")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Environment != "local" {
		t.Fatalf("expected env local, got %s", cfg.Environment)
	}
	if cfg.HTTPAddress != ":8084" {
		t.Fatalf("expected HTTP address :8084, got %s", cfg.HTTPAddress)
	}
	if cfg.SMTPPort != "1025" {
		t.Fatalf("expected SMTP port 1025, got %s", cfg.SMTPPort)
	}
	if cfg.SMTPFrom != "notifications@borrowhub.local" {
		t.Fatalf("expected SMTP from notifications@borrowhub.local, got %s", cfg.SMTPFrom)
	}
	if cfg.InternalToken != "dev-internal-token-secret" {
		t.Fatalf("expected internal token dev-internal-token-secret, got %s", cfg.InternalToken)
	}
}

func TestConfigLoadCustom(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/testdb")
	t.Setenv("RABBITMQ_URL", "amqp://user:pass@localhost:5672/vhost")
	t.Setenv("SMTP_HOST", "mail.example.com")
	t.Setenv("SMTP_PORT", "587")
	t.Setenv("SMTP_USER", "smtpuser")
	t.Setenv("SMTP_PASS", "smtppass")
	t.Setenv("SMTP_FROM", "test@example.com")
	t.Setenv("IDENTITY_BASE_URL", "http://identity.example.com")
	t.Setenv("INTERNAL_TOKEN", "custom-internal-token")
	t.Setenv("JWT_SECRET", "custom-jwt-secret")

	cfg, err := Load(":8084")
	if err != nil {
		t.Fatalf("unexpected load error: %v", err)
	}

	if cfg.Environment != "staging" {
		t.Fatalf("expected staging, got %s", cfg.Environment)
	}
	if cfg.HTTPAddress != ":9090" {
		t.Fatalf("expected :9090, got %s", cfg.HTTPAddress)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/testdb" {
		t.Fatalf("unexpected db url: %s", cfg.DatabaseURL)
	}
	if cfg.RabbitMQURL != "amqp://user:pass@localhost:5672/vhost" {
		t.Fatalf("unexpected rabbitmq url: %s", cfg.RabbitMQURL)
	}
	if cfg.SMTPHost != "mail.example.com" || cfg.SMTPPort != "587" {
		t.Fatalf("unexpected smtp host/port: %s:%s", cfg.SMTPHost, cfg.SMTPPort)
	}
	if cfg.SMTPUser != "smtpuser" || cfg.SMTPPass != "smtppass" {
		t.Fatalf("unexpected smtp creds: %s/%s", cfg.SMTPUser, cfg.SMTPPass)
	}
	if cfg.SMTPFrom != "test@example.com" {
		t.Fatalf("unexpected smtp from: %s", cfg.SMTPFrom)
	}
	if cfg.IdentityBaseURL != "http://identity.example.com" {
		t.Fatalf("unexpected identity url: %s", cfg.IdentityBaseURL)
	}
	if cfg.InternalToken != "custom-internal-token" {
		t.Fatalf("unexpected internal token: %s", cfg.InternalToken)
	}
	if cfg.JWTSecret != "custom-jwt-secret" {
		t.Fatalf("unexpected jwt secret: %s", cfg.JWTSecret)
	}
}

func TestConfigLoadInvalidEnv(t *testing.T) {
	t.Setenv("APP_ENV", "invalid_env")
	_, err := Load(":8084")
	if err == nil {
		t.Fatal("expected error for invalid APP_ENV, got nil")
	}
}

func TestConfigLoadInvalidHTTPAddr(t *testing.T) {
	t.Setenv("HTTP_ADDR", "invalid-addr:not-a-port")
	_, err := Load(":8084")
	if err == nil {
		t.Fatal("expected error for invalid HTTP_ADDR, got nil")
	}
}
