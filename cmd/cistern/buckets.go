package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/estradax/cistern/internal/bucket"
	"github.com/jmoiron/sqlx"
)

func handleBuckets(ctx context.Context, db *sqlx.DB, action, payload string) error {
	repo := bucket.NewRepository(db)

	switch action {
	case "create":
		var input bucket.CreateBucketInput
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for create: %w", err)
		}

		b, err := repo.Create(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
		return printJSON(b)

	case "read":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("bucket ID cannot be empty")
		}

		b, err := repo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to retrieve bucket: %w", err)
		}
		if b == nil {
			return fmt.Errorf("bucket not found: %s", id)
		}
		return printJSON(b)

	case "edit", "update":
		var input bucket.UpdateBucketInput
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for update: %w", err)
		}

		b, err := repo.Update(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to update bucket: %w", err)
		}
		return printJSON(b)

	case "delete":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("bucket ID cannot be empty")
		}

		err := repo.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete bucket: %w", err)
		}
		fmt.Fprintf(outWriter, `{"status":"success","deleted_id":%q}`+"\n", id)
		return nil

	default:
		printUsageAndExit()
		return nil
	}
}
