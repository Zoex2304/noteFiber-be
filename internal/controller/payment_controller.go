// FILE: internal/controller/payment_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	"fmt"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type IPaymentController interface {
	RegisterRoutes(r fiber.Router)
	GetPlans(ctx *fiber.Ctx) error
	GetOrderSummary(ctx *fiber.Ctx) error // NEW
	Checkout(ctx *fiber.Ctx) error
	Webhook(ctx *fiber.Ctx) error
	GetStatus(ctx *fiber.Ctx) error
	CancelSubscription(ctx *fiber.Ctx) error
}

type paymentController struct {
	service service.IPaymentService
}

func NewPaymentController(service service.IPaymentService) IPaymentController {
	return &paymentController{service: service}
}

func (c *paymentController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/payment")
	h.Post("/midtrans/notification", c.Webhook)
	h.Get("/plans", c.GetPlans)
	h.Get("/summary", c.GetOrderSummary) // Public route, just needs plan_id
	
	// Protected Routes
	h.Post("/checkout", c.authMiddleware, c.Checkout)
	h.Get("/status", c.authMiddleware, c.GetStatus)
	h.Post("/cancel", c.authMiddleware, c.CancelSubscription)
}

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

// NEW Handler
func (c *paymentController) GetOrderSummary(ctx *fiber.Ctx) error {
	planIdStr := ctx.Query("plan_id")
	if planIdStr == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "plan_id is required"))
	}

	planId, err := uuid.Parse(planIdStr)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "invalid plan_id format"))
	}

	res, err := c.service.GetOrderSummary(ctx.Context(), planId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Order summary", res))
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
		fmt.Printf("[WEBHOOK ERROR] Body parsing failed: %v\n", err)
		return err
	}

	err := c.service.HandleNotification(ctx.Context(), &req)
	if err != nil {
		fmt.Printf("[WEBHOOK ERROR] Service handling failed: %v\n", err)
	}

	return ctx.SendStatus(fiber.StatusOK)
}

func (c *paymentController) GetStatus(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	res, err := c.service.GetSubscriptionStatus(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Subscription status", res))
}

func (c *paymentController) CancelSubscription(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	err := c.service.CancelSubscription(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("Subscription canceled", nil))
}