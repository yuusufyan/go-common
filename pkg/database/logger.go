package database

import (
	"context"
	"errors"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/yuusufyan/go-common/pkg/logger"
	gormlogger "gorm.io/gorm/logger"
)

// GormLogger is a custom GORM logger that uses logrus and is context-aware
type GormLogger struct {
	log           logger.Logger
	SlowThreshold time.Duration
}

// NewGormLogger creates a new GORM logger bridge
func NewGormLogger(log logger.Logger) *GormLogger {
	return &GormLogger{
		log:           log,
		SlowThreshold: 200 * time.Millisecond, // Default threshold for slow queries
	}
}

func (l *GormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return l
}

func (l *GormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	l.log.WithCtx(ctx).Infof(msg, data...)
}

func (l *GormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.log.WithCtx(ctx).Warnf(msg, data...)
}

func (l *GormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	l.log.WithCtx(ctx).Errorf(msg, data...)
}

func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := logrus.Fields{
		"elapsed": elapsed,
		"rows":    rows,
		"sql":     sql,
	}

	entry := l.log.WithCtx(ctx).WithFields(fields)

	if err != nil && !errors.Is(err, gormlogger.ErrRecordNotFound) {
		entry.Errorf("DB Error: %v", err)
		return
	}

	if elapsed > l.SlowThreshold {
		entry.Warnf("Slow Query Detected")
	}
}
