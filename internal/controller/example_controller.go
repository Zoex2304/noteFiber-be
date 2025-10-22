package controller

import (
	"ai-notetaking-be/internal/dto"
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

type IExampleController interface {
	RegisterRoutes(r fiber.Router)
	HelloWorld(ctx *fiber.Ctx) error
}

type exampleController struct {
	service service.IExampleService
}

func NewExampleController(service service.IExampleService) IExampleController {
	return &exampleController{service: service}
}

func (c *exampleController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/example/v1")
	h.Post("/hello-world", c.HelloWorld)
}

func (c *exampleController) HelloWorld(ctx *fiber.Ctx) error {
	var req dto.HelloWorldRequest
	if err := ctx.BodyParser(&req); err != nil {
		return err
	}

	err := serverutils.ValidateRequest(req)
	if err != nil {
		return err
	}

	res, err := c.service.HelloWorld(ctx.Context(), &req)
	if err != nil {
		return err
	}

	return ctx.JSON(serverutils.SuccessResponse("Success", res))
}
