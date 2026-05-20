package fiber

import (
	"fmt"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
	"github.com/yuusufyan/go-common/pkg/logger"
	"github.com/yuusufyan/go-common/response"
)

// Recover returns a middleware that recovers from panics and logs them using logrus
func Recover(log *logrus.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				err, ok := r.(error)
				if !ok {
					err = fmt.Errorf("%v", r)
				}

				stack := debug.Stack()

				logger.WithCtx(c.UserContext(), log).WithFields(logrus.Fields{
					"stack":  string(stack),
					"method": c.Method(),
					"path":   c.Path(),
				}).Errorf("Panic recovered: %v", err)

				_ = c.Status(fiber.StatusInternalServerError).JSON(response.Response[any]{
					Code:    fiber.StatusInternalServerError,
					Status:  "error",
					Message: "Internal Server Error (Panic)",
				})
			}
		}()

		return c.Next()
	}
}
