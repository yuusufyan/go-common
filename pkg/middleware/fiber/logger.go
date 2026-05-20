package fiber

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/yuusufyan/go-common/pkg/logger"
)

// Logger returns a middleware that logs HTTP requests using the structured logger
func Logger(log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()

		// Process request
		err := c.Next()

		// Execution time
		latency := time.Since(start)

		// Get Trace ID and Request ID from context (set by Telemetry middleware)
		entry := logger.WithCtx(c.UserContext(), log).WithFields(logrus.Fields{
			"method":     c.Method(),
			"path":       c.Path(),
			"status":     c.Response().StatusCode(),
			"latency_ms": latency.Milliseconds(),
			"ip":         c.IP(),
			"user_agent": c.Get(fiber.HeaderUserAgent),
		})

		status := c.Response().StatusCode()
		switch {
		case status >= 500:
			if err != nil {
				entry.WithError(err).Error("HTTP Request Failed")
			} else {
				entry.Error("HTTP Request Failed")
			}
		case status >= 400:
			if err != nil {
				entry.WithError(err).Warn("HTTP Request Warning")
			} else {
				entry.Warn("HTTP Request Warning")
			}
		default:
			entry.Info("HTTP Request")
		}

		return err
	}
}
