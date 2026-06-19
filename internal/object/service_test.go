package object

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/storage"
	"github.com/estradax/cistern/internal/testutil"
)

func TestObjectServiceAndRepository(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	testutil.CleanDatabase(t, db)

	clientRepo := client.NewRepository(db)
	bucketRepo := bucket.NewRepository(db)
	objRepo := NewRepository(db)

	tempDir, err := os.MkdirTemp("", "cistern-object-service-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := storage.NewLocalDriver(tempDir)
	if err != nil {
		t.Fatalf("failed to create storage driver: %v", err)
	}

	service := NewService(objRepo, store, "test-secret")
	ctx := context.Background()

	c, err := clientRepo.Create(ctx, client.CreateClientInput{Name: "Test Client"})
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	b, err := bucketRepo.Create(ctx, bucket.CreateBucketInput{
		BucketKey: "test-bucket",
		OwnerID:   c.ID,
	})
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	objectKey := "documents/notes.txt"
	contentType := "text/plain"
	content := []byte("Cistern object storage test payload.")

	obj, err := service.Upload(ctx, b.ID, objectKey, contentType, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to upload object: %v", err)
	}

	if obj.ID == "" {
		t.Error("expected non-empty object ID")
	}
	if obj.BucketID != b.ID {
		t.Errorf("expected bucket ID %s, got %s", b.ID, obj.BucketID)
	}
	
	if !strings.HasPrefix(obj.ObjectKey, "documents/notes.txt") || len(obj.ObjectKey) != len("documents/notes.txt")+5 {
		t.Errorf("expected key to have prefix %q and suffix of 5 characters, got %q", "documents/notes.txt", obj.ObjectKey)
	}
	if obj.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), obj.Size)
	}
	if obj.ContentType != contentType {
		t.Errorf("expected content type %s, got %s", contentType, obj.ContentType)
	}
	if obj.ETag == "" {
		t.Error("expected non-empty ETag")
	}

	retrievedObj, err := service.Get(ctx, obj.ID)
	if err != nil {
		t.Fatalf("failed to get object by ID: %v", err)
	}
	if retrievedObj == nil {
		t.Fatal("expected object to be retrieved, got nil")
	}
	if retrievedObj.ObjectKey != obj.ObjectKey {
		t.Errorf("expected key %s, got %s", obj.ObjectKey, retrievedObj.ObjectKey)
	}

	retrievedObjByKey, err := service.GetByBucketAndKey(ctx, b.ID, obj.ObjectKey)
	if err != nil {
		t.Fatalf("failed to get object by key: %v", err)
	}
	if retrievedObjByKey == nil {
		t.Fatal("expected object to be retrieved by key, got nil")
	}
	if retrievedObjByKey.ID != obj.ID {
		t.Errorf("expected object ID %s, got %s", obj.ID, retrievedObjByKey.ID)
	}

	retrievedObjByGlobalKey, err := service.GetByKey(ctx, obj.ObjectKey)
	if err != nil {
		t.Fatalf("failed to get object by global key: %v", err)
	}
	if retrievedObjByGlobalKey == nil {
		t.Fatal("expected object to be retrieved by global key, got nil")
	}
	if retrievedObjByGlobalKey.ID != obj.ID {
		t.Errorf("expected object ID %s, got %s", obj.ID, retrievedObjByGlobalKey.ID)
	}

	meta, reader, err := service.DownloadByKey(ctx, obj.ObjectKey)
	if err != nil {
		t.Fatalf("failed to download object by key: %v", err)
	}
	defer reader.Close()

	if meta.ID != obj.ID {
		t.Errorf("download metadata ID mismatch: expected %s, got %s", obj.ID, meta.ID)
	}

	readBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read downloaded stream: %v", err)
	}
	if string(readBytes) != string(content) {
		t.Errorf("expected downloaded content %q, got %q", content, readBytes)
	}

	list, err := service.ListByBucket(ctx, b.ID)
	if err != nil {
		t.Fatalf("failed to list objects: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("expected list length 1, got %d", len(list))
	} else if list[0].ID != obj.ID {
		t.Errorf("listed object ID mismatch: expected %s, got %s", obj.ID, list[0].ID)
	}

	
	objectKeyWithSpaces := "my cool note 🌏.txt"
	obj2, err := service.Upload(ctx, b.ID, objectKeyWithSpaces, contentType, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to upload object with spaces/non-ASCII: %v", err)
	}
	expectedPrefix := "my-cool-note-.txt"
	if !strings.HasPrefix(obj2.ObjectKey, expectedPrefix) || len(obj2.ObjectKey) != len(expectedPrefix)+5 {
		t.Errorf("expected transformed key prefix %q and suffix of length 5, got %q", expectedPrefix, obj2.ObjectKey)
	}

	err = service.DeleteByKey(ctx, obj.ObjectKey)
	if err != nil {
		t.Fatalf("failed to delete object by key: %v", err)
	}

	deletedObj, err := service.GetByKey(ctx, obj.ObjectKey)
	if err != nil {
		t.Fatalf("error checking object deletion: %v", err)
	}
	if deletedObj != nil {
		t.Error("expected object database record to be deleted, but it still exists")
	}

	_, _, err = service.DownloadByKey(ctx, obj.ObjectKey)
	if err == nil {
		t.Error("expected download of deleted object to fail, but it succeeded")
	}

	
	t.Run("Presigned URL Signature Verification", func(t *testing.T) {
		baseURL := "http://localhost:3000"
		bucketKey := "test-bucket"
		objectKey := "documents/notes.txt"
		expiresIn := int64(60)

		
		getURLStr, err := service.GeneratePresignedURL(baseURL, "GET", "", objectKey, expiresIn)
		if err != nil {
			t.Fatalf("failed to generate GET presigned URL: %v", err)
		}

		
		parsedURL, err := url.Parse(getURLStr)
		if err != nil {
			t.Fatalf("failed to parse generated URL: %v", err)
		}
		
		q := parsedURL.Query()
		expiresStr := q.Get("expires")
		sig := q.Get("signature")
		
		var expires int64
		_, err = fmt.Sscan(expiresStr, &expires)
		if err != nil {
			t.Fatalf("failed to parse expires timestamp: %v", err)
		}

		if !service.VerifyPresignedURL("GET", "", objectKey, expires, sig) {
			t.Error("expected valid GET presigned URL signature to verify successfully")
		}

		if service.VerifyPresignedURL("GET", "", "other-key.txt", expires, sig) {
			t.Error("expected signature verification to fail for mismatched object key")
		}

		if service.VerifyPresignedURL("GET", "", objectKey, expires-100, sig) {
			t.Error("expected signature verification to fail for expired timestamp")
		}

		if service.VerifyPresignedURL("POST", bucketKey, objectKey, expires, sig) {
			t.Error("expected signature verification to fail for mismatched method")
		}

		
		postURLStr, err := service.GeneratePresignedURL(baseURL, "POST", bucketKey, objectKey, expiresIn)
		if err != nil {
			t.Fatalf("failed to generate POST presigned URL: %v", err)
		}

		parsedPostURL, err := url.Parse(postURLStr)
		if err != nil {
			t.Fatalf("failed to parse POST URL: %v", err)
		}

		qPost := parsedPostURL.Query()
		postExpiresStr := qPost.Get("expires")
		postSig := qPost.Get("signature")
		postBucketKey := qPost.Get("bucket_key")

		if postBucketKey != bucketKey {
			t.Errorf("expected bucket_key query param %q, got %q", bucketKey, postBucketKey)
		}

		var postExpires int64
		_, _ = fmt.Sscan(postExpiresStr, &postExpires)

		if !service.VerifyPresignedURL("POST", bucketKey, objectKey, postExpires, postSig) {
			t.Error("expected valid POST presigned URL signature to verify successfully")
		}
	})
}
