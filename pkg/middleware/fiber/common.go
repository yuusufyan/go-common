package fiber

import (
	gofiber "github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// InstallCommonMiddleware installs the standard set of middleware for a Fiber application.
// It includes Telemetry, Security (CORS/Helmet), Recovery, and Structured Logging.
func InstallCommonMiddleware(app *gofiber.App, log *logrus.Logger) {
	app.Use(Telemetry())
	app.Use(Security())
	app.Use(Recover(log))
	app.Use(Logger(log))
}
