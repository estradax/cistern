package main

import (
	"github.com/estradax/cistern/internal/client"
	"github.com/gofiber/fiber/v3"
)

// @Summary Create a client
// @Description Create a new client with the given name
// @Tags clients
// @Accept json
// @Produce json
// @Param body body client.CreateClientInput true "Create client payload"
// @Success 201 {object} client.Client
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /clients [post]
func (s *Server) CreateClient(c fiber.Ctx) error {
	var input client.CreateClientInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	cli, err := s.clientRepo.Create(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(cli)
}

// @Summary Get a client
// @Description Get a client by its unique ID
// @Tags clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 200 {object} client.Client
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /clients/{id} [get]
func (s *Server) GetClient(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing client ID"})
	}

	cli, err := s.clientRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if cli == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "client not found"})
	}

	return c.JSON(cli)
}

// @Summary Update a client
// @Description Update a client's name by its unique ID
// @Tags clients
// @Accept json
// @Produce json
// @Param id path string true "Client ID"
// @Param body body client.CreateClientInput true "Update client payload (only name is needed)"
// @Success 200 {object} client.Client
// @Failure 400 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /clients/{id} [put]
func (s *Server) UpdateClient(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing client ID"})
	}

	var input client.CreateClientInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	updateInput := client.UpdateClientInput{
		ID:   id,
		Name: input.Name,
	}

	cli, err := s.clientRepo.Update(c.Context(), updateInput)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.JSON(cli)
}

// @Summary Delete a client
// @Description Delete a client by its unique ID
// @Tags clients
// @Produce json
// @Param id path string true "Client ID"
// @Success 204 "No Content"
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /clients/{id} [delete]
func (s *Server) DeleteClient(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing client ID"})
	}

	if err := s.clientRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
