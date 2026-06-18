package main

import (
	"github.com/estradax/cistern/internal/bucket"
	"github.com/gofiber/fiber/v3"
)

// @Summary Create a bucket
// @Description Create a new bucket with the given key and owner client ID
// @Tags buckets
// @Accept json
// @Produce json
// @Param body body bucket.CreateBucketInput true "Create bucket payload"
// @Success 201 {object} bucket.Bucket
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets [post]
func (s *Server) CreateBucket(c fiber.Ctx) error {
	var input bucket.CreateBucketInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	b, err := s.bucketRepo.Create(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(b)
}

// @Summary Get a bucket
// @Description Get a bucket by its unique ID
// @Tags buckets
// @Produce json
// @Param id path string true "Bucket ID"
// @Success 200 {object} bucket.Bucket
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{id} [get]
func (s *Server) GetBucket(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket ID"})
	}

	b, err := s.bucketRepo.Get(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	return c.JSON(b)
}

// @Summary Update a bucket
// @Description Update a bucket's key and/or owner ID by its unique ID
// @Tags buckets
// @Accept json
// @Produce json
// @Param id path string true "Bucket ID"
// @Param body body bucket.CreateBucketInput true "Update bucket payload"
// @Success 200 {object} bucket.Bucket
// @Failure 400 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{id} [put]
func (s *Server) UpdateBucket(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket ID"})
	}

	var input bucket.CreateBucketInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	updateInput := bucket.UpdateBucketInput{
		ID:        id,
		BucketKey: input.BucketKey,
		OwnerID:   input.OwnerID,
	}

	b, err := s.bucketRepo.Update(c.Context(), updateInput)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.JSON(b)
}

// @Summary Delete a bucket
// @Description Delete a bucket by its unique ID
// @Tags buckets
// @Produce json
// @Param id path string true "Bucket ID"
// @Success 204 "No Content"
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{id} [delete]
func (s *Server) DeleteBucket(c fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket ID"})
	}

	if err := s.bucketRepo.Delete(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
