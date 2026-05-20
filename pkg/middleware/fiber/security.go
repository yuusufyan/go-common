package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

// Security applies standard security headers and CORS policy
func Security() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Use helmet for standard security headers
		// (XSS Protection, Content Type Options, etc.)
		h := helmet.New()
		
		// Use CORS
		co := cors.New(cors.Config{
			AllowOrigins: "*", // Adjust for production!
			AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Trace-ID",
			AllowMethods: "GET, POST, PUT, DELETE, OPTIONS, PATCH",
		})

		// Apply sequentially
		if err := h(c); err != nil {
			return err
		}
		return co(c)
	}
}
