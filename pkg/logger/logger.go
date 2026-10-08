package logger

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type contextKey string

const (
	TraceIDKey   contextKey = "trace_id"
	RequestIDKey contextKey = "request_id"
)

// Logger defines the interface for logging
type Logger interface {
	logrus.FieldLogger
	WithCtx(ctx context.Context) *logrus.Entry
}

type appLogger struct {
	*logrus.Logger
}

func (l *appLogger) WithCtx(ctx context.Context) *logrus.Entry {
	return WithCtx(ctx, l.Logger)
}

// WithCtx extracts trace and request information from context and returns a log entry
func WithCtx(ctx context.Context, log *logrus.Logger) *logrus.Entry {
	if ctx == nil {
		return log.WithFields(logrus.Fields{})
	}

	fields := logrus.Fields{}

	// Extract Request ID
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		fields["request_id"] = id
	} else if id, ok := ctx.Value("request_id").(string); ok { // Fallback for raw string keys
		fields["request_id"] = id
	}

	// Extract Trace ID (for distributed tracing)
	if tid, ok := ctx.Value(TraceIDKey).(string); ok {
		fields["trace_id"] = tid
	}

	return log.WithFields(fields)
}

// Config configures a logger created with NewWithConfig.
type Config struct {
	// JSON enables the JSON formatter (recommended for production / ELK / Loki).
	JSON bool
	// Level is the minimum log level ("debug", "info", "warn", "error", ...).
	// Defaults to "info" when JSON is true, otherwise "debug".
	Level string
	// Output defaults to os.Stdout.
	Output io.Writer
	// Fields are attached to every log entry (e.g. {"service": "order-service"}).
	Fields logrus.Fields
	// SensitiveKeys are masked in log fields. Defaults to DefaultSensitiveKeys.
	SensitiveKeys []string
}

// New initializes a new logrus logger with standardized formatting
func New(isProd bool) Logger {
	return NewWithConfig(Config{JSON: isProd})
}

// NewWithConfig initializes a new logger from cfg.
func NewWithConfig(cfg Config) Logger {
	log := logrus.New()

	if cfg.JSON {
		// In production, use JSON for centralized logging (ELK, Loki, etc.)
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
		})
	} else {
		// In development, use colored text for readability
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: "15:04:05.000",
			FullTimestamp:   true,
			ForceColors:     true,
		})
	}

	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	log.SetOutput(cfg.Output)

	if len(cfg.Fields) > 0 {
		log.AddHook(&fieldsHook{fields: cfg.Fields})
	}

	maskHook := NewMaskHook()
	if len(cfg.SensitiveKeys) > 0 {
		maskHook.SensitiveKeys = cfg.SensitiveKeys
	}
	log.AddHook(maskHook)

	// Set default log level
	level := logrus.DebugLevel
	if cfg.JSON {
		level = logrus.InfoLevel
	}
	if cfg.Level != "" {
		if lvl, err := logrus.ParseLevel(cfg.Level); err == nil {
			level = lvl
		}
	}
	log.SetLevel(level)

	return &appLogger{Logger: log}
}

// fieldsHook attaches static fields to every entry without overriding per-entry fields.
type fieldsHook struct {
	fields logrus.Fields
}

func (h *fieldsHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *fieldsHook) Fire(entry *logrus.Entry) error {
	for k, v := range h.fields {
		if _, ok := entry.Data[k]; !ok {
			entry.Data[k] = v
		}
	}
	return nil
}
