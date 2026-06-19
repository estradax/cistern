package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/estradax/cistern/internal/apikey"
	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/object"
	"github.com/estradax/cistern/internal/testutil"
)

func TestObjectHandlersAndPresignedURLs(t *testing.T) {
	env := setupTestApp(t)
	defer env.Teardown()

	testutil.CleanDatabase(t, env.DB)

	ctx := context.Background()

	
	c, err := env.ClientRepo.Create(ctx, client.CreateClientInput{Name: "Acme Storage Corp"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	
	keyName := "prod-key"
	apiResult, err := env.APIKeyRepo.Create(ctx, apikey.CreateAPIKeyInput{
		ClientID: c.ID,
		Name:     &keyName,
	})
	if err != nil {
		t.Fatalf("failed to generate api key: %v", err)
	}

	
	_, err = env.BucketRepo.Create(ctx, bucket.CreateBucketInput{
		BucketKey: "test-bucket",
		OwnerID:   c.ID,
	})
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	
	setAuthHeaders := func(req *http.Request) {
		req.Header.Set("X-Cistern-Access-Key", apiResult.APIKey.AccessKey)
		req.Header.Set("X-Cistern-Secret-Key", apiResult.SecretKey)
	}

	
	t.Run("Upload and Generate Presigned GET URL", func(t *testing.T) {
		
		
		boundary := "testboundary"
		multipartBody := "--" + boundary + "\r\n" +
			"Content-Disposition: form-data; name=\"file\"; filename=\"hello.txt\"\r\n" +
			"Content-Type: text/plain\r\n\r\n" +
			"Hello Cistern Presigned GET!" +
			"\r\n--" + boundary + "--\r\n"

		req := httptest.NewRequest("POST", "/api/v1/buckets/test-bucket/objects", bytes.NewBufferString(multipartBody))
		req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
		setAuthHeaders(req)

		resp, err := env.App.Test(req)
		if err != nil {
			t.Fatalf("Failed standard upload request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("Expected standard upload status 201, got %d", resp.StatusCode)
		}

		var objRec object.Object
		if err := json.NewDecoder(resp.Body).Decode(&objRec); err != nil {
			t.Fatalf("failed to decode upload response: %v", err)
		}

		
		presignReqBody := `{"method": "GET", "expires_in": 60}`
		reqPresign := httptest.NewRequest("POST", "/api/v1/objects/"+url.PathEscape(objRec.ObjectKey)+"/presign", bytes.NewBufferString(presignReqBody))
		reqPresign.Header.Set("Content-Type", "application/json")
		setAuthHeaders(reqPresign)

		respPresign, err := env.App.Test(reqPresign)
		if err != nil {
			t.Fatalf("failed to request presigned URL: %v", err)
		}
		if respPresign.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 for presigned generation, got %d", respPresign.StatusCode)
		}

		var presignResult map[string]string
		if err := json.NewDecoder(respPresign.Body).Decode(&presignResult); err != nil {
			t.Fatalf("failed to decode presign result: %v", err)
		}

		presignedURLStr := presignResult["url"]
		if presignedURLStr == "" {
			t.Fatal("expected non-empty presigned URL")
		}

		
		parsed, err := url.Parse(presignedURLStr)
		if err != nil {
			t.Fatalf("failed to parse generated URL: %v", err)
		}

		
		reqGet := httptest.NewRequest("GET", parsed.RequestURI(), nil)
		respGet, err := env.App.Test(reqGet)
		if err != nil {
			t.Fatalf("failed to request GET via presigned URL: %v", err)
		}
		if respGet.StatusCode != http.StatusOK {
			t.Errorf("expected GET status 200, got %d", respGet.StatusCode)
		}

		bodyBytes, _ := io.ReadAll(respGet.Body)
		if string(bodyBytes) != "Hello Cistern Presigned GET!" {
			t.Errorf("expected content 'Hello Cistern Presigned GET!', got %q", string(bodyBytes))
		}

		
		badURI := parsed.RequestURI() + "tempered"
		reqBadSig := httptest.NewRequest("GET", badURI, nil)
		respBadSig, err := env.App.Test(reqBadSig)
		if err != nil {
			t.Fatalf("failed to request with bad signature: %v", err)
		}
		if respBadSig.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403 for bad signature, got %d", respBadSig.StatusCode)
		}

		
		expiredQuery := parsed.Query()
		expiredQuery.Set("expires", fmt.Sprintf("%d", time.Now().Add(-10*time.Second).Unix()))
		
		parsed.RawQuery = expiredQuery.Encode()
		reqExpired := httptest.NewRequest("GET", parsed.RequestURI(), nil)
		respExpired, err := env.App.Test(reqExpired)
		if err != nil {
			t.Fatalf("failed to request with expired url: %v", err)
		}
		if respExpired.StatusCode != http.StatusForbidden {
			t.Errorf("expected status 403 for expired URL, got %d", respExpired.StatusCode)
		}
	})

	
	t.Run("Generate Presigned POST URL and Upload Content", func(t *testing.T) {
		objectKey := "uploads/presigned-file.txt"
		presignReqBody := `{"method": "POST", "expires_in": 60, "bucket_key": "test-bucket"}`
		reqPresign := httptest.NewRequest("POST", "/api/v1/objects/"+url.PathEscape(objectKey)+"/presign", bytes.NewBufferString(presignReqBody))
		reqPresign.Header.Set("Content-Type", "application/json")
		setAuthHeaders(reqPresign)

		respPresign, err := env.App.Test(reqPresign)
		if err != nil {
			t.Fatalf("failed to request presigned URL: %v", err)
		}
		if respPresign.StatusCode != http.StatusOK {
			t.Fatalf("Expected status 200 for presigned POST generation, got %d", respPresign.StatusCode)
		}

		var presignResult map[string]string
		if err := json.NewDecoder(respPresign.Body).Decode(&presignResult); err != nil {
			t.Fatalf("failed to decode presign result: %v", err)
		}

		presignedURLStr := presignResult["url"]
		parsed, err := url.Parse(presignedURLStr)
		if err != nil {
			t.Fatalf("failed to parse generated URL: %v", err)
		}

		
		uploadContent := "This content is uploaded directly via POST presigned URL!"
		reqPost := httptest.NewRequest("POST", parsed.RequestURI(), bytes.NewBufferString(uploadContent))
		reqPost.Header.Set("Content-Type", "text/plain")
		respPost, err := env.App.Test(reqPost)
		if err != nil {
			t.Fatalf("failed to execute POST upload: %v", err)
		}
		if respPost.StatusCode != http.StatusCreated {
			t.Fatalf("expected POST upload status 201, got %d", respPost.StatusCode)
		}

		var uploadedObj object.Object
		if err := json.NewDecoder(respPost.Body).Decode(&uploadedObj); err != nil {
			t.Fatalf("failed to decode upload response: %v", err)
		}

		
		if uploadedObj.Size != int64(len(uploadContent)) {
			t.Errorf("expected uploaded object size %d, got %d", len(uploadContent), uploadedObj.Size)
		}

		
		reqDownload := httptest.NewRequest("GET", "/api/v1/objects/"+url.PathEscape(uploadedObj.ObjectKey), nil)
		setAuthHeaders(reqDownload)
		respDownload, err := env.App.Test(reqDownload)
		if err != nil {
			t.Fatalf("failed to download uploaded object: %v", err)
		}
		if respDownload.StatusCode != http.StatusOK {
			t.Fatalf("expected download status 200, got %d", respDownload.StatusCode)
		}

		downloadedContent, _ := io.ReadAll(respDownload.Body)
		if string(downloadedContent) != uploadContent {
			t.Errorf("expected downloaded content %q, got %q", uploadContent, string(downloadedContent))
		}
	})
}
