package httpapi

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMiddlewareAndEnvelopes(t *testing.T) {
	var logs bytes.Buffer

	logger := slog.New(
		slog.NewJSONHandler(&logs, nil),
	)

	handler := RequestIDs(
		Logging(logger)(
			http.HandlerFunc(func(
				w http.ResponseWriter,
				r *http.Request,
			) {
				WriteError(
					w,
					r,
					http.StatusBadRequest,
					"INVALID_REQUEST",
					"Request tidak valid.",
					map[string]any{
						"name": "required",
					},
				)
			}),
		),
	)

	request := httptest.NewRequest(
		http.MethodPost,
		"/test",
		nil,
	)
	request.Header.Set(
		HeaderCorrelationID,
		"correlation-123",
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: %d", response.Code)
	}

	requestID := response.Header().Get(HeaderRequestID)

	if requestID == "" {
		t.Fatal("request ID is empty")
	}

	if response.Header().Get(HeaderCorrelationID) != "correlation-123" {
		t.Fatal("correlation ID was not preserved")
	}

	var envelope ErrorEnvelope

	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}

	if envelope.Error.Code != "INVALID_REQUEST" {
		t.Fatalf("unexpected error code: %s", envelope.Error.Code)
	}

	if envelope.Error.RequestID != requestID {
		t.Fatal("response request ID does not match header")
	}

	if !strings.Contains(
		logs.String(),
		`"error_code":"INVALID_REQUEST"`,
	) {
		t.Fatal("error code was not logged")
	}
}

func TestInvalidCorrelationIDUsesRequestID(t *testing.T) {
	handler := RequestIDs(
		http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			WriteJSON(
				w,
				r,
				http.StatusOK,
				map[string]string{
					"status": "ok",
				},
			)
		}),
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/test",
		nil,
	)
	request.Header.Set(
		HeaderCorrelationID,
		"invalid correlation",
	)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if response.Header().Get(HeaderCorrelationID) !=
		response.Header().Get(HeaderRequestID) {
		t.Fatal("invalid correlation ID was accepted")
	}
}
