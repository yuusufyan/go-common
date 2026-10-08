package utils

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// HealthCheckResponse defines the structure of the health check response
type HealthCheckResponse struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// HealthCheck is a single dependency check.
type HealthCheck struct {
	Name  string
	Check func(ctx context.Context) error
	// Critical marks the service DOWN (503) when this check fails.
	// Non-critical failures are reported but keep the status UP.
	Critical bool
}

// HealthConfig configures NewHealthHandlerWithConfig.
type HealthConfig struct {
	Checks []HealthCheck
	// Timeout per check (default: 2s).
	Timeout time.Duration
}

// DBHealthCheck returns a critical check that pings the database.
func DBHealthCheck(db *gorm.DB) HealthCheck {
	return HealthCheck{
		Name:     "database",
		Critical: true,
		Check: func(ctx context.Context) error {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			return sqlDB.PingContext(ctx)
		},
	}
}

// RedisHealthCheck returns a non-critical check that pings redis.
func RedisHealthCheck(rdb *redis.Client) HealthCheck {
	return HealthCheck{
		Name: "redis",
		Check: func(ctx context.Context) error {
			return rdb.Ping(ctx).Err()
		},
	}
}

// NewHealthHandler creates a standardized health check handler
func NewHealthHandler(db *gorm.DB, rdb *redis.Client) fiber.Handler {
	var checks []HealthCheck
	if db != nil {
		checks = append(checks, DBHealthCheck(db))
	}
	if rdb != nil {
		checks = append(checks, RedisHealthCheck(rdb))
	}
	return NewHealthHandlerWithConfig(HealthConfig{Checks: checks})
}

// NewHealthHandlerWithConfig creates a health check handler running arbitrary checks.
func NewHealthHandlerWithConfig(cfg HealthConfig) fiber.Handler {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 2 * time.Second
	}

	return func(c *fiber.Ctx) error {
		status := "UP"
		checks := make(map[string]string, len(cfg.Checks))

		for _, hc := range cfg.Checks {
			ctx, cancel := context.WithTimeout(c.UserContext(), cfg.Timeout)
			err := hc.Check(ctx)
			cancel()

			if err != nil {
				checks[hc.Name] = "DOWN: " + err.Error()
				if hc.Critical {
					status = "DOWN"
				}
			} else {
				checks[hc.Name] = "UP"
			}
		}

		httpStatus := fiber.StatusOK
		if status == "DOWN" {
			httpStatus = fiber.StatusServiceUnavailable
		}

		return c.Status(httpStatus).JSON(HealthCheckResponse{
			Status:    status,
			Timestamp: time.Now(),
			Checks:    checks,
		})
	}
}

// RegisterHealthCheck is a helper to register the standardized health check endpoint
func RegisterHealthCheck(router fiber.Router, db *gorm.DB, rdb *redis.Client) {
	router.Get("/health", NewHealthHandler(db, rdb))
}
