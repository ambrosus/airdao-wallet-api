package watcher

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	service Service
}

func NewHandler(service Service) (*Handler, error) {
	if service == nil {
		return nil, errors.New("[watcher_handler] invalid watcher service")
	}

	return &Handler{service: service}, nil
}

func (h *Handler) SetupRoutes(router fiber.Router) {
	router.Post("/watcher", h.AddAddressWatcherHandler)
}

type AddAddressWatcher struct {
	Address   string `json:"address" validate:"required,address"`
	PushToken string `json:"push_token" validate:"required"`
}

func (h *Handler) AddAddressWatcherHandler(c *fiber.Ctx) error {
	var reqBody AddAddressWatcher

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})

	}

	if err := Validate(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.AddAddressWatcher(c.Context(), reqBody.Address, reqBody.PushToken); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}
