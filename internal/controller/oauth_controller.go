// FILE: internal/controller/oauth_controller.go
package controller

import (
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	"fmt"
	"log"
	"os"

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
	log.Printf("[OAuth] Login initiated for provider: %s", provider)
	
	url, err := c.service.GetLoginURL(provider)
	if err != nil {
		log.Printf("[OAuth] ERROR - Failed to get login URL: %v", err)
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, err.Error()))
	}
	
	log.Printf("[OAuth] Redirecting user to: %s", url)
	// Redirect user to Google
	return ctx.Redirect(url)
}

func (c *oauthController) Callback(ctx *fiber.Ctx) error {
	provider := ctx.Params("provider")
	code := ctx.Query("code")
	// state := ctx.Query("state") // validate state if needed

	log.Printf("[OAuth] Callback received for provider: %s", provider)
	log.Printf("[OAuth] Authorization code: %s", code[:10]+"...") // Log partial code for security

	if code == "" {
		log.Printf("[OAuth] ERROR - Missing authorization code")
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Missing code"))
	}

	res, err := c.service.HandleCallback(ctx.Context(), provider, code)
	if err != nil {
		log.Printf("[OAuth] ERROR - HandleCallback failed: %v", err)
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}

	log.Printf("[OAuth] ✅ User authenticated successfully")
	log.Printf("[OAuth] User ID: %s", res.User.Id)
	log.Printf("[OAuth] User Email: %s", res.User.Email)
	log.Printf("[OAuth] User Name: %s", res.User.FullName)
	log.Printf("[OAuth] Access Token generated: %s", res.AccessToken[:20]+"...")

	// ✅ SUCCESS: Redirect to Frontend with Token in URL
	frontendURL := os.Getenv("FRONTEND_URL") // e.g., http://localhost:5173
	if frontendURL == "" {
		frontendURL = "http://localhost:5173" // fallback default
		log.Printf("[OAuth] WARNING - FRONTEND_URL not set in .env, using default: %s", frontendURL)
	}
	
	redirectURL := fmt.Sprintf("%s/app?token=%s", frontendURL, res.AccessToken)
	log.Printf("[OAuth] Redirecting to Frontend: %s", frontendURL+"/app?token=***")
	
	return ctx.Redirect(redirectURL, fiber.StatusTemporaryRedirect)
}