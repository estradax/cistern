package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
)

func TestClientHandlers(t *testing.T) {
	env := setupTestApp(t)
	defer env.Teardown()

	testutil.CleanDatabase(t, env.DB)

	payload := `{"name":"Acme Corp"}`
	req := httptest.NewRequest("POST", "/api/v1/clients", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var c client.Client
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if c.Name != "Acme Corp" {
		t.Errorf("Expected client name 'Acme Corp', got %q", c.Name)
	}
	if c.ID == "" {
		t.Error("Expected non-empty client ID")
	}

	req = httptest.NewRequest("GET", "/api/v1/clients/"+c.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var cGet client.Client
	if err := json.NewDecoder(resp.Body).Decode(&cGet); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if cGet.ID != c.ID || cGet.Name != "Acme Corp" {
		t.Errorf("Mismatch in retrieved client: %+v", cGet)
	}

	updatePayload := `{"name":"Acme Inc."}`
	req = httptest.NewRequest("PUT", "/api/v1/clients/"+c.ID, bytes.NewBufferString(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var cUpdate client.Client
	if err := json.NewDecoder(resp.Body).Decode(&cUpdate); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if cUpdate.Name != "Acme Inc." {
		t.Errorf("Expected client name 'Acme Inc.', got %q", cUpdate.Name)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/clients/"+c.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/api/v1/clients/"+c.ID, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted client, got %d", resp.StatusCode)
	}
}
