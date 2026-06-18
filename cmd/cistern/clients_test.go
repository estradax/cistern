package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
)

func TestHandleClients_CreateAndRead(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	ctx := context.Background()

	defer func() {
		outWriter = os.Stdout
	}()

	t.Run("create client successfully", func(t *testing.T) {
		testutil.CleanDatabase(t, db)

		var buf bytes.Buffer
		outWriter = &buf

		payload := `{"name": "Test Client"}`
		err := handleClients(ctx, db, "create", payload)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		output := buf.String()
		var created client.Client
		if err := json.Unmarshal([]byte(output), &created); err != nil {
			t.Fatalf("Failed to parse output JSON: %v. Output was: %q", err, output)
		}

		if created.Name != "Test Client" {
			t.Errorf("Expected client name to be 'Test Client', got %q", created.Name)
		}
		if created.ID == "" {
			t.Errorf("Expected client ID to be non-empty")
		}

		t.Run("read client successfully", func(t *testing.T) {
			buf.Reset()
			readPayload := `{"id": "` + created.ID + `"}`
			err := handleClients(ctx, db, "read", readPayload)
			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			outputRead := buf.String()
			var retrieved client.Client
			if err := json.Unmarshal([]byte(outputRead), &retrieved); err != nil {
				t.Fatalf("Failed to parse read output JSON: %v. Output was: %q", err, outputRead)
			}

			if retrieved.ID != created.ID || retrieved.Name != created.Name {
				t.Errorf("Retrieved client doesn't match created client. Got %+v, want %+v", retrieved, created)
			}
		})
	})

	t.Run("invalid JSON payload error", func(t *testing.T) {
		err := handleClients(ctx, db, "create", `invalid-json`)
		if err == nil {
			t.Fatal("Expected error due to invalid JSON, got nil")
		}
		if !strings.Contains(err.Error(), "invalid JSON payload") {
			t.Errorf("Expected error to mention 'invalid JSON payload', got: %v", err)
		}
	})
}
