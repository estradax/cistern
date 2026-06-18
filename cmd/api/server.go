package main

import (
	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
	"github.com/gofiber/fiber/v3"
)

type APIError struct {
	Error string `json:"error"`
}

type Server struct {
	clientRepo *client.Repository
	apiKeyRepo *apikey.Repository
	bucketRepo *bucket.Repository
	objService *object.Service
}

func NewServer(
	clientRepo *client.Repository,
	apiKeyRepo *apikey.Repository,
	bucketRepo *bucket.Repository,
	objService *object.Service,
) *Server {
	return &Server{
		clientRepo: clientRepo,
		apiKeyRepo: apiKeyRepo,
		bucketRepo: bucketRepo,
		objService: objService,
	}
}

func (s *Server) AuthMiddleware(c fiber.Ctx) error {
	accessKey := c.Get("X-Cistern-Access-Key")
	secretKey := c.Get("X-Cistern-Secret-Key")

	if accessKey == "" || secretKey == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(APIError{Error: "missing authorization keys"})
	}

	keyRecord, err := s.apiKeyRepo.GetByAccessKey(c.Context(), accessKey)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(APIError{Error: err.Error()})
	}
	if keyRecord == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(APIError{Error: "invalid access key or secret key"})
	}

	if !apikey.VerifySecretKey(secretKey, keyRecord.SecretKeyHash) {
		return c.Status(fiber.StatusUnauthorized).JSON(APIError{Error: "invalid access key or secret key"})
	}

	c.Locals("client_id", keyRecord.ClientID)
	return c.Next()
}

