package database

import "time"

type DBConfig struct {
	Host     string
	User     string
	Password string
	DBName   string
	Port     int
	// SSLMode is the postgres sslmode (default: "disable").
	SSLMode string
	// TimeZone is the session time zone, e.g. "UTC" or "Asia/Jakarta".
	// When empty the database server default is used.
	TimeZone string
	// DSN overrides all connection fields above when set.
	DSN string

	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int // In minutes

	// SlowQueryThreshold for the GORM logger (default: 200ms).
	SlowQueryThreshold time.Duration
	// DisableAuditPlugin skips registering the AuditPlugin.
	DisableAuditPlugin bool
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}
