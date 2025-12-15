package goadstc

import (
	"context"
	"log/slog"
	"os"
)

// Logger defines the interface for structured logging in goadstc.
// It follows the standard slog.Logger interface for compatibility.
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
	With(args ...any) Logger
}

// slogAdapter adapts slog.Logger to our Logger interface.
type slogAdapter struct {
	logger *slog.Logger
}

func (s *slogAdapter) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *slogAdapter) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *slogAdapter) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *slogAdapter) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

func (s *slogAdapter) With(args ...any) Logger {
	return &slogAdapter{logger: s.logger.With(args...)}
}

// noopLogger implements Logger with no-op operations for minimal overhead.
type noopLogger struct{}

func (n *noopLogger) Debug(msg string, args ...any) {}
func (n *noopLogger) Info(msg string, args ...any)  {}
func (n *noopLogger) Warn(msg string, args ...any)  {}
func (n *noopLogger) Error(msg string, args ...any) {}
func (n *noopLogger) With(args ...any) Logger       { return n }

var (
	// DefaultLogger is a no-op logger to minimize overhead when logging is not configured.
	DefaultLogger Logger = &noopLogger{}
)

// NewSlogLogger creates a Logger from a slog.Logger.
func NewSlogLogger(logger *slog.Logger) Logger {
	if logger == nil {
		return DefaultLogger
	}
	return &slogAdapter{logger: logger}
}

// NewDefaultLogger creates a basic JSON logger writing to stderr.
func NewDefaultLogger() Logger {
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return &slogAdapter{logger: slog.New(handler)}
}

// WithLogger returns a new option that sets the logger for the client.
func WithLogger(logger Logger) Option {
	return func(c *clientConfig) error {
		c.logger = logger
		return nil
	}
}

// LogContext adds context values to a logger if the context contains log fields.
type logContextKey struct{}

// ContextWithLogFields adds log fields to a context that will be automatically
// included in log messages when using LoggerFromContext.
func ContextWithLogFields(ctx context.Context, args ...any) context.Context {
	existing, _ := ctx.Value(logContextKey{}).([]any)
	return context.WithValue(ctx, logContextKey{}, append(existing, args...))
}

// LoggerFromContext returns a logger with context fields attached.
func LoggerFromContext(ctx context.Context, logger Logger) Logger {
	if fields, ok := ctx.Value(logContextKey{}).([]any); ok && len(fields) > 0 {
		return logger.With(fields...)
	}
	return logger
}
