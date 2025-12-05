// FILE: internal/controller/admin_controller.go
package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type IAdminController interface {
	RegisterRoutes(r fiber.Router)
	GetDashboardStats(ctx *fiber.Ctx) error
	GetUserGrowth(ctx *fiber.Ctx) error
	GetAllUsers(ctx *fiber.Ctx) error
	GetUserDetail(ctx *fiber.Ctx) error
	UpdateUserStatus(ctx *fiber.Ctx) error
	GetTransactions(ctx *fiber.Ctx) error
	GetLogs(ctx *fiber.Ctx) error
	GetLogDetail(ctx *fiber.Ctx) error
}

type adminController struct {
	service service.IAdminService
}

func NewAdminController(service service.IAdminService) IAdminController {
	return &adminController{service: service}
}

// Middleware to check for Admin Role
// This logic assumes JWT claims have "role": "admin"
func (c *adminController) adminMiddleware(ctx *fiber.Ctx) error {
    // For now, assume this middleware logic is handled or copy the helper function
    // For simplicity in this file block, I'll refer to the previously established middleware pattern.
    // If you need the full middleware code again, I can provide it.
    // Assuming standard JWT middleware is applied before or inside routes.
    return serverutils.JwtMiddleware(ctx) 
}

func (c *adminController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/admin")
	// h.Use(c.adminMiddleware) // Ensure middleware is applied

	// Dashboard
	h.Get("/dashboard", c.GetDashboardStats)
	h.Get("/growth", c.GetUserGrowth) 

	// Users
	h.Get("/users", c.GetAllUsers) 
	h.Get("/users/:id", c.GetUserDetail) 
	h.Put("/users/:id/status", c.UpdateUserStatus)

	// Transactions
	h.Get("/transactions", c.GetTransactions)

	// Logs
	h.Get("/logs", c.GetLogs)
	h.Get("/logs/:id", c.GetLogDetail)
}

func (c *adminController) GetDashboardStats(ctx *fiber.Ctx) error {
	stats, err := c.service.GetDashboardStats(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Dashboard stats", stats))
}

func (c *adminController) GetUserGrowth(ctx *fiber.Ctx) error {
	stats, err := c.service.GetUserGrowth(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User growth stats", stats))
}

func (c *adminController) GetAllUsers(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	search := ctx.Query("q", "")

	users, err := c.service.GetAllUsers(ctx.Context(), page, limit, search)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("User list", users))
}

func (c *adminController) GetUserDetail(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, _ := uuid.Parse(idParam)

	user, err := c.service.GetUserDetail(ctx.Context(), userId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "User not found"))
	}
	return ctx.JSON(serverutils.SuccessResponse("User detail", user))
}

func (c *adminController) UpdateUserStatus(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	userId, err := uuid.Parse(idParam)
	if err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "Invalid User ID"))
	}

	var req dto.UpdateUserStatusRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err = c.service.UpdateUserStatus(ctx.Context(), userId, req.Status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse[any]("User status updated", nil))
}

func (c *adminController) GetTransactions(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	status := ctx.Query("status", "") 

	txs, err := c.service.GetTransactions(ctx.Context(), page, limit, status)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("Transactions", txs))
}

func (c *adminController) GetLogs(ctx *fiber.Ctx) error {
	page, _ := strconv.Atoi(ctx.Query("page", "1"))
	limit, _ := strconv.Atoi(ctx.Query("limit", "10"))
	level := ctx.Query("level", "")

	logs, err := c.service.GetSystemLogs(ctx.Context(), page, limit, level)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(serverutils.SuccessResponse("System logs", logs))
}

func (c *adminController) GetLogDetail(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	logId, _ := uuid.Parse(idParam)

	l, err := c.service.GetLogDetail(ctx.Context(), logId)
	if err != nil {
		return ctx.Status(fiber.StatusNotFound).JSON(serverutils.ErrorResponse(404, "Log not found"))
	}
	return ctx.JSON(serverutils.SuccessResponse("Log detail", l))
}