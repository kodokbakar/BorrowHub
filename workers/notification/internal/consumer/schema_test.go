package consumer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type EventSchema struct {
	Schema     string                 `json:"$schema"`
	ID         string                 `json:"$id"`
	Title      string                 `json:"title"`
	Type       string                 `json:"type"`
	Required   []string               `json:"required"`
	Properties map[string]PropertyDef `json:"properties"`
}

type PropertyDef struct {
	Type       any                    `json:"type"`
	Const      string                 `json:"const"`
	Required   []string               `json:"required"`
	Properties map[string]PropertyDef `json:"properties"`
}

func TestEventSchemas_AllDraft202012(t *testing.T) {
	// Find repo root from current directory
	// In test run: directory is workers/notification/internal/consumer
	pattern := filepath.Join("..", "..", "..", "..", "contracts", "events", "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob event schemas: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no event schema files found matching %s", pattern)
	}

	requiredFiles := map[string]bool{
		"loan.cancelled.schema.json":   false,
		"loan.rejected.schema.json":    false,
		"loan.requested.schema.json":   false,
		"loan.approved.schema.json":    false,
		"loan.checked_out.schema.json": false,
		"loan.overdue.schema.json":     false,
		"loan.returned.schema.json":    false,
		"envelope.schema.json":         false,
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read file %s: %v", file, err)
		}

		var s EventSchema
		if err := json.Unmarshal(data, &s); err != nil {
			t.Fatalf("file %s is invalid JSON: %v", file, err)
		}

		if s.Schema != "https://json-schema.org/draft/2020-12/schema" {
			t.Errorf("file %s has invalid $schema: got %q, expected draft 2020-12", file, s.Schema)
		}

		base := filepath.Base(file)
		if _, ok := requiredFiles[base]; ok {
			requiredFiles[base] = true
		}

		// Verify envelope required fields
		envelopeRequired := []string{"event_id", "event_type", "occurred_at", "data"}
		for _, req := range envelopeRequired {
			found := false
			for _, item := range s.Required {
				if item == req {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("file %s missing required envelope field %q", file, req)
			}
		}
	}

	for fileName, found := range requiredFiles {
		if !found {
			t.Errorf("expected required schema file %s not found", fileName)
		}
	}
}

func TestEventSchema_LoanCancelled(t *testing.T) {
	file := filepath.Join("..", "..", "..", "..", "contracts", "events", "loan.cancelled.schema.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read loan.cancelled.schema.json: %v", err)
	}

	var s EventSchema
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if s.Properties["event_type"].Const != "loan.cancelled" {
		t.Fatalf("expected event_type const 'loan.cancelled', got %s", s.Properties["event_type"].Const)
	}

	dataProp, ok := s.Properties["data"]
	if !ok {
		t.Fatal("missing data property in loan.cancelled.schema.json")
	}

	hasUserID := false
	hasLoanReqID := false
	for _, req := range dataProp.Required {
		if req == "user_id" {
			hasUserID = true
		}
		if req == "loan_request_id" {
			hasLoanReqID = true
		}
	}

	if !hasUserID {
		t.Fatal("expected user_id to be required in data for loan.cancelled")
	}
	if !hasLoanReqID {
		t.Fatal("expected loan_request_id to be required in data for loan.cancelled")
	}
}

func TestEventSchema_LoanRejected(t *testing.T) {
	file := filepath.Join("..", "..", "..", "..", "contracts", "events", "loan.rejected.schema.json")
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read loan.rejected.schema.json: %v", err)
	}

	var s EventSchema
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if s.Properties["event_type"].Const != "loan.rejected" {
		t.Fatalf("expected event_type const 'loan.rejected', got %s", s.Properties["event_type"].Const)
	}

	dataProp, ok := s.Properties["data"]
	if !ok {
		t.Fatal("missing data property in loan.rejected.schema.json")
	}

	hasAdminID := false
	hasLoanReqID := false
	for _, req := range dataProp.Required {
		if req == "admin_user_id" {
			hasAdminID = true
		}
		if req == "loan_request_id" {
			hasLoanReqID = true
		}
	}

	if !hasAdminID {
		t.Fatal("expected admin_user_id to be required in data for loan.rejected")
	}
	if !hasLoanReqID {
		t.Fatal("expected loan_request_id to be required in data for loan.rejected")
	}
}
