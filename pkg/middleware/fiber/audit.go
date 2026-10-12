package fiber

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/yuusufyan/go-common/pkg/apperror"
	"github.com/yuusufyan/go-common/pkg/audit"
)

// AuditTrailConfig configures the AuditTrail middleware.
type AuditTrailConfig struct {
	// Methods to audit (default: POST, PUT, PATCH, DELETE).
	Methods []string
	// Next skips the middleware when it returns true (e.g. for /health).
	Next func(c fiber.Ctx) bool
}

// AuditTrail records a TypeHTTP audit trail for every matching request.
// Register it after Telemetry and Identity so request IDs and the user are included.
func AuditTrail(rec *audit.Recorder, cfg ...AuditTrailConfig) fiber.Handler {
	var c0 AuditTrailConfig
	if len(cfg) > 0 {
		c0 = cfg[0]
	}
	if len(c0.Methods) == 0 {
		c0.Methods = []string{fiber.MethodPost, fiber.MethodPut, fiber.MethodPatch, fiber.MethodDelete}
	}
	methods := make(map[string]struct{}, len(c0.Methods))
	for _, m := range c0.Methods {
		methods[m] = struct{}{}
	}

	return func(c fiber.Ctx) error {
		if c0.Next != nil && c0.Next(c) {
			return c.Next()
		}
		if _, ok := methods[c.Method()]; !ok {
			return c.Next()
		}

		start := time.Now()
		err := c.Next()

		route := c.Path()
		if r := c.Route(); r != nil && r.Path != "" {
			route = r.Path
		}

		t := &audit.Trail{
			Type:       audit.TypeHTTP,
			Action:     c.Method(),
			Entity:     route,
			Method:     c.Method(),
			Path:       c.OriginalURL(),
			StatusCode: statusCode(c, err),
			LatencyMs:  time.Since(start).Milliseconds(),
			IP:         c.IP(),
			UserAgent:  c.Get(fiber.HeaderUserAgent),
		}
		if identity := GetUserIdentity(c); identity != nil {
			t.UserID = identity.ID
			t.UserEmail = identity.Email
		}
		if err != nil {
			t.Metadata = audit.JSONMap{"error": err.Error()}
		}
		rec.Record(c.Context(), t)

		return err
	}
}

// statusCode resolves the final status: the error handler runs after middleware,
// so the response status is not yet set when a handler returns an error.
func statusCode(c fiber.Ctx, err error) int {
	if err == nil {
		return c.Response().StatusCode()
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		return fiberErr.Code
	}
	return fiber.StatusInternalServerError
}
