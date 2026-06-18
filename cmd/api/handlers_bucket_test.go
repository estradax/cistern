package main

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/testutil"
	"github.com/gofiber/fiber/v3"
)

func TestBucketAndObjectHandlers(t *testing.T) {
	env := setupTestApp(t)
	defer env.Teardown()

	testutil.CleanDatabase(t, env.DB)

	cli, err := env.ClientRepo.Create(fiber.NewDefaultCtx(nil).Context(), client.CreateClientInput{Name: "BucketOwner"})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	payload := `{"bucket_key":"my-test-bucket","owner_id":"` + cli.ID + `"}`
	req := httptest.NewRequest("POST", "/api/v1/buckets", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var b bucket.Bucket
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if b.BucketKey != "my-test-bucket" || b.OwnerID != cli.ID {
		t.Errorf("Mismatch in created bucket: %+v", b)
	}

	// Test GET bucket by key
	req = httptest.NewRequest("GET", "/api/v1/buckets/"+b.BucketKey, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var bGet bucket.Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bGet); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if bGet.ID != b.ID {
		t.Errorf("Mismatch in retrieved bucket ID: expected %s, got %s", b.ID, bGet.ID)
	}

	updatePayload := `{"bucket_key":"my-updated-bucket","owner_id":"` + cli.ID + `"}`
	req = httptest.NewRequest("PUT", "/api/v1/buckets/"+b.BucketKey, bytes.NewBufferString(updatePayload))
	req.Header.Set("Content-Type", "application/json")
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var bUpdate bucket.Bucket
	if err := json.NewDecoder(resp.Body).Decode(&bUpdate); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if bUpdate.BucketKey != "my-updated-bucket" {
		t.Errorf("Expected key 'my-updated-bucket', got %q", bUpdate.BucketKey)
	}

	bodyBuf := &bytes.Buffer{}
	mw := multipart.NewWriter(bodyBuf)
	part, err := mw.CreateFormFile("file", "notes.txt")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte("Hello, this is my note content.")); err != nil {
		t.Fatalf("Failed to write to file part: %v", err)
	}
	if err := mw.WriteField("key", "documents/notes.txt"); err != nil {
		t.Fatalf("Failed to write form field key: %v", err)
	}
	mw.Close()

	req = httptest.NewRequest("POST", "/api/v1/buckets/my-updated-bucket/objects", bodyBuf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	var obj object.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.HasPrefix(obj.ObjectKey, "documents/notes.txt") || len(obj.ObjectKey) != len("documents/notes.txt")+5 || obj.BucketID != b.ID {
		t.Errorf("Mismatch in uploaded object: %+v", obj)
	}

	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey+"/metadata", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var objGet object.Object
	if err := json.NewDecoder(resp.Body).Decode(&objGet); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if objGet.ID != obj.ID {
		t.Errorf("Mismatch in retrieved object metadata: %+v", objGet)
	}

	req = httptest.NewRequest("GET", "/api/v1/buckets/my-updated-bucket/objects", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var list []object.Object
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("Failed to decode list response: %v", err)
	}
	if len(list) != 1 || list[0].ID != obj.ID {
		t.Errorf("Expected list containing 1 object, got: %+v", list)
	}

	// Test default GET /objects/{key} (inline content disposition)
	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	cd := resp.Header.Get("Content-Disposition")
	expectedFilename := filepath.Base(obj.ObjectKey)
	if cd != `inline; filename="`+expectedFilename+`"` {
		t.Errorf("Expected Content-Disposition to be 'inline; filename=\"%s\"', got %q", expectedFilename, cd)
	}
	dlContent, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read download content: %v", err)
	}
	if string(dlContent) != "Hello, this is my note content." {
		t.Errorf("Expected downloaded content 'Hello, this is my note content.', got %q", string(dlContent))
	}

	// Test GET /objects/{key}?contentDisposition=attachment (attachment content disposition)
	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey+"?contentDisposition=attachment", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	cdAttachment := resp.Header.Get("Content-Disposition")
	if cdAttachment != `attachment; filename="`+expectedFilename+`"` {
		t.Errorf("Expected Content-Disposition to be 'attachment; filename=\"%s\"', got %q", expectedFilename, cdAttachment)
	}

	// Test GET /objects/{key}?content_disposition=attachment (alternative snake_case param name)
	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey+"?content_disposition=attachment", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	cdAttachment2 := resp.Header.Get("Content-Disposition")
	if cdAttachment2 != `attachment; filename="`+expectedFilename+`"` {
		t.Errorf("Expected Content-Disposition to be 'attachment; filename=\"%s\"', got %q", expectedFilename, cdAttachment2)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/objects/"+obj.ObjectKey, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey, nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted object content, got %d", resp.StatusCode)
	}

	req = httptest.NewRequest("GET", "/api/v1/objects/"+obj.ObjectKey+"/metadata", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted object metadata, got %d", resp.StatusCode)
	}

	// Test DELETE bucket by key
	req = httptest.NewRequest("DELETE", "/api/v1/buckets/my-updated-bucket", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204 on delete, got %d", resp.StatusCode)
	}

	// Verify delete
	req = httptest.NewRequest("GET", "/api/v1/buckets/my-updated-bucket", nil)
	resp, err = env.App.Test(req)
	if err != nil {
		t.Fatalf("Failed to perform request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted bucket, got %d", resp.StatusCode)
	}
}
