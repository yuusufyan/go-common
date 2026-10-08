package utils

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/yuusufyan/go-common/pkg/apperror"
	"github.com/yuusufyan/go-common/pkg/logger"
	"github.com/yuusufyan/go-common/response"

	"github.com/gofiber/fiber/v2"
)

// NewErrorHandler returns a Fiber error handler that maps AppError / fiber.Error to the
// standard response envelope. log accepts either a logger.Logger or a *logrus.Logger.
func NewErrorHandler(log logrus.FieldLogger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		entry := ctxEntry(c.UserContext(), log)

		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			if appErr.Code >= 500 {
				logger.WithCtx(c.UserContext(), log).WithError(err).Error("App Error")
			}
			return response.Error(c, appErr.Code, appErr.Message, appErr.Errors)
		}

		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			if fiberErr.Code >= 500 {
				logger.WithCtx(c.UserContext(), log).WithError(err).Error("Fiber Error")
			}
			return response.Error(c, fiberErr.Code, fiberErr.Message, nil)
		}

		// Log unhandled errors
		logger.WithCtx(c.UserContext(), log).WithError(err).Error("Unhandled Error")
		return response.Error(c, fiber.StatusInternalServerError, "Internal Server Error", nil)
	}
}

// ctxEntry attaches trace/request IDs from ctx when the logger supports it.
func ctxEntry(ctx context.Context, log logrus.FieldLogger) logrus.FieldLogger {
	switch l := log.(type) {
	case logger.Logger:
		return l.WithCtx(ctx)
	case *logrus.Logger:
		return logger.WithCtx(ctx, l)
	default:
		return log
	}
}

func NotFoundHandler(c *fiber.Ctx) error {
	return response.Error(c, fiber.StatusNotFound, "Endpoint Not Found", nil)
}
