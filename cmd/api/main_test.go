package main

import (
	"os"
	"testing"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/storage"
	"github.com/estradax/cistern/internal/testutil"
	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
)

type TestEnv struct {
	App        *fiber.App
	DB         *sqlx.DB
	ClientRepo *client.Repository
	APIKeyRepo *apikey.Repository
	BucketRepo *bucket.Repository
	ObjService *object.Service
	Teardown   func()
}

func setupTestApp(t *testing.T) *TestEnv {
	t.Helper()

	db := testutil.SetupTestDB(t)

	tempDir, err := os.MkdirTemp("", "cistern-test-storage-*")
	if err != nil {
		db.Close()
		t.Fatalf("Failed to create temp storage directory: %v", err)
	}

	store, err := storage.NewLocalDriver(tempDir)
	if err != nil {
		os.RemoveAll(tempDir)
		db.Close()
		t.Fatalf("Failed to initialize storage driver: %v", err)
	}

	clientRepo := client.NewRepository(db)
	apiKeyRepo := apikey.NewRepository(db)
	bucketRepo := bucket.NewRepository(db)
	objRepo := object.NewRepository(db)
	objService := object.NewService(objRepo, store)

	server := NewServer(clientRepo, apiKeyRepo, bucketRepo, objService)

	app := fiber.New()
	api := app.Group("/api/v1")

	api.Post("/clients", server.CreateClient)
	api.Get("/clients/:id", server.GetClient)
	api.Put("/clients/:id", server.UpdateClient)
	api.Delete("/clients/:id", server.DeleteClient)

	api.Post("/apikeys", server.GenerateAPIKey)
	api.Get("/apikeys/:id", server.GetAPIKey)
	api.Delete("/apikeys/:id", server.DeleteAPIKey)

	api.Post("/buckets", server.CreateBucket)
	api.Get("/buckets/:bucket_key", server.GetBucket)
	api.Put("/buckets/:bucket_key", server.UpdateBucket)
	api.Delete("/buckets/:bucket_key", server.DeleteBucket)

	api.Post("/buckets/:bucket_key/objects", server.UploadObject)
	api.Get("/buckets/:bucket_key/objects", server.ListObjects)

	api.Get("/objects/*/metadata", server.GetObjectMetadata)
	api.Get("/objects/*", server.GetObjectContent)
	api.Delete("/objects/*", server.DeleteObject)

	teardown := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return &TestEnv{
		App:        app,
		DB:         db,
		ClientRepo: clientRepo,
		APIKeyRepo: apiKeyRepo,
		BucketRepo: bucketRepo,
		ObjService: objService,
		Teardown:   teardown,
	}
}
