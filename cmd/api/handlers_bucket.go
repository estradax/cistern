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
// @Description Get a bucket by its unique key
// @Tags buckets
// @Produce json
// @Param bucket_key path string true "Bucket Key"
// @Success 200 {object} bucket.Bucket
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{bucket_key} [get]
func (s *Server) GetBucket(c fiber.Ctx) error {
	bucketKey := c.Params("bucket_key")
	if bucketKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket key"})
	}

	b, err := s.bucketRepo.GetByKey(c.Context(), bucketKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	return c.JSON(b)
}

// @Summary Update a bucket
// @Description Update a bucket's key and/or owner ID by its unique key
// @Tags buckets
// @Accept json
// @Produce json
// @Param bucket_key path string true "Bucket Key"
// @Param body body bucket.CreateBucketInput true "Update bucket payload"
// @Success 200 {object} bucket.Bucket
// @Failure 400 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{bucket_key} [put]
func (s *Server) UpdateBucket(c fiber.Ctx) error {
	bucketKey := c.Params("bucket_key")
	if bucketKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket key"})
	}

	var input bucket.CreateBucketInput
	if err := c.Bind().Body(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid request body: " + err.Error()})
	}

	b, err := s.bucketRepo.GetByKey(c.Context(), bucketKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	updateInput := bucket.UpdateBucketInput{
		ID:        b.ID,
		BucketKey: input.BucketKey,
		OwnerID:   input.OwnerID,
	}

	updatedBucket, err := s.bucketRepo.Update(c.Context(), updateInput)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.JSON(updatedBucket)
}

// @Summary Delete a bucket
// @Description Delete a bucket by its unique key
// @Tags buckets
// @Produce json
// @Param bucket_key path string true "Bucket Key"
// @Success 204 "No Content"
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{bucket_key} [delete]
func (s *Server) DeleteBucket(c fiber.Ctx) error {
	bucketKey := c.Params("bucket_key")
	if bucketKey == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket key"})
	}

	b, err := s.bucketRepo.GetByKey(c.Context(), bucketKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	if err := s.bucketRepo.Delete(c.Context(), b.ID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
