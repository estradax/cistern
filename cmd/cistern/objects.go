package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/storage"
	"github.com/jmoiron/sqlx"
)

func handleObjects(ctx context.Context, db *sqlx.DB, action, payload string, extraArgs []string) error {
	storageDir := os.Getenv("STORAGE_DIR")
	if storageDir == "" {
		storageDir = "./data/storage"
	}
	store, err := storage.NewLocalDriver(storageDir)
	if err != nil {
		return fmt.Errorf("failed to initialize storage driver: %w", err)
	}

	repo := object.NewRepository(db)
	svc := object.NewService(repo, store)

	switch action {
	case "upload":
		if len(extraArgs) < 1 {
			return fmt.Errorf("file path argument is required for upload. Usage: cistern objects upload '<json_payload>' <filepath>")
		}
		filePath := extraArgs[0]

		var input struct {
			BucketID    string `json:"bucket_id"`
			ObjectKey   string `json:"object_key"`
			ContentType string `json:"content_type"`
		}
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for upload: %w", err)
		}

		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open local file: %w", err)
		}
		defer file.Close()

		obj, err := svc.Upload(ctx, input.BucketID, input.ObjectKey, input.ContentType, file)
		if err != nil {
			return fmt.Errorf("failed to upload object: %w", err)
		}
		return printJSON(obj)

	case "download":
		if len(extraArgs) < 1 {
			return fmt.Errorf("destination path argument is required for download. Usage: cistern objects download '<id_or_json>' <destination_path>")
		}
		destPath := extraArgs[0]

		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("object ID cannot be empty")
		}

		_, reader, err := svc.Download(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to download object: %w", err)
		}
		defer reader.Close()

		destDir := filepath.Dir(destPath)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directories: %w", err)
		}

		destFile, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer destFile.Close()

		if _, err := io.Copy(destFile, reader); err != nil {
			return fmt.Errorf("failed to write downloaded content: %w", err)
		}

		fmt.Fprintf(outWriter, `{"status":"success","downloaded_id":%q,"path":%q}`+"\n", id, destPath)
		return nil

	case "read":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("object ID cannot be empty")
		}

		obj, err := svc.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to retrieve object: %w", err)
		}
		if obj == nil {
			return fmt.Errorf("object not found: %s", id)
		}
		return printJSON(obj)

	case "delete":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("object ID cannot be empty")
		}

		err = svc.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete object: %w", err)
		}
		fmt.Fprintf(outWriter, `{"status":"success","deleted_id":%q}`+"\n", id)
		return nil

	case "list":
		bucketID := extractID(payload)
		if bucketID == "" {
			var input struct {
				BucketID string `json:"bucket_id"`
			}
			if err := json.Unmarshal([]byte(payload), &input); err == nil {
				bucketID = input.BucketID
			}
		}
		if bucketID == "" {
			return fmt.Errorf("bucket ID cannot be empty")
		}

		list, err := svc.ListByBucket(ctx, bucketID)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}
		return printJSON(list)

	default:
		printUsageAndExit()
		return nil
	}
}
