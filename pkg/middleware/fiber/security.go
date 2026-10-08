package fiber

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
)

const (
	defaultAllowHeaders = "Origin, Content-Type, Accept, Authorization, X-Request-ID, X-Trace-ID"
	defaultAllowMethods = "GET, POST, PUT, DELETE, OPTIONS, PATCH"
)

// SecurityConfig configures the helmet headers and CORS policy.
// Empty fields fall back to the defaults.
type SecurityConfig struct {
	// AllowOrigins is a comma-separated list of origins (default: "*").
	// Restrict this in production.
	AllowOrigins     string
	AllowHeaders     string
	AllowMethods     string
	ExposeHeaders    string
	AllowCredentials bool
	// MaxAge in seconds for preflight cache.
	MaxAge int
	// Helmet overrides the helmet config (optional).
	Helmet *helmet.Config
}

// UseSecurity registers the helmet security headers and CORS middleware on the router.
//
// Helmet and CORS each call c.Next(), so they must be registered as separate
// handlers rather than combined into a single one.
func UseSecurity(router fiber.Router, cfg SecurityConfig) {
	router.Use(Helmet(cfg), CORS(cfg))
}

// Helmet returns the standard security headers middleware
// (XSS Protection, Content Type Options, etc.)
func Helmet(cfg SecurityConfig) fiber.Handler {
	if cfg.Helmet != nil {
		return helmet.New(*cfg.Helmet)
	}
	return helmet.New()
}

// CORS returns the CORS middleware built from cfg.
func CORS(cfg SecurityConfig) fiber.Handler {
	if cfg.AllowOrigins == "" {
		cfg.AllowOrigins = "*"
	}
	if cfg.AllowHeaders == "" {
		cfg.AllowHeaders = defaultAllowHeaders
	}
	if cfg.AllowMethods == "" {
		cfg.AllowMethods = defaultAllowMethods
	}

	return cors.New(cors.Config{
		AllowOrigins:     cfg.AllowOrigins,
		AllowHeaders:     cfg.AllowHeaders,
		AllowMethods:     cfg.AllowMethods,
		ExposeHeaders:    cfg.ExposeHeaders,
		AllowCredentials: cfg.AllowCredentials,
		MaxAge:           cfg.MaxAge,
	})
}
