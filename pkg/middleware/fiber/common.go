package fiber

import (
	gofiber "github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/pkg/logger"
)

// InstallCommonMiddleware installs the standard set of middleware for a Fiber application.
// It includes Telemetry, Security (CORS/Helmet), Recovery, and Structured Logging.
func InstallCommonMiddleware(app *gofiber.App, log logger.Logger) {
	app.Use(Telemetry())
	app.Use(Security())
	app.Use(Recover(log))
	app.Use(Logger(log))
}
