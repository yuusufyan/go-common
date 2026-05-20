package fiber

import (
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/response"
)

const HeaderInternalSecret = "X-Internal-Secret"

// InternalSecretFromEnv reads the expected secret from the INTERNAL_SERVICE_SECRET env variable.
// Falls back to the provided defaultSecret if env is not set.
// Use this in production so the secret can be rotated without a code change.
//
// Usage:
//
//	app.Use(fiber.InternalSecretFromEnv("smf-fallback-key"))
func InternalSecretFromEnv(defaultSecret string) fiber.Handler {
	secret := os.Getenv("INTERNAL_SERVICE_SECRET")
	if secret == "" {
		secret = defaultSecret
	}
	return InternalSecret(secret)
}

// InternalSecret verifies the X-Internal-Secret header matches the expected value.
// Ensures requests to Go services originate from the trusted BFF, not directly from clients.
func InternalSecret(expectedSecret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		incoming := c.Get(HeaderInternalSecret)

		if incoming == "" || incoming != expectedSecret {
			return response.Error(c, fiber.StatusForbidden,
				"Direct access forbidden: request must route through BFF", nil)
		}

		return c.Next()
	}
}
