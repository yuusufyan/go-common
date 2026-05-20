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

// NewHealthHandler creates a standardized health check handler
func NewHealthHandler(db *gorm.DB, rdb *redis.Client) fiber.Handler {
	return func(c *fiber.Ctx) error {
		status := "UP"
		checks := make(map[string]string)

		// Check Database
		if db != nil {
			sqlDB, err := db.DB()
			if err != nil {
				status = "DOWN"
				checks["database"] = "ERROR: " + err.Error()
			} else {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				if err := sqlDB.PingContext(ctx); err != nil {
					status = "DOWN"
					checks["database"] = "DOWN: " + err.Error()
				} else {
					checks["database"] = "UP"
				}
			}
		}

		// Check Redis
		if rdb != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := rdb.Ping(ctx).Err(); err != nil {
				checks["redis"] = "DOWN: " + err.Error()
			} else {
				checks["redis"] = "UP"
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
