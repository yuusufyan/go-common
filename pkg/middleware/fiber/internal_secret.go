package fiber

import (
	"crypto/subtle"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/response"
)

const (
	HeaderInternalSecret = "X-Internal-Secret"

	// DefaultInternalSecretEnv is the env variable read by InternalSecretFromEnv.
	DefaultInternalSecretEnv = "INTERNAL_SERVICE_SECRET"

	defaultInternalSecretMessage = "Direct access forbidden: request must come from a trusted service"
)

// InternalSecretConfig configures the InternalSecret middleware.
type InternalSecretConfig struct {
	// Secret is the expected value. If empty, it is read from EnvKey.
	Secret string
	// EnvKey is the env variable holding the secret (default: INTERNAL_SERVICE_SECRET).
	EnvKey string
	// Header is the request header carrying the secret (default: X-Internal-Secret).
	Header string
	// Message is returned with 403 when the secret is missing or invalid.
	Message string
	// Next skips the middleware when it returns true (e.g. for /health).
	Next func(c *fiber.Ctx) bool
}

// InternalSecretFromEnv reads the expected secret from the INTERNAL_SERVICE_SECRET env variable.
// Falls back to the provided defaultSecret if env is not set.
// Use this in production so the secret can be rotated without a code change.
//
// Usage:
//
//	app.Use(fiber.InternalSecretFromEnv(""))
func InternalSecretFromEnv(defaultSecret string) fiber.Handler {
	secret := os.Getenv(DefaultInternalSecretEnv)
	if secret == "" {
		secret = defaultSecret
	}
	return InternalSecret(secret)
}

// InternalSecret verifies the X-Internal-Secret header matches the expected value.
// Ensures requests to the service originate from a trusted upstream (gateway/BFF), not directly from clients.
func InternalSecret(expectedSecret string) fiber.Handler {
	return InternalSecretWithConfig(InternalSecretConfig{Secret: expectedSecret})
}

// InternalSecretWithConfig is the configurable version of InternalSecret.
// If no secret can be resolved, every request is rejected.
func InternalSecretWithConfig(cfg InternalSecretConfig) fiber.Handler {
	if cfg.EnvKey == "" {
		cfg.EnvKey = DefaultInternalSecretEnv
	}
	if cfg.Secret == "" {
		cfg.Secret = os.Getenv(cfg.EnvKey)
	}
	if cfg.Header == "" {
		cfg.Header = HeaderInternalSecret
	}
	if cfg.Message == "" {
		cfg.Message = defaultInternalSecretMessage
	}
	expected := []byte(cfg.Secret)

	return func(c *fiber.Ctx) error {
		if cfg.Next != nil && cfg.Next(c) {
			return c.Next()
		}

		incoming := c.Get(cfg.Header)
		if len(expected) == 0 || incoming == "" ||
			subtle.ConstantTimeCompare([]byte(incoming), expected) != 1 {
			return response.Error(c, fiber.StatusForbidden, cfg.Message, nil)
		}

		return c.Next()
	}
}
