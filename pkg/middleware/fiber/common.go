package fiber

import (
	gofiber "github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/pkg/logger"
)

// CommonConfig configures InstallCommonMiddlewareWithConfig.
type CommonConfig struct {
	Security SecurityConfig
	// DisableSecurity skips helmet & CORS (e.g. when handled by an API gateway).
	DisableSecurity bool
	// DisableLogger skips the HTTP request logger.
	DisableLogger bool
}

// InstallCommonMiddleware installs the standard set of middleware for a Fiber application.
// It includes Telemetry, Security (CORS/Helmet), Recovery, and Structured Logging.
func InstallCommonMiddleware(app *gofiber.App, log logger.Logger) {
	InstallCommonMiddlewareWithConfig(app, log, CommonConfig{})
}

// InstallCommonMiddlewareWithConfig is the configurable version of InstallCommonMiddleware.
func InstallCommonMiddlewareWithConfig(app *gofiber.App, log logger.Logger, cfg CommonConfig) {
	app.Use(Telemetry())
	if !cfg.DisableSecurity {
		UseSecurity(app, cfg.Security)
	}
	app.Use(Recover(log))
	if !cfg.DisableLogger {
		app.Use(Logger(log))
	}
}
