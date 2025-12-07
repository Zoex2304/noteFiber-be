// FILE: internal/controller/location_controller.go
package controller

import (
	"ai-notetaking-be/internal/pkg/serverutils"
	"ai-notetaking-be/internal/service"

	"github.com/gofiber/fiber/v2"
)

type ILocationController interface {
	RegisterRoutes(r fiber.Router)
	DetectCountry(ctx *fiber.Ctx) error
	GetCountries(ctx *fiber.Ctx) error
	GetCities(ctx *fiber.Ctx) error
	GetStates(ctx *fiber.Ctx) error
	GetZipCodes(ctx *fiber.Ctx) error
}

type locationController struct {
	service service.ILocationService
}

func NewLocationController(service service.ILocationService) ILocationController {
	return &locationController{service: service}
}

func (c *locationController) RegisterRoutes(r fiber.Router) {
	h := r.Group("/location")
	h.Get("/detect-country", c.DetectCountry)
	h.Get("/countries", c.GetCountries)
	h.Get("/cities", c.GetCities)
	h.Get("/states", c.GetStates)
	h.Get("/zipcodes", c.GetZipCodes)
}

func (c *locationController) DetectCountry(ctx *fiber.Ctx) error {
	res, err := c.service.DetectCountry(ctx.Context())
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(res)
}

func (c *locationController) GetCountries(ctx *fiber.Ctx) error {
	query := ctx.Query("query", "")
	if query == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "query parameter is required"))
	}

	res, err := c.service.GetCountries(ctx.Context(), query)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(res)
}

func (c *locationController) GetCities(ctx *fiber.Ctx) error {
	country := ctx.Query("country", "ID")
	query := ctx.Query("query", "") // Bisa kosong jika state ada
	state := ctx.Query("state", "") // Parameter baru

	// Validasi: Minimal salah satu (query atau state) harus ada
	if query == "" && state == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "query parameter or state parameter is required"))
	}

	res, err := c.service.GetCities(ctx.Context(), country, query, state)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(res)
}

func (c *locationController) GetStates(ctx *fiber.Ctx) error {
	country := ctx.Query("country", "")
	city := ctx.Query("city", "")

	if country == "" || city == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "country and city parameters are required"))
	}

	res, err := c.service.GetStates(ctx.Context(), country, city)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(res)
}

func (c *locationController) GetZipCodes(ctx *fiber.Ctx) error {
	country := ctx.Query("country", "")
	city := ctx.Query("city", "")
	state := ctx.Query("state", "")

	if country == "" || city == "" {
		return ctx.Status(fiber.StatusBadRequest).JSON(serverutils.ErrorResponse(400, "country and city parameters are required"))
	}

	res, err := c.service.GetZipCodes(ctx.Context(), country, city, state)
	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(serverutils.ErrorResponse(500, err.Error()))
	}
	return ctx.JSON(res)
}