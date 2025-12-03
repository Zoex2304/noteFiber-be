// FILE: internal/controller/auth_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

type IAuthController interface {
	RegisterRoutes(r fiber.Router)
	Register(ctx *fiber.Ctx) error
	Login(ctx *fiber.Ctx) error
	ForgotPassword(ctx *fiber.Ctx) error
	ResetPassword(ctx *fiber.Ctx) error
	VerifyEmail(ctx *fiber.Ctx) error // New
}

type authController struct {
	service service.IAuthService
}

func NewAuthController(service service.IAuthService) IAuthController {
	return &authController{service: service}
}

func (c *authController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/auth")
	h.Post("/register", c.Register)
	h.Post("/verify-email", c.VerifyEmail) // New Route
	h.Post("/login", c.Login)
	h.Post("/forgot-password", c.ForgotPassword)
	h.Post("/reset-password", c.ResetPassword)
}

func (c *authController) Register(ctx *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	res, err := c.service.Register(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User registered successfully. Check console for OTP.", res))
}

func (c *authController) VerifyEmail(ctx *fiber.Ctx) error {
	var req dto.VerifyEmailRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	err := c.service.VerifyEmail(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Email verified successfully", nil))
}

func (c *authController) Login(ctx *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	res, err := c.service.Login(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(serverutils.ErrorResponse(401, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Login successful", res))
}

func (c *authController) ForgotPassword(ctx *fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	c.service.ForgotPassword(ctx.Context(), &req) // Ignore error
	// Explicitly specifying [any] solves the inference error
	return ctx.JSON(serverutils.SuccessResponse[any]("If email exists, reset token sent", nil))
}

func (c *authController) ResetPassword(ctx *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	err := c.service.ResetPassword(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, err.Error()))
	}
	// Explicitly specifying [any] solves the inference error
	return ctx.JSON(serverutils.SuccessResponse[any]("Password reset successful", nil))
}