package logger

import (
	"context"
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

// New initializes a new logrus logger with standardized formatting
func New(isProd bool) Logger {
	log := logrus.New()

	if isProd {
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

	log.SetOutput(os.Stdout)
	log.AddHook(NewMaskHook())
	
	// Set default log level
	if isProd {
		log.SetLevel(logrus.InfoLevel)
	} else {
		log.SetLevel(logrus.DebugLevel)
	}

	return &appLogger{Logger: log}
}
