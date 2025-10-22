package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type INoteController interface {
	RegisterRoutes(r fiber.Router)
	Create(ctx *fiber.Ctx) error
	Show(ctx *fiber.Ctx) error
	Update(ctx *fiber.Ctx) error
	Delete(ctx *fiber.Ctx) error
	MoveNote(ctx *fiber.Ctx) error
	SemanticSearch(ctx *fiber.Ctx) error
}

type noteController struct {
	noteService service.INoteService
}

func NewNoteController(noteService service.INoteService) INoteController {
	return &noteController{
		noteService: noteService,
	}
}

func (c *noteController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/note/v1")
	h.Get("semantic-search", c.SemanticSearch)
	h.Post("", c.Create)
	h.Get(":id", c.Show)
	h.Put(":id", c.Update)
	h.Put(":id/move", c.MoveNote)
	h.Delete(":id", c.Delete)
}

func (c *noteController) Create(ctx *fiber.Ctx) error {
	var req dto.CreateNoteRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := serverutils.ValidateRequest(req)
	if err != nil {
		return err
	}

	res, err := c.noteService.Create(ctx.Context(), &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success create note", res))
}

func (c *noteController) Show(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)
	res, err := c.noteService.Show(ctx.Context(), id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success show note", res))
}

func (c *noteController) Update(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	var req dto.UpdateNoteRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	req.Id = id

	err := serverutils.ValidateRequest(req)
	if err != nil {
		return err
	}

	res, err := c.noteService.Update(ctx.Context(), &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success update note", res))
}

func (c *noteController) Delete(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	err := c.noteService.Delete(ctx.Context(), id)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Success delete note", nil))
}

func (c *noteController) MoveNote(ctx *fiber.Ctx) error {
	idParam := ctx.Params("id")
	id, _ := uuid.Parse(idParam)

	var req dto.MoveNoteRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}
	req.Id = id

	err := serverutils.ValidateRequest(req)
	if err != nil {
		return err
	}

	res, err := c.noteService.MoveNote(ctx.Context(), &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success move note", res))
}

func (c *noteController) SemanticSearch(ctx *fiber.Ctx) error {
	q := ctx.Query("q", "")

	res, err := c.noteService.SemanticSearch(ctx.Context(), q)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success semantic search notes", res))
}
