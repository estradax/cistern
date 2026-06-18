package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/jmoiron/sqlx"
)

func handleAPIKeys(ctx context.Context, db *sqlx.DB, action, payload string) error {
	repo := apikey.NewRepository(db)

	switch action {
	case "generate":
		var input apikey.CreateAPIKeyInput
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for generate: %w", err)
		}

		res, err := repo.Create(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to generate API key: %w", err)
		}
		return printJSON(res)

	case "read":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("API key ID cannot be empty")
		}

		key, err := repo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to retrieve API key: %w", err)
		}
		if key == nil {
			return fmt.Errorf("API key not found: %s", id)
		}
		return printJSON(key)

	case "delete":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("API key ID cannot be empty")
		}

		err := repo.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete API key: %w", err)
		}
		fmt.Fprintf(outWriter, `{"status":"success","deleted_id":%q}`+"\n", id)
		return nil

	default:
		printUsageAndExit()
		return nil
	}
}
