package serverutils

import (
	"errors"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandlerMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v\n%s", r, debug.Stack())
				_ = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":   "Internal Server Error",
					"message": fmt.Sprintf("%v", r),
				})
			}
		}()

		err := c.Next()
		if err == nil {
			return nil
		}

		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(ErrorResponse(
				fiber.StatusNotFound, "Entity not found",
			))
		}

		if fiberErr, ok := err.(*fiber.Error); ok {
			return c.Status(fiberErr.Code).JSON(ErrorResponse(
				fiberErr.Code, fiberErr.Message,
			))
		}

		if ve, ok := err.(*ValidationError); ok {
			return c.Status(fiber.StatusBadRequest).JSON(ValidationErrorResponse(ve.ToErrorDetails()))
		}

		log.Printf("[ERROR] %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse(
			fiber.StatusInternalServerError, err.Error(),
		))
	}
}
