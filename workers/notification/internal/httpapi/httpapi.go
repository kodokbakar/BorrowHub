package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
)

type contextKey uint8

const (
	requestIDKey contextKey = iota
)

type Meta struct {
	RequestID string `json:"request_id,omitempty"`
	Total     *int   `json:"total,omitempty"`
	Limit     *int   `json:"limit,omitempty"`
	Offset    *int   `json:"offset,omitempty"`
}

type SuccessEnvelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

func RequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}

func RequestIDs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = newID()
		}
		w.Header().Set("X-Request-ID", reqID)
		ctx := context.WithValue(r.Context(), requestIDKey, reqID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any, meta *Meta) {
	if meta == nil {
		meta = &Meta{}
	}
	if meta.RequestID == "" {
		meta.RequestID = RequestID(r.Context())
	}

	env := SuccessEnvelope{
		Data: data,
		Meta: meta,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string, details any) {
	env := ErrorEnvelope{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
