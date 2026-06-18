package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/estradax/cistern/internal/client"
	"github.com/jmoiron/sqlx"
)

func handleClients(ctx context.Context, db *sqlx.DB, action, payload string) error {
	repo := client.NewRepository(db)

	switch action {
	case "create":
		var input client.CreateClientInput
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for create: %w", err)
		}

		c, err := repo.Create(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}
		return printJSON(c)

	case "read":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("client ID cannot be empty")
		}

		c, err := repo.Get(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to retrieve client: %w", err)
		}
		if c == nil {
			return fmt.Errorf("client not found: %s", id)
		}
		return printJSON(c)

	case "update":
		var input client.UpdateClientInput
		if err := json.Unmarshal([]byte(payload), &input); err != nil {
			return fmt.Errorf("invalid JSON payload for update: %w", err)
		}

		c, err := repo.Update(ctx, input)
		if err != nil {
			return fmt.Errorf("failed to update client: %w", err)
		}
		return printJSON(c)

	case "delete":
		id := extractID(payload)
		if id == "" {
			return fmt.Errorf("client ID cannot be empty")
		}

		err := repo.Delete(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to delete client: %w", err)
		}
		fmt.Fprintf(outWriter, `{"status":"success","deleted_id":%q}`+"\n", id)
		return nil

	default:
		printUsageAndExit()
		return nil
	}
}
