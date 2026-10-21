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

	LogAudit(context.Background(), logger, "loan.approved", "loan_id", "1234", "reviewer", "admin")

	var record map[string]any
	if err := json.Unmarshal(buf.Bytes(), &record); err != nil {
		t.Fatalf("failed to decode json log: %v", err)
	}

	if record["audit"] != true {
		t.Fatalf("expected audit=true in log record, got %v", record["audit"])
	}
	if record["action"] != "loan.approved" {
		t.Fatalf("expected action=loan.approved, got %v", record["action"])
	}
	if record["loan_id"] != "1234" {
		t.Fatalf("expected loan_id=1234, got %v", record["loan_id"])
	}
}
