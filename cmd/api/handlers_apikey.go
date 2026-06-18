package main

import (
	"github.com/estradax/cistern/internal/apikey"
	"github.com/gofiber/fiber/v3"
)

// @Summary Generate an API key
// @Description Generate a new API key (client_id is required)
// @Tags API Keys
// @Accept json
// @Produce json
// @Param body body apikey.CreateAPIKeyInput true "Generate API Key payload"
// @Success 201 {object} apikey.CreateAPIKeyResult
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /apikeys [post]
func (s *Server) GenerateAPIKey(c fiber.Ctx) error {
	var input apikey.CreateAPIKeyInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	res, err := s.apiKeyRepo.Create(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(res)
}

// @Summary Get an API key
// @Description Get API key metadata by its unique ID
// @Tags API Keys
// @Produce json
// @Param id path string true "API Key ID"
// @Success 200 {object} apikey.APIKey
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /apikeys/{id} [get]
func (s *Server) GetAPIKey(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing API key ID"})
	}

	key, err := s.apiKeyRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if key == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "API key not found"})
	}

	return c.JSON(key)
}

// @Summary Delete an API key
// @Description Delete an API key by its unique ID
// @Tags API Keys
// @Produce json
// @Param id path string true "API Key ID"
// @Success 204 "No Content"
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /apikeys/{id} [delete]
func (s *Server) DeleteAPIKey(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing API key ID"})
	}

	if err := s.apiKeyRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
