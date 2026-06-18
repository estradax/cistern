package object

import (
	"bytes"
	"context"
	"io"
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

	service := NewService(objRepo, store)
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
	// Verify key transformation: all ASCII, spaces changed to '-', random 5-char suffix at the end.
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

	// Test uploading with spaces and non-ASCII characters
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
}
