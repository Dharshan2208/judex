package logutil

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	requestIDKey ctxKey = "request_id"
)

func Init(service string, level slog.Level) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
	logger := slog.New(handler).With("service", service)
	slog.SetDefault(logger)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

func logger(ctx context.Context) *slog.Logger {
	l := slog.Default()
	if reqID := GetRequestID(ctx); reqID != "" {
		l = l.With("request_id", reqID)
	}
	return l
}

func Debug(ctx context.Context, msg string, args ...any) {
	logger(ctx).Debug(msg, args...)
}

func Info(ctx context.Context, msg string, args ...any) {
	logger(ctx).Info(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	logger(ctx).Warn(msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	logger(ctx).Error(msg, args...)
}

func Fatal(ctx context.Context, msg string, args ...any) {
	logger(ctx).Error(msg, args...)
	os.Exit(1)
}
