package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestLogAudit(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	LogAudit(context.Background(), logger, "auth.login_failed", "email", "attacker@test.local", "reason", "bad_password")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode json log: %v", err)
	}

	if record["audit"] != true {
		t.Fatalf("expected audit=true in log record, got %v", record["audit"])
	}
	if record["action"] != "auth.login_failed" {
		t.Fatalf("expected action=auth.login_failed, got %v", record["action"])
	}
	if record["email"] != "attacker@test.local" {
		t.Fatalf("expected email=attacker@test.local, got %v", record["email"])
	}
}
