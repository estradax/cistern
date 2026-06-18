package main

import (
	"fmt"
	"path/filepath"

	_ "github.com/estradax/cistern/internal/object"
	"github.com/gofiber/fiber/v3"
)

// @Summary Upload an object
// @Description Upload a file to a specific bucket. The bucket key is passed in the URL path. The file should be sent as multipart/form-data. An optional object key and content-type can be supplied.
// @Tags objects
// @Accept multipart/form-data
// @Produce json
// @Param bucket_key path string true "Bucket Key"
// @Param file formData file true "File to upload"
// @Param key formData string false "Object Key (if omitted, defaults to the filename)"
// @Param content_type formData string false "Content-Type (if omitted, defaults to file mime-type)"
// @Success 201 {object} object.Object
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{bucket_key}/objects [post]
func (s *Server) UploadObject(c fiber.Ctx) error {
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

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing file: " + err.Error()})
	}

	key := c.FormValue("key")
	if key == "" {
		key = file.Filename
	}

	contentType := c.FormValue("content_type")
	if contentType == "" {
		contentType = file.Header.Get("Content-Type")
	}

	src, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: "failed to open file: " + err.Error()})
	}
	defer src.Close()

	obj, err := s.objService.Upload(c.Context(), b.ID, key, contentType, src)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(obj)
}

// @Summary Get object metadata
// @Description Get metadata of an object by its unique object key
// @Tags objects
// @Produce json
// @Param key path string true "Object Key"
// @Success 200 {object} object.Object
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /objects/{key}/metadata [get]
func (s *Server) GetObjectMetadata(c fiber.Ctx) error {
	key := c.Params("*")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	obj, err := s.objService.GetByKey(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if obj == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "object not found"})
	}

	return c.JSON(obj)
}

// @Summary Get object content (Download/Stream)
// @Description Retrieve the raw payload of an object by its unique object key. Supports content disposition customization via query parameters.
// @Tags objects
// @Produce octet-stream
// @Param key path string true "Object Key"
// @Param contentDisposition query string false "Content disposition type: inline (default) or attachment"
// @Success 200 {file} file
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /objects/{key} [get]
func (s *Server) GetObjectContent(c fiber.Ctx) error {
	key := c.Params("*")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	obj, reader, err := s.objService.DownloadByKey(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: err.Error()})
	}

	c.Set(fiber.HeaderContentType, obj.ContentType)
	if obj.Size > 0 {
		c.Set(fiber.HeaderContentLength, fmt.Sprintf("%d", obj.Size))
	}

	disposition := "inline"
	if c.Query("contentDisposition") == "attachment" || c.Query("content_disposition") == "attachment" {
		disposition = "attachment"
	}
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("%s; filename=%q", disposition, filepath.Base(obj.ObjectKey)))

	return c.SendStream(reader)
}

// @Summary Delete an object
// @Description Delete an object's metadata and its physical storage by object key
// @Tags objects
// @Produce json
// @Param key path string true "Object Key"
// @Success 204 "No Content"
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /objects/{key} [delete]
func (s *Server) DeleteObject(c fiber.Ctx) error {
	key := c.Params("*")
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	if err := s.objService.DeleteByKey(c.Context(), key); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// @Summary List objects in a bucket
// @Description Get a list of all objects inside the specified bucket
// @Tags objects
// @Produce json
// @Param bucket_key path string true "Bucket Key"
// @Success 200 {array} object.Object
// @Failure 400 {object} APIError
// @Failure 500 {object} APIError
// @Router /buckets/{bucket_key}/objects [get]
func (s *Server) ListObjects(c fiber.Ctx) error {
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

	list, err := s.objService.ListByBucket(c.Context(), b.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.JSON(list)
}
