package swagger

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// DefaultPath is the route registered by SetupSwagger.
const DefaultPath = "/swagger/*"

// SetupSwagger registers the swagger endpoint to the provided Fiber app.
// Make sure you have initialized the swagger docs (using swag init) in your main application
// and imported it anonymously (e.g. _ "your-app/docs")
func SetupSwagger(app *fiber.App) {
	SetupSwaggerWithConfig(app, DefaultPath)
}

// SetupSwaggerWithConfig registers the swagger endpoint on a custom path (e.g. "/docs/*")
// and an optional swagger UI config. An empty path falls back to DefaultPath.
func SetupSwaggerWithConfig(router fiber.Router, path string, cfg ...swagger.Config) {
	if path == "" {
		path = DefaultPath
	}
	router.Get(path, swagger.New(cfg...))
}
