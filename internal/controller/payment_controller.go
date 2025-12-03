// FILE: internal/controller/payment_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type IPaymentController interface {
	RegisterRoutes(r fiber.Router)
	GetPlans(ctx *fiber.Ctx) error
	Checkout(ctx *fiber.Ctx) error
	Webhook(ctx *fiber.Ctx) error
}

type paymentController struct {
	service service.IPaymentService
}

func NewPaymentController(service service.IPaymentService) IPaymentController {
	return &paymentController{service: service}
}

func (c *paymentController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/payment")
	h.Get("/plans", c.GetPlans)
	h.Post("/checkout", c.authMiddleware, c.Checkout) // Protected
	h.Post("/midtrans/notification", c.Webhook)
}

// Simple internal middleware for extraction
func (c *paymentController) authMiddleware(ctx *fiber.Ctx) error {
	authHeader := ctx.Get("Authorization")
	if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Missing token"})
	}
	tokenStr := authHeader[7:]

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !token.Valid {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid token"})
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"message": "Invalid claims"})
	}

	ctx.Locals("user_id", claims["user_id"])
	return ctx.Next()
}

func (c *paymentController) GetPlans(ctx *fiber.Ctx) error {
	res, err := c.service.GetPlans(ctx.Context())
	if err != nil {
		return err
	}
	return ctx.JSON(serverutils.SuccessResponse("Success fetching plans", res))
}

func (c *paymentController) Checkout(ctx *fiber.Ctx) error {
	var req dto.CheckoutRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	res, err := c.service.CreateSubscription(ctx.Context(), userId, &req)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Subscription created", res))
}

func (c *paymentController) Webhook(ctx *fiber.Ctx) error {
	var req dto.MidtransWebhookRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := c.service.HandleNotification(ctx.Context(), &req)
	if err != nil {
		// Log error
	}

	return ctx.SendStatus(fiber.StatusOK)
}