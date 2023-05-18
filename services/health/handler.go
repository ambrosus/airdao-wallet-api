package health

import (
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) SetupRoutes(router fiber.Router) {
	router.Get("/health", h.HealthCheckHandler)
}

func (h *Handler) HealthCheckHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "OK"})
}
