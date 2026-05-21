package swagger

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
)

// SetupSwagger registers the swagger endpoint to the provided Fiber app.
// Make sure you have initialized the swagger docs (using swag init) in your main application
// and imported it anonymously (e.g. _ "your-app/docs")
func SetupSwagger(app *fiber.App) {
	app.Get("/swagger/*", swagger.HandlerDefault)
}
