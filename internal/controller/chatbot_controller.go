package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type IChatbotController interface {
	RegisterRoutes(r fiber.Router)
	CreateSession(ctx *fiber.Ctx) error
	GetAllSessions(ctx *fiber.Ctx) error
	GetChatHistory(ctx *fiber.Ctx) error
	SendChat(ctx *fiber.Ctx) error
	DeleteSession(ctx *fiber.Ctx) error
}

type chatbotController struct {
	chatbotService service.IChatbotService
}

func NewChatbotController(chatbotService service.IChatbotService) IChatbotController {
	return &chatbotController{
		chatbotService: chatbotService,
	}
}

func (c *chatbotController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/chatbot/v1")
	h.Get("sessions", c.GetAllSessions)
	h.Get("chat-history", c.GetChatHistory)
	h.Post("create-session", c.CreateSession)
	h.Post("send-chat", c.SendChat)
	h.Delete("delete-session", c.DeleteSession)
}

func (c *chatbotController) CreateSession(ctx *fiber.Ctx) error {
	res, err := c.chatbotService.CreateSession(ctx.Context())
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success create session", res))
}

func (c *chatbotController) GetAllSessions(ctx *fiber.Ctx) error {
	res, err := c.chatbotService.GetAllSessions(ctx.Context())
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success get all sessions", res))
}

func (c *chatbotController) GetChatHistory(ctx *fiber.Ctx) error {
	idStr := ctx.Query("chat_session_id")
	sessionId, _ := uuid.Parse(idStr)

	res, err := c.chatbotService.GetChatHistory(ctx.Context(), sessionId)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success get chat history", res))
}

func (c *chatbotController) SendChat(ctx *fiber.Ctx) error {
	var request dto.SendChatRequest

	err := ctx.BodyParser(&request)
	if err != nil {
		return err
	}

	if err = serverutils.ValidateRequest(request); err != nil {
		return err
	}

	res, err := c.chatbotService.SendChat(ctx.Context(), &request)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success send chat", res))
}

func (c *chatbotController) DeleteSession(ctx *fiber.Ctx) error {
	var request dto.DeleteSessionRequest

	err := ctx.BodyParser(&request)
	if err != nil {
		return err
	}

	if err = serverutils.ValidateRequest(request); err != nil {
		return err
	}

	err = c.chatbotService.DeleteSession(ctx.Context(), &request)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse[any]("Success delete session", nil))
}
