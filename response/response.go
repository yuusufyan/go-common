package response

import (
	"math"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/pkg/apperror"
)

// Response is the Gold Standard generic response envelope
type Response[T any] struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
}

// Common is a type alias for Response[any] for backward compatibility
type Common = Response[any]

// TokenExpires is a standardized token DTO
type TokenExpires struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expires"`
}

// Tokens is a standardized pair of tokens DTO
type Tokens struct {
	Access  TokenExpires `json:"access"`
	Refresh TokenExpires `json:"refresh"`
}

// Meta contains metadata for paginated results (for JSON output)
type Meta struct {
	Total        int64 `json:"total"`
	Page         int   `json:"page"`
	Limit        int   `json:"limit"`
	TotalPages   int   `json:"total_pages"`
	TotalResults int64 `json:"total_results"`
}

// Pagination contains data and its metadata in a flat structure for internal use
type Pagination struct {
	Data         interface{} `json:"data"`
	Total        int64       `json:"total"`
	Page         int         `json:"page"`
	Limit        int         `json:"limit"`
	TotalPages   int         `json:"total_pages"`
	TotalResults int64       `json:"total_results"`
}

// JSON is the standard way to send any generic response in Fiber
func JSON[T any](c *fiber.Ctx, code int, message string, data T) error {
	status := "success"
	if code >= 400 {
		status = "error"
	}

	return c.Status(code).JSON(Response[T]{
		Code:    code,
		Status:  status,
		Message: message,
		Data:    data,
	})
}

// Success is a convenience wrapper for successful JSON responses
func Success(c *fiber.Ctx, code int, message string, data interface{}) error {
	return JSON(c, code, message, data)
}

// Error is a convenience wrapper for error JSON responses
func Error(c *fiber.Ctx, code int, message string, data interface{}) error {
	return JSON(c, code, message, data)
}

// RespondWithError is a helper to respond using an error object for Fiber
func RespondWithError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	code := fiber.StatusInternalServerError
	message := err.Error()
	var details interface{}

	// Check if it's our custom AppError
	if appErr, ok := err.(*apperror.AppError); ok {
		code = appErr.Code
		message = appErr.Message
		details = appErr.Errors
	}

	status := "error"
	return c.Status(code).JSON(Response[interface{}]{
		Code:    code,
		Status:  status,
		Message: message,
		Data:    details,
	})
}

// Paginate is the standard way to send paginated responses in Fiber
func Paginate[T any](c *fiber.Ctx, message string, data []T, total int64, page, limit int) error {
	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := Meta{
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
		TotalResults: total,
	}

	return c.Status(fiber.StatusOK).JSON(Response[[]T]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// RespondWithPagination is a helper that takes a flat Pagination object and sends a standardized nested response
func RespondWithPagination(c *fiber.Ctx, message string, p *Pagination) error {
	if p == nil {
		return c.Status(fiber.StatusOK).JSON(Response[[]interface{}]{
			Code:    fiber.StatusOK,
			Status:  "success",
			Message: message,
			Data:    []interface{}{},
			Meta: Meta{
				Total:        0,
				Page:         1,
				Limit:        10,
				TotalPages:   0,
				TotalResults: 0,
			},
		})
	}

	meta := Meta{
		Total:        p.Total,
		Page:         p.Page,
		Limit:        p.Limit,
		TotalPages:   p.TotalPages,
		TotalResults: p.TotalResults,
	}

	return c.Status(fiber.StatusOK).JSON(Response[interface{}]{
		Code:    fiber.StatusOK,
		Status:  "success",
		Message: message,
		Data:    p.Data,
		Meta:    meta,
	})
}
