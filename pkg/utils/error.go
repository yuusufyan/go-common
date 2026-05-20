package utils

import (
	"errors"

	"github.com/yuusufyan/go-common/pkg/apperror"
	"github.com/yuusufyan/go-common/pkg/logger"
	"github.com/yuusufyan/go-common/response"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

func NewErrorHandler(log *logrus.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
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

func NotFoundHandler(c *fiber.Ctx) error {
	return response.Error(c, fiber.StatusNotFound, "Endpoint Not Found", nil)
}
