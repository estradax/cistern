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

	"github.com/estradax/cistern/internal/apikey"
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

	// Create Client A
	cliA, err := env.ClientRepo.Create(fiber.NewDefaultCtx(nil).Context(), client.CreateClientInput{Name: "ClientA"})
	if err != nil {
		t.Fatalf("Failed to create client A: %v", err)
	}

	// Create API Key for Client A
	keyA, err := env.APIKeyRepo.Create(fiber.NewDefaultCtx(nil).Context(), apikey.CreateAPIKeyInput{
		ClientID: cliA.ID,
		Name:     nil,
	})
	if err != nil {
		t.Fatalf("Failed to create API key for client A: %v", err)
	}

	// Create Client B
	cliB, err := env.ClientRepo.Create(fiber.NewDefaultCtx(nil).Context(), client.CreateClientInput{Name: "ClientB"})
	if err != nil {
		t.Fatalf("Failed to create client B: %v", err)
	}

	// Create API Key for Client B
	keyB, err := env.APIKeyRepo.Create(fiber.NewDefaultCtx(nil).Context(), apikey.CreateAPIKeyInput{
		ClientID: cliB.ID,
		Name:     nil,
	})
	if err != nil {
		t.Fatalf("Failed to create API key for client B: %v", err)
	}

	// Helper for Client A requests
	sendReqA := func(method, path string, body io.Reader, contentType string) *http.Response {
		req := httptest.NewRequest(method, path, body)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("X-Cistern-Access-Key", keyA.APIKey.AccessKey)
		req.Header.Set("X-Cistern-Secret-Key", keyA.SecretKey)
		resp, err := env.App.Test(req)
		if err != nil {
			t.Fatalf("Failed request A: %v", err)
		}
		return resp
	}

	// Helper for Client B requests
	sendReqB := func(method, path string, body io.Reader, contentType string) *http.Response {
		req := httptest.NewRequest(method, path, body)
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		req.Header.Set("X-Cistern-Access-Key", keyB.APIKey.AccessKey)
		req.Header.Set("X-Cistern-Secret-Key", keyB.SecretKey)
		resp, err := env.App.Test(req)
		if err != nil {
			t.Fatalf("Failed request B: %v", err)
		}
		return resp
	}

	// 1. Test Auth Middleware (Missing keys)
	{
		req := httptest.NewRequest("GET", "/api/v1/buckets/some-bucket", nil)
		resp, err := env.App.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for missing headers, got %d", resp.StatusCode)
		}
	}

	// 2. Test Auth Middleware (Invalid keys)
	{
		req := httptest.NewRequest("GET", "/api/v1/buckets/some-bucket", nil)
		req.Header.Set("X-Cistern-Access-Key", "invalid_access")
		req.Header.Set("X-Cistern-Secret-Key", "invalid_secret")
		resp, err := env.App.Test(req)
		if err != nil {
			t.Fatalf("Failed to perform request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for invalid keys, got %d", resp.StatusCode)
		}
	}

	// 3. Test Create Bucket for Client A
	payload := `{"bucket_key":"my-test-bucket","owner_id":"` + cliA.ID + `"}`
	resp := sendReqA("POST", "/api/v1/buckets", bytes.NewBufferString(payload), "application/json")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	var b bucket.Bucket
	if err := json.NewDecoder(resp.Body).Decode(&b); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if b.BucketKey != "my-test-bucket" || b.OwnerID != cliA.ID {
		t.Errorf("Mismatch in created bucket: %+v", b)
	}

	// 4. Test Tenant Isolation: Client B tries to GET Client A's bucket
	resp = sendReqB("GET", "/api/v1/buckets/"+b.BucketKey, nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 (Forbidden) for Client B trying to access Client A's bucket, got %d", resp.StatusCode)
	}

	// 5. Test GET bucket by key (Client A - Authorized)
	resp = sendReqA("GET", "/api/v1/buckets/"+b.BucketKey, nil, "")
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

	// 6. Test Tenant Isolation: Client B tries to UPDATE Client A's bucket
	updatePayload := `{"bucket_key":"my-updated-bucket","owner_id":"` + cliA.ID + `"}`
	resp = sendReqB("PUT", "/api/v1/buckets/"+b.BucketKey, bytes.NewBufferString(updatePayload), "application/json")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 (Forbidden) on cross-tenant PUT, got %d", resp.StatusCode)
	}

	// 7. Test UPDATE bucket (Client A - Authorized)
	resp = sendReqA("PUT", "/api/v1/buckets/"+b.BucketKey, bytes.NewBufferString(updatePayload), "application/json")
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

	// 8. Test Tenant Isolation: Client B tries to upload object to Client A's bucket
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

	// Capture the raw multipart body to reuse for client A's request
	multipartBodyBytes := bodyBuf.Bytes()

	resp = sendReqB("POST", "/api/v1/buckets/my-updated-bucket/objects", bytes.NewReader(multipartBodyBytes), mw.FormDataContentType())
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 (Forbidden) on cross-tenant UploadObject, got %d", resp.StatusCode)
	}

	// 9. Test Upload Object (Client A - Authorized)
	resp = sendReqA("POST", "/api/v1/buckets/my-updated-bucket/objects", bytes.NewReader(multipartBodyBytes), mw.FormDataContentType())
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected status 201, got %d", resp.StatusCode)
	}

	var obj object.Object
	if err := json.NewDecoder(resp.Body).Decode(&obj); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if !strings.HasPrefix(obj.ObjectKey, "documents/notes.txt") || len(obj.ObjectKey) != len("documents/notes.txt")+5 || obj.BucketID != b.ID {
		t.Errorf("Mismatch in uploaded object: %+v", obj)
	}

	// 10. Test Tenant Isolation: Client B tries to view Client A's object metadata
	resp = sendReqB("GET", "/api/v1/objects/"+obj.ObjectKey+"/metadata", nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403 on cross-tenant GET metadata, got %d", resp.StatusCode)
	}

	// 11. Test Get object metadata (Client A - Authorized)
	resp = sendReqA("GET", "/api/v1/objects/"+obj.ObjectKey+"/metadata", nil, "")
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

	// 12. Test Tenant Isolation: Client B tries to list objects in Client A's bucket
	resp = sendReqB("GET", "/api/v1/buckets/my-updated-bucket/objects", nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 on cross-tenant ListObjects, got %d", resp.StatusCode)
	}

	// 13. Test List objects in bucket (Client A - Authorized)
	resp = sendReqA("GET", "/api/v1/buckets/my-updated-bucket/objects", nil, "")
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

	// 14. Test Tenant Isolation: Client B tries to download Client A's object content
	resp = sendReqB("GET", "/api/v1/objects/"+obj.ObjectKey, nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 on cross-tenant GET object content, got %d", resp.StatusCode)
	}

	// 15. Test default GET /objects/{key} (inline content disposition, Client A - Authorized)
	resp = sendReqA("GET", "/api/v1/objects/"+obj.ObjectKey, nil, "")
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

	// 16. Test GET /objects/{key}?contentDisposition=attachment (attachment content disposition, Client A - Authorized)
	resp = sendReqA("GET", "/api/v1/objects/"+obj.ObjectKey+"?contentDisposition=attachment", nil, "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	cdAttachment := resp.Header.Get("Content-Disposition")
	if cdAttachment != `attachment; filename="`+expectedFilename+`"` {
		t.Errorf("Expected Content-Disposition to be 'attachment; filename=\"%s\"', got %q", expectedFilename, cdAttachment)
	}

	// 17. Test Tenant Isolation: Client B tries to DELETE Client A's object
	resp = sendReqB("DELETE", "/api/v1/objects/"+obj.ObjectKey, nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 on cross-tenant DELETE object, got %d", resp.StatusCode)
	}

	// 18. Test DELETE object (Client A - Authorized)
	resp = sendReqA("DELETE", "/api/v1/objects/"+obj.ObjectKey, nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204, got %d", resp.StatusCode)
	}

	// Verify delete
	resp = sendReqA("GET", "/api/v1/objects/"+obj.ObjectKey, nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted object content, got %d", resp.StatusCode)
	}

	resp = sendReqA("GET", "/api/v1/objects/"+obj.ObjectKey+"/metadata", nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted object metadata, got %d", resp.StatusCode)
	}

	// 19. Test Tenant Isolation: Client B tries to DELETE Client A's bucket
	resp = sendReqB("DELETE", "/api/v1/buckets/my-updated-bucket", nil, "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected 403 on cross-tenant DELETE bucket, got %d", resp.StatusCode)
	}

	// 20. Test DELETE bucket (Client A - Authorized)
	resp = sendReqA("DELETE", "/api/v1/buckets/my-updated-bucket", nil, "")
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status 204 on delete, got %d", resp.StatusCode)
	}

	// Verify delete
	resp = sendReqA("GET", "/api/v1/buckets/my-updated-bucket", nil, "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for deleted bucket, got %d", resp.StatusCode)
	}
}
