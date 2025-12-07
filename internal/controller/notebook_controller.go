package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type INotebookController interface {
	RegisterRoutes(r fiber.Router)
	Create(ctx *fiber.Ctx) error
	Show(ctx *fiber.Ctx) error
	Update(ctx *fiber.Ctx) error
	Delete(ctx *fiber.Ctx) error
	GetAll(ctx *fiber.Ctx) error
	MoveNotebook(ctx *fiber.Ctx) error
}

type notebookController struct {
	service service.INotebookService
}

func NewNotebookController(service service.INotebookService) INotebookController {
	return &notebookController{service: service}
}

func (c *notebookController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/notebook/v1")
	h.Use(serverutils.JwtMiddleware) // ✅ PROTECTED
	h.Get("", c.GetAll)
	h.Post("", c.Create)
	h.Get(":id", c.Show)
	h.Put(":id", c.Update)
	h.Delete(":id", c.Delete)
	h.Put(":id/move", c.MoveNotebook)
}

func (c *notebookController) GetAll(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string) // Extract User ID
	userId, _ := uuid.Parse(userIdStr)

	res, err := c.service.GetAll(ctx.Context(), userId)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success get all notebook", res))
}

func (c *notebookController) Create(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)

	var req dto.CreateNotebookRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	if err := serverutils.ValidateRequest(req); err != nil {
		return err
	}

	res, err := c.service.Create(ctx.Context(), userId, &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success create notebook", res))
}

func (c *notebookController) Show(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	res, err := c.service.Show(ctx.Context(), userId, id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success show notebook", res))
}

func (c *notebookController) Update(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	var req dto.UpdateNotebookRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	req.Id = id

	res, err := c.service.Update(ctx.Context(), userId, &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success update notebook", res))
}

func (c *notebookController) Delete(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	err := c.service.Delete(ctx.Context(), userId, id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Success delete notebook", nil))
}

func (c *notebookController) MoveNotebook(ctx *fiber.Ctx) error {
	userIdStr := ctx.Locals("user_id").(string)
	userId, _ := uuid.Parse(userIdStr)
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	var req dto.MoveNotebookRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	req.Id = id

	res, err := c.service.MoveNotebook(ctx.Context(), userId, &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success move notebook", res))
}