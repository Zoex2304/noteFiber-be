// FILE: internal/controller/user_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type IUserController interface {
	RegisterRoutes(r fiber.Router)
	GetProfile(ctx *fiber.Ctx) error
	UpdateProfile(ctx *fiber.Ctx) error
	DeleteAccount(ctx *fiber.Ctx) error
	UploadAvatar(ctx *fiber.Ctx) error // New Route
}

type userController struct {
	service service.IUserService
}

func NewUserController(service service.IUserService) IUserController {
	return &userController{service: service}
}

func (c *userController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/user")
	h.Use(serverutils.JwtMiddleware) // Ensure this middleware is available or reuse the one from payment controller logic
	h.Get("/profile", c.GetProfile)
	h.Put("/profile", c.UpdateProfile)
	h.Delete("/account", c.DeleteAccount)
	h.Post("/avatar", c.UploadAvatar) // Registering the new endpoint
}

func (c *userController) GetProfile(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	res, err := c.service.GetProfile(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User profile", res))
}

func (c *userController) UpdateProfile(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.UpdateProfileRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	err := c.service.UpdateProfile(ctx.Context(), userId, &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Profile updated", nil))
}

func (c *userController) DeleteAccount(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	err := c.service.DeleteAccount(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Account deleted", nil))
}

func (c *userController) UploadAvatar(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	// Get file from request
	file, err := ctx.FormFile("avatar")
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Image file is required"))
	}

	url, err := c.service.UploadAvatar(ctx.Context(), userId, file)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	return ctx.JSON(serverutils.SuccessResponse("Avatar uploaded successfully", map[string]string{
		"avatar_url": url,
	}))
}