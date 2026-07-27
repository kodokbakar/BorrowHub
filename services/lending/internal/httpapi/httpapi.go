package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
)

type contextKey uint8

const (
	requestIDKey contextKey = iota
	correlationIDKey
)

type Meta struct {
	RequestID string `json:"request_id"`
}

type SuccessEnvelope struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type ErrorDetail struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Fields    map[string]any `json:"fields,omitempty"`
	RequestID string         `json:"request_id"`
}

type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func CorrelationID(ctx context.Context) string {
	value, _ := ctx.Value(correlationIDKey).(string)
	return value
}

func RequestIDs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newID()
		correlationID := validCorrelationID(
			r.Header.Get(HeaderCorrelationID),
		)

		if correlationID == "" {
			correlationID = requestID
		}

		w.Header().Set(HeaderRequestID, requestID)
		w.Header().Set(HeaderCorrelationID, correlationID)

		ctx := context.WithValue(
			r.Context(),
			requestIDKey,
			requestID,
		)
		ctx = context.WithValue(
			ctx,
			correlationIDKey,
			correlationID,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Logging(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			writer := &responseWriter{
				ResponseWriter: w,
			}

			next.ServeHTTP(writer, r)

			status := writer.status
			if status == 0 {
				status = http.StatusOK
			}

			route := r.Pattern
			if route == "" {
				route = "unmatched"
			}

			level := slog.LevelInfo

			switch {
			case status >= http.StatusInternalServerError:
				level = slog.LevelError
			case status >= http.StatusBadRequest:
				level = slog.LevelWarn
			}

			fields := []any{
				"request_id", RequestID(r.Context()),
				"correlation_id", CorrelationID(r.Context()),
				"method", r.Method,
				"route", route,
				"status", status,
				"duration_ms", time.Since(started).Milliseconds(),
			}

			if writer.errorCode != "" {
				fields = append(
					fields,
					"error_code",
					writer.errorCode,
				)
			}

			logger.Log(
				r.Context(),
				level,
				"http request",
				fields...,
			)
		})
	}
}

func WriteJSON(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	data any,
) {
	writeJSON(
		w,
		status,
		SuccessEnvelope{
			Data: data,
			Meta: Meta{
				RequestID: RequestID(r.Context()),
			},
		},
	)
}

func WriteError(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	code string,
	message string,
	fields map[string]any,
) {
	if writer, ok := w.(interface{ setErrorCode(string) }); ok {
		writer.setErrorCode(code)
	}

	writeJSON(
		w,
		status,
		ErrorEnvelope{
			Error: ErrorDetail{
				Code:      code,
				Message:   message,
				Fields:    fields,
				RequestID: RequestID(r.Context()),
			},
		},
	)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func newID() string {
	var value [16]byte

	if _, err := rand.Read(value[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}

	return hex.EncodeToString(value[:])
}

func validCorrelationID(value string) string {
	value = strings.TrimSpace(value)

	if value == "" || len(value) > 128 {
		return ""
	}

	if strings.IndexFunc(value, func(char rune) bool {
		return unicode.IsControl(char) || unicode.IsSpace(char)
	}) >= 0 {
		return ""
	}

	return value
}

type responseWriter struct {
	http.ResponseWriter
	status    int
	errorCode string
}

func (w *responseWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}

	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *responseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(body)
}

func (w *responseWriter) setErrorCode(code string) {
	w.errorCode = code
}
