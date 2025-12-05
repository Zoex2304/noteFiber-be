// FILE: internal/controller/auth_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

type IAuthController interface {
	RegisterRoutes(r fiber.Router)
	Register(ctx *fiber.Ctx) error
	Login(ctx *fiber.Ctx) error
	ForgotPassword(ctx *fiber.Ctx) error
	ResetPassword(ctx *fiber.Ctx) error
	VerifyEmail(ctx *fiber.Ctx) error
	Logout(ctx *fiber.Ctx) error // New
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
	h.Post("/verify-email", c.VerifyEmail)
	h.Post("/login", c.Login)
	h.Post("/forgot-password", c.ForgotPassword)
	h.Post("/reset-password", c.ResetPassword)
	h.Post("/logout", c.Logout) // New Route
}

func (c *authController) Register(ctx *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	res, err := c.service.Register(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"code":    400,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "User registered successfully. Check console for OTP.",
		"data":    res,
	})
}

func (c *authController) VerifyEmail(ctx *fiber.Ctx) error {
	var req dto.VerifyEmailRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := c.service.VerifyEmail(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"code":    400,
			"message": err.Error(),
		})
	}

	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "Email verified successfully",
		"data":    nil,
	})
}

func (c *authController) Login(ctx *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	res, err := c.service.Login(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"code":    401,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "Login successful",
		"data":    res,
	})
}

func (c *authController) ForgotPassword(ctx *fiber.Ctx) error {
	var req dto.ForgotPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	c.service.ForgotPassword(ctx.Context(), &req)
	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "If email exists, reset token sent",
		"data":    nil,
	})
}

func (c *authController) ResetPassword(ctx *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := c.service.ResetPassword(ctx.Context(), &req)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"code":    400,
			"message": err.Error(),
		})
	}
	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "Password reset successful",
		"data":    nil,
	})
}

func (c *authController) Logout(ctx *fiber.Ctx) error {
	// Stateless JWT logout is handled on client side by removing token.
	// We just return success here.
	return ctx.JSON(fiber.Map{
		"success": true,
		"code":    200,
		"message": "Logged out successfully",
		"data":    nil,
	})
}