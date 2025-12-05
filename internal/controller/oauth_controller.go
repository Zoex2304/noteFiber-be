// FILE: internal/controller/oauth_controller.go
package controller

import (
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

type IOAuthController interface {
	RegisterRoutes(r fiber.Router)
	Login(ctx *fiber.Ctx) error
	Callback(ctx *fiber.Ctx) error
}

type oauthController struct {
	service service.IOAuthService
}

func NewOAuthController(service service.IOAuthService) IOAuthController {
	return &oauthController{service: service}
}

func (c *oauthController) RegisterRoutes(r fiber.Router) {
	// e.g., /auth/google
	h := r.Group("/auth")
	h.Get("/:provider", c.Login)
	h.Get("/:provider/callback", c.Callback)
}

func (c *oauthController) Login(ctx *fiber.Ctx) error {
	provider := ctx.Params("provider")
	url, err := c.service.GetLoginURL(provider)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, err.Error()))
	}
	// Redirect user to Google
	return ctx.Redirect(url)
}

func (c *oauthController) Callback(ctx *fiber.Ctx) error {
	provider := ctx.Params("provider")
	code := ctx.Query("code")
	// state := ctx.Query("state") // validate state if needed

	if code == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Missing code"))
	}

	res, err := c.service.HandleCallback(ctx.Context(), provider, code)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	// Success: Return JWT or Redirect to Frontend with Token
	return ctx.JSON(serverutils.SuccessResponse("OAuth login successful", res))
}