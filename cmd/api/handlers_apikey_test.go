package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
	"github.com/gofiber/fiber/v3"
)

func TestAPIKeyHandlers(t *testing.T) {
	env := setupTestApp(t)
	defer env.Teardown()

	testutil.CleanDatabase(t, env.DB)

	cli, err := env.ClientRepo.Create(fiber.NewDefaultCtx(nil).Context(), client.CreateClientInput{Name: "KeyOwner"})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	payload := `{"client_id":"` + cli.ID + `","name":"Production Key"}`
	req := httptest.NewRequest("POST", "/api/v1/apikeys", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var res apikey.CreateAPIKeyResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if res.APIKey.ClientID != cli.ID {
		t.Errorf("Expected client ID %q, got %q", cli.ID, res.APIKey.ClientID)
	}
	if res.SecretKey == "" {
		t.Error("Expected raw secret key to be returned")
	}

	req = httptest.NewRequest("GET", "/api/v1/apikeys/"+res.APIKey.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var kGet apikey.APIKey
	if err := json.NewDecoder(resp.Body).Decode(&kGet); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if kGet.ID != res.APIKey.ID || *kGet.Name != "Production Key" {
		t.Errorf("Mismatch in retrieved API key: %+v", kGet)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/apikeys/"+res.APIKey.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/api/v1/apikeys/"+res.APIKey.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted key, got %d", resp.StatusCode)
	}
}
