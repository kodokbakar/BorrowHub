package observability

import (
	"context"
	"log/slog"
)

// LogAudit logs key security and audit events with the audit=true attribute.
func LogAudit(ctx context.Context, logger *slog.Logger, action string, args ...any) {
	if logger == nil {
		logger = slog.Default()
	}
	allArgs := make([]any, 0, len(args)+4)
	allArgs = append(allArgs, slog.Bool("audit", true), slog.String("action", action))
	allArgs = append(allArgs, args...)
	logger.InfoContext(ctx, "audit: "+action, allArgs...)
}
