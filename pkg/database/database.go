package database

import (
	"fmt"
	"time"

	"github.com/yuusufyan/go-common/pkg/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type ContextKey string

const (
	TxKey          ContextKey = "tx"
	UserContextKey ContextKey = "user"
)

func Connect(cfg *DBConfig, log logger.Logger, isProd bool) (*gorm.DB, error) {
	dsn := BuildDSN(cfg)

	gormConfig := &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		TranslateError:         true,
	}

	if log != nil {
		gormLogger := NewGormLogger(log)
		if cfg.SlowQueryThreshold > 0 {
			gormLogger.SlowThreshold = cfg.SlowQueryThreshold
		}
		gormConfig.Logger = gormLogger
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, errDB := db.DB()
	if errDB != nil {
		return nil, fmt.Errorf("failed to get sql db: %w", errDB)
	}

	// Config connection pooling
	maxIdleConns := cfg.MaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = 10
	}
	maxOpenConns := cfg.MaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = 100
	}
	connMaxLifetime := cfg.ConnMaxLifetime
	if connMaxLifetime == 0 {
		connMaxLifetime = 60
	}

	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Minute)

	if !cfg.DisableAuditPlugin {
		if err := db.Use(&AuditPlugin{}); err != nil {
			return nil, fmt.Errorf("failed to register audit plugin: %w", err)
		}
	}

	return db, nil
}

// BuildDSN builds a postgres DSN from cfg. cfg.DSN is returned as-is when set.
func BuildDSN(cfg *DBConfig) string {
	if cfg.DSN != "" {
		return cfg.DSN
	}

	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		cfg.Host, cfg.User, cfg.Password, cfg.DBName, cfg.Port, sslMode,
	)
	if cfg.TimeZone != "" {
		dsn += " TimeZone=" + cfg.TimeZone
	}
	return dsn
}
