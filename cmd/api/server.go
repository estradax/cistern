package main

import (
	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
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
