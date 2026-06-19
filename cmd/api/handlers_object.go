package main

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/estradax/cistern/internal/object"
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
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
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

	clientID := c.Locals("client_id").(string)
	if b.OwnerID != clientID {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this bucket"})
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
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
// @Router /objects/{key}/metadata [get]
func (s *Server) GetObjectMetadata(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
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

	b, err := s.bucketRepo.Get(c.Context(), obj.BucketID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	clientID := c.Locals("client_id").(string)
	if b.OwnerID != clientID {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this object"})
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
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
// @Router /objects/{key} [get]
func (s *Server) GetObjectContent(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	obj, reader, err := s.objService.DownloadByKey(c.Context(), key)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: err.Error()})
	}

	b, err := s.bucketRepo.Get(c.Context(), obj.BucketID)
	if err != nil {
		reader.Close()
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		reader.Close()
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	clientID := c.Locals("client_id").(string)
	if b.OwnerID != clientID {
		reader.Close()
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this object"})
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
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
// @Router /objects/{key} [delete]
func (s *Server) DeleteObject(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
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

	b, err := s.bucketRepo.Get(c.Context(), obj.BucketID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	clientID := c.Locals("client_id").(string)
	if b.OwnerID != clientID {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this object"})
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
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
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

	clientID := c.Locals("client_id").(string)
	if b.OwnerID != clientID {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this bucket"})
	}

	list, err := s.objService.ListByBucket(c.Context(), b.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}

	return c.JSON(list)
}

// @Summary Generate a presigned URL
// @Description Generate a presigned URL for downloading or uploading a specific object key.
// @Tags objects
// @Accept json
// @Produce json
// @Param key path string true "Object Key"
// @Param body body object.GeneratePresignedURLInput true "Presign parameters"
// @Success 200 {object} object.GeneratePresignedURLResponse
// @Failure 400 {object} APIError
// @Failure 401 {object} APIError
// @Failure 403 {object} APIError
// @Failure 500 {object} APIError
// @Security AccessKey
// @Security SecretKey
// @Router /objects/{key}/presign [post]
func (s *Server) GeneratePresignedURL(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	var input object.GeneratePresignedURLInput
	if err := c.Bind().JSON(&input); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid JSON payload: " + err.Error()})
	}

	if input.Method == "" {
		input.Method = "GET"
	}
	if input.Method != "GET" && input.Method != "PUT" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid method: only GET and PUT are supported"})
	}

	clientID := c.Locals("client_id").(string)

	var bucketKey string
	if input.Method == "PUT" {
		if input.BucketKey == "" {
			return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "bucket_key is required for PUT method"})
		}
		bucketKey = input.BucketKey
		
		b, err := s.bucketRepo.GetByKey(c.Context(), bucketKey)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
		}
		if b == nil {
			return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
		}
		if b.OwnerID != clientID {
			return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this bucket"})
		}
	} else {
		obj, err := s.objService.GetByKey(c.Context(), key)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
		}
		if obj == nil {
			return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "object not found"})
		}
		b, err := s.bucketRepo.Get(c.Context(), obj.BucketID)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
		}
		if b == nil {
			return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
		}
		if b.OwnerID != clientID {
			return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "access denied to this object"})
		}
	}

	baseURL := fmt.Sprintf("%s://%s", c.Protocol(), c.Host())
	presignedURL, err := s.objService.GeneratePresignedURL(baseURL, input.Method, bucketKey, key, input.ExpiresIn)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: "failed to generate presigned URL: " + err.Error()})
	}

	return c.JSON(object.GeneratePresignedURLResponse{
		URL: presignedURL,
	})
}

// @Summary Download object using presigned URL
// @Description Retrieve object content without headers by verifying the signature and expiration parameter.
// @Tags objects
// @Produce octet-stream
// @Param key path string true "Object Key"
// @Param expires query int true "Expires timestamp"
// @Param signature query string true "HMAC Signature"
// @Success 200 {file} file
// @Failure 400 {object} APIError
// @Failure 403 {object} APIError
// @Failure 404 {object} APIError
// @Failure 500 {object} APIError
// @Router /presigned/objects/{key} [get]
func (s *Server) GetPresignedObjectContent(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	expiresStr := c.Query("expires")
	signature := c.Query("signature")
	if expiresStr == "" || signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing expires or signature query parameter"})
	}

	var expires int64
	if _, err := fmt.Sscan(expiresStr, &expires); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid expires parameter"})
	}

	if !s.objService.VerifyPresignedURL("GET", "", key, expires, signature) {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "invalid or expired presigned URL"})
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

// @Summary Upload object using presigned URL
// @Description Upload object content without headers by verifying the signature and expiration parameter.
// @Tags objects
// @Accept octet-stream
// @Produce json
// @Param key path string true "Object Key"
// @Param bucket_key query string true "Bucket Key"
// @Param expires query int true "Expires timestamp"
// @Param signature query string true "HMAC Signature"
// @Success 201 {object} object.Object
// @Failure 400 {object} APIError
// @Failure 430 {object} APIError
// @Failure 500 {object} APIError
// @Router /presigned/objects/{key} [put]
func (s *Server) UploadPresignedObjectContent(c fiber.Ctx) error {
	key, err := url.PathUnescape(c.Params("*"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid object key encoding: " + err.Error()})
	}
	if key == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing object key"})
	}

	bucketKey := c.Query("bucket_key")
	expiresStr := c.Query("expires")
	signature := c.Query("signature")
	if bucketKey == "" || expiresStr == "" || signature == "" {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "missing bucket_key, expires, or signature parameter"})
	}

	var expires int64
	if _, err := fmt.Sscan(expiresStr, &expires); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: "invalid expires parameter"})
	}

	if !s.objService.VerifyPresignedURL("PUT", bucketKey, key, expires, signature) {
		return c.Status(fiber.StatusForbidden).JSON(APIError{Error: "invalid or expired presigned URL"})
	}

	b, err := s.bucketRepo.GetByKey(c.Context(), bucketKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if b == nil {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "bucket not found"})
	}

	contentType := c.Get(fiber.HeaderContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	bodyReader := bytes.NewReader(c.Body())

	obj, err := s.objService.Upload(c.Context(), b.ID, key, contentType, bodyReader)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(APIError{Error: err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(obj)
}

