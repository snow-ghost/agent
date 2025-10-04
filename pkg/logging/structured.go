package logging

import (
	"context"
	"log/slog"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps both slog and zap loggers
type Logger struct {
	slog *slog.Logger
	zap  *zap.Logger
}

// Config holds logging configuration
type Config struct {
	Level     string
	Format    string // "json" or "console"
	Output    string // "stdout" or "stderr"
	AddCaller bool
	AddStack  bool
}

// NewLogger creates a new structured logger
func NewLogger(config Config) (*Logger, error) {
	// Create slog logger
	slogLevel := parseSlogLevel(config.Level)
	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	})
	slogLogger := slog.New(slogHandler)

	// Create zap logger
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = parseZapLevel(config.Level)
	zapConfig.Encoding = config.Format
	zapConfig.OutputPaths = []string{config.Output}
	zapConfig.ErrorOutputPaths = []string{config.Output}
	zapConfig.DisableCaller = !config.AddCaller
	zapConfig.DisableStacktrace = !config.AddStack

	zapLogger, err := zapConfig.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		slog: slogLogger,
		zap:  zapLogger,
	}, nil
}

// parseSlogLevel parses slog level from string
func parseSlogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// parseZapLevel parses zap level from string
func parseZapLevel(level string) zap.AtomicLevel {
	switch level {
	case "debug":
		return zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		return zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		return zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	default:
		return zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
}

// WithRequestID adds request ID to logger context
func (l *Logger) WithRequestID(ctx context.Context, requestID string) *Logger {
	return &Logger{
		slog: l.slog.With("request_id", requestID),
		zap:  l.zap.With(zap.String("request_id", requestID)),
	}
}

// WithTraceID adds trace ID to logger context
func (l *Logger) WithTraceID(ctx context.Context, traceID string) *Logger {
	return &Logger{
		slog: l.slog.With("trace_id", traceID),
		zap:  l.zap.With(zap.String("trace_id", traceID)),
	}
}

// WithSpanID adds span ID to logger context
func (l *Logger) WithSpanID(ctx context.Context, spanID string) *Logger {
	return &Logger{
		slog: l.slog.With("span_id", spanID),
		zap:  l.zap.With(zap.String("span_id", spanID)),
	}
}

// WithFields adds fields to logger context
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	slogAttrs := make([]any, 0, len(fields)*2)
	zapFields := make([]zap.Field, 0, len(fields))

	for key, value := range fields {
		slogAttrs = append(slogAttrs, key, value)
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return &Logger{
		slog: l.slog.With(slogAttrs...),
		zap:  l.zap.With(zapFields...),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.slog.Debug(msg, args...)
	l.zap.Debug(msg, convertToZapFields(args)...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.slog.Info(msg, args...)
	l.zap.Info(msg, convertToZapFields(args)...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...interface{}) {
	l.slog.Warn(msg, args...)
	l.zap.Warn(msg, convertToZapFields(args)...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.slog.Error(msg, args...)
	l.zap.Error(msg, convertToZapFields(args)...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	l.slog.Error(msg, args...)
	l.zap.Fatal(msg, convertToZapFields(args)...)
}

// convertToZapFields converts interface{} args to zap.Field
func convertToZapFields(args []interface{}) []zap.Field {
	if len(args) == 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(args)/2)
	for i := 0; i < len(args)-1; i += 2 {
		if key, ok := args[i].(string); ok {
			fields = append(fields, zap.Any(key, args[i+1]))
		}
	}
	return fields
}

// LogRequest logs an HTTP request
func (l *Logger) LogRequest(ctx context.Context, method, path string, statusCode int, duration time.Duration, requestID string) {
	fields := map[string]interface{}{
		"method":      method,
		"path":        path,
		"status_code": statusCode,
		"duration_ms": float64(duration.Nanoseconds()) / 1e6,
		"request_id":  requestID,
	}

	logger := l.WithFields(fields)
	logger.Info("HTTP request completed")
}

// LogLLMRequest logs an LLM request
func (l *Logger) LogLLMRequest(ctx context.Context, provider, model string, status string, duration time.Duration, tokens int, cost float64, requestID string) {
	fields := map[string]interface{}{
		"provider":    provider,
		"model":       model,
		"status":      status,
		"duration_ms": float64(duration.Nanoseconds()) / 1e6,
		"tokens":      tokens,
		"cost":        cost,
		"request_id":  requestID,
	}

	logger := l.WithFields(fields)
	logger.Info("LLM request completed")
}

// LogCacheOperation logs a cache operation
func (l *Logger) LogCacheOperation(ctx context.Context, operation string, hit bool, requestID string) {
	fields := map[string]interface{}{
		"operation":  operation,
		"hit":        hit,
		"request_id": requestID,
	}

	logger := l.WithFields(fields)
	if hit {
		logger.Info("Cache hit")
	} else {
		logger.Info("Cache miss")
	}
}

// LogRetry logs a retry operation
func (l *Logger) LogRetry(ctx context.Context, provider, model, reason string, attempt int, requestID string) {
	fields := map[string]interface{}{
		"provider":   provider,
		"model":      model,
		"reason":     reason,
		"attempt":    attempt,
		"request_id": requestID,
	}

	logger := l.WithFields(fields)
	logger.Warn("Request retry")
}

// LogCircuitBreaker logs a circuit breaker operation
func (l *Logger) LogCircuitBreaker(ctx context.Context, provider, model, state string, requestID string) {
	fields := map[string]interface{}{
		"provider":   provider,
		"model":      model,
		"state":      state,
		"request_id": requestID,
	}

	logger := l.WithFields(fields)
	logger.Warn("Circuit breaker state changed")
}

// Sync syncs the logger
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Close closes the logger
func (l *Logger) Close() error {
	return l.zap.Sync()
}

// GetSlog returns the slog logger
func (l *Logger) GetSlog() *slog.Logger {
	return l.slog
}

// GetZap returns the zap logger
func (l *Logger) GetZap() *zap.Logger {
	return l.zap
}
