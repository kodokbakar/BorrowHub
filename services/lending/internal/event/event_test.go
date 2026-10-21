package event

import (
	"encoding/json"
	"regexp"
	"testing"
	"time"

	"github.com/google/uuid"
)

var eventTypeRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*\.[a-z][a-z0-9_]*$`)

func TestBuildOutboxEvent(t *testing.T) {
	orgID := uuid.New()
	data := map[string]any{
		"loan_request_id": uuid.New().String(),
		"item_id":         uuid.New().String(),
		"quantity":        1,
	}

	event := BuildOutboxEvent(
		EventLoanRequested,
		"req-123",
		orgID,
		"corr-456",
		"caus-789",
		data,
	)

	if event.EventType != EventLoanRequested {
		t.Fatalf("expected event type %s, got %s", EventLoanRequested, event.EventType)
	}
	if !eventTypeRegex.MatchString(event.EventType) {
		t.Fatalf("event type %s violates pattern", event.EventType)
	}
	if event.EventVersion != 1 {
		t.Fatalf("expected version 1, got %d", event.EventVersion)
	}
	if event.Source != SourceName {
		t.Fatalf("expected source %s, got %s", SourceName, event.Source)
	}
	if event.Subject != "req-123" {
		t.Fatalf("expected subject req-123, got %s", event.Subject)
	}
	if event.OrganizationID != orgID {
		t.Fatalf("expected orgID %s, got %s", orgID, event.OrganizationID)
	}
	if event.CorrelationID != "corr-456" {
		t.Fatalf("expected correlation ID corr-456, got %s", event.CorrelationID)
	}
	if event.CausationID != "caus-789" {
		t.Fatalf("expected causation ID caus-789, got %s", event.CausationID)
	}

	// Test envelope json serialization
	envelope := DomainEnvelope{
		EventID:        event.EventID.String(),
		EventType:      event.EventType,
		EventVersion:   event.EventVersion,
		Source:         event.Source,
		Subject:        event.Subject,
		OrganizationID: event.OrganizationID.String(),
		OccurredAt:     event.OccurredAt.UTC().Format(time.RFC3339Nano),
		CorrelationID:  event.CorrelationID,
		CausationID:    event.CausationID,
		Data:           event.Data,
	}

	bytes, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshaling envelope: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("unmarshaling envelope: %v", err)
	}

	requiredFields := []string{
		"event_id", "event_type", "event_version", "source", "subject",
		"organization_id", "occurred_at", "correlation_id", "causation_id", "data",
	}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Fatalf("missing required field in envelope: %s", field)
		}
	}
}
