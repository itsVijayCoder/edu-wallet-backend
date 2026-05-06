package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	RequestIDKey ctxKey = "request_id"
	UserIDKey    ctxKey = "user_id"
)

// New creates a structured logger appropriate for the environment.
// Production: JSON output at Info level.
// Development: Text output at Debug level.
func New(env string) *slog.Logger {
	var handler slog.Handler
	if env == "production" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}
	return slog.New(handler)
}

// WithContext returns a logger enriched with request_id and user_id from context.
func WithContext(ctx context.Context, log *slog.Logger) *slog.Logger {
	if reqID, ok := ctx.Value(RequestIDKey).(string); ok {
		log = log.With("request_id", reqID)
	}
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		log = log.With("user_id", userID)
	}
	return log
}

// ContextWithRequestID stores a request ID in the context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// ContextWithUserID stores a user ID in the context.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}
