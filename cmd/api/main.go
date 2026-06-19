// @title Cistern API
// @version 1.0
// @description API Server for Cistern Object Storage
// @host localhost:3000
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey AccessKey
// @in header
// @name X-Cistern-Access-Key
// @description Client Access Key

// @securityDefinitions.apikey SecretKey
// @in header
// @name X-Cistern-Secret-Key
// @description Client Secret Key
package main

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/contrib/v3/swaggerui"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/storage"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN environment variable is not set")
	}

	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./data/storage"
	}

	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	store, err := storage.NewLocalDriver(storageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage driver: %v", err)
	}

	presignSecret := os.Getenv("PRESIGN_SECRET")
	if presignSecret == "" {
		log.Fatal("PRESIGN_SECRET environment variable is not set")
	}

	clientRepo := client.NewRepository(db)
	apiKeyRepo := apikey.NewRepository(db)
	bucketRepo := bucket.NewRepository(db)
	objRepo := object.NewRepository(db)
	objService := object.NewService(objRepo, store, presignSecret)

	server := NewServer(clientRepo, apiKeyRepo, bucketRepo, objService)
	app := fiber.New()

	app.Use(cors.New())

	swaggerCfg := swaggerui.Config{
		FilePath: "./docs/swagger.json",
		Path:     "swagger",
		Title:    "Cistern API Documentation",
	}
	app.Use(swaggerui.New(swaggerCfg))

	api := app.Group("/api/v1")

	api.Post("/clients", server.CreateClient)
	api.Get("/clients/:id", server.GetClient)
	api.Put("/clients/:id", server.UpdateClient)
	api.Delete("/clients/:id", server.DeleteClient)

	api.Post("/apikeys", server.GenerateAPIKey)
	api.Get("/apikeys/:id", server.GetAPIKey)
	api.Delete("/apikeys/:id", server.DeleteAPIKey)

	
	api.Get("/presigned/objects/*", server.GetPresignedObjectContent)
	api.Post("/presigned/objects/*", server.UploadPresignedObjectContent)

	auth := api.Group("", server.AuthMiddleware)

	auth.Post("/buckets", server.CreateBucket)
	auth.Get("/buckets/:bucket_key", server.GetBucket)
	auth.Put("/buckets/:bucket_key", server.UpdateBucket)
	auth.Delete("/buckets/:bucket_key", server.DeleteBucket)

	auth.Post("/buckets/:bucket_key/objects", server.UploadObject)
	auth.Get("/buckets/:bucket_key/objects", server.ListObjects)

	auth.Post("/objects/*/presign", server.GeneratePresignedURL)
	auth.Get("/objects/*/metadata", server.GetObjectMetadata)
	auth.Get("/objects/*", server.GetObjectContent)
	auth.Delete("/objects/*", server.DeleteObject)

	app.Use(func(c fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(APIError{Error: "route not found"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Cistern API Server is starting on port %s...", port)
	log.Fatal(app.Listen(":" + port))
}
