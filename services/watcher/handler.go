package watcher

import (
	"errors"
	"net/url"

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
	router.Get("/watcher/:token", h.GetWatcherHandler)
	router.Get("/watcher-historical-prices", h.GetWatcherHistoryPricesHandler)

	router.Post("/watcher", h.CreateWatcherHandler)
	router.Put("/watcher", h.UpdateWatcherHandler)

	router.Delete("/watcher", h.DeleteWatcherHandler)
	router.Delete("/watcher-addresses", h.DeleteWatcherAddressesHandler)
}

func (h *Handler) GetWatcherHandler(c *fiber.Ctx) error {
	paramToken := c.Params("token")

	decodedParamToken, err := url.QueryUnescape(paramToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if decodedParamToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid params"})
	}

	watcher, err := h.service.GetWatcher(c.Context(), decodedParamToken)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(watcher)
}

func (h *Handler) GetWatcherHistoryPricesHandler(c *fiber.Ctx) error {
	return c.JSON(h.service.GetWatcherHistoryPrices(c.Context()))
}

type CreateWatcher struct {
	PushToken string `json:"push_token" validate:"required"`
}

func (h *Handler) CreateWatcherHandler(c *fiber.Ctx) error {
	var reqBody CreateWatcher

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := Validate(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.CreateWatcher(c.Context(), reqBody.PushToken); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}

type UpdateWatcher struct {
	PushToken string    `json:"push_token" validate:"required"`
	Addresses *[]string `json:"addresses" validate:"omitempty,addresses"`
	// Threshold *int      `json:"threshold" validate:"omitempty,threshold"`
	Threshold    *float64 `json:"threshold" validate:"omitempty"`
	Notification *string  `json:"notification" validate:"omitempty,notification"`
}

func (h *Handler) UpdateWatcherHandler(c *fiber.Ctx) error {
	var reqBody UpdateWatcher

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := Validate(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.UpdateWatcher(c.Context(), reqBody.PushToken, reqBody.Addresses, reqBody.Threshold, reqBody.Notification); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}

type DeleteWatcher struct {
	PushToken string `json:"push_token" validate:"required"`
}

func (h *Handler) DeleteWatcherHandler(c *fiber.Ctx) error {
	var reqBody DeleteWatcher

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := Validate(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.DeleteWatcher(c.Context(), reqBody.PushToken); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}

type DeleteWatcherAddresses struct {
	PushToken string   `json:"push_token" validate:"required"`
	Addresses []string `json:"addresses" validate:"required,addresses"`
}

func (h *Handler) DeleteWatcherAddressesHandler(c *fiber.Ctx) error {
	var reqBody DeleteWatcherAddresses

	if err := c.BodyParser(&reqBody); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"message": err.Error(),
		})
	}

	if err := Validate(reqBody); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.service.DeleteWatcherAddresses(c.Context(), reqBody.PushToken, reqBody.Addresses); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"status": "OK"})
}
