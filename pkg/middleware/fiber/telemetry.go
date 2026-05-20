package fiber

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/yuusufyan/go-common/pkg/logger"
)

// Telemetry returns a middleware that manages Trace IDs and Request IDs
func Telemetry() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Get or Generate Request ID
		requestID := c.Get(fiber.HeaderXRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set(fiber.HeaderXRequestID, requestID)

		// 2. Get or Generate Trace ID (for distributed tracing)
		traceID := c.Get("X-Trace-ID")
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("X-Trace-ID", traceID)

		// 3. Store in Fiber locals for easy access in handlers
		c.Locals("request_id", requestID)
		c.Locals("trace_id", traceID)

		// 4. Create a context with these IDs and pass it down
		// Note: When you can add go.opentelemetry.io/otel, replace this with
		// actual OTel trace propagation.
		ctx := c.UserContext()
		ctx = context.WithValue(ctx, logger.RequestIDKey, requestID)
		ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)
		c.SetUserContext(ctx)

		return c.Next()
	}
}
