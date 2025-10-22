package serverutils

import "github.com/gofiber/fiber/v2"

type BaseResponse[T any] struct {
	Success bool          `json:"success"`
	Code    int           `json:"code"`
	Message string        `json:"message"`
	Data    T             `json:"data,omitempty"`
	Errors  []ErrorDetail `json:"errors,omitempty"`
}

func SuccessResponse[T any](message string, data T) BaseResponse[T] {
	return BaseResponse[T]{
		Success: true,
		Code:    fiber.StatusOK,
		Message: message,
		Data:    data,
	}
}

func SuccessWithCodeResponse[T any](message string, code int, data T) BaseResponse[T] {
	return BaseResponse[T]{
		Success: true,
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func ErrorResponse(statusCode int, message string) BaseResponse[any] {
	return BaseResponse[any]{
		Success: false,
		Code:    statusCode,
		Message: message,
		Data:    nil,
	}
}

func ValidationErrorResponse(details []ErrorDetail) BaseResponse[any] {
	return BaseResponse[any]{
		Success: true,
		Code:    fiber.StatusBadRequest,
		Message: "Validation failed",
		Errors:  details,
	}
}
