package bucket_test

import (
	"context"
	"testing"
	"time"

	"github.com/estradax/cistern/internal/bucket"
	"github.com/estradax/cistern/internal/client"
	"github.com/estradax/cistern/internal/testutil"
)

func TestBucketRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("successful creation", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		bucketRepo := bucket.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, err := clientRepo.Create(ctx, client.CreateClientInput{Name: "Acme Corp"})
		if err != nil {
			t.Fatalf("failed to create client: %v", err)
		}

		b, err := bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "my-bucket",
			OwnerID:   c.ID,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if b.ID == "" {
			t.Fatal("expected bucket ID to be generated")
		}
		if b.BucketKey != "my-bucket" {
			t.Errorf("expected bucket key 'my-bucket', got %q", b.BucketKey)
		}
		if b.OwnerID != c.ID {
			t.Errorf("expected owner ID %q, got %q", c.ID, b.OwnerID)
		}
	})

	t.Run("validation errors", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		bucketRepo := bucket.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "",
			OwnerID:   "some-owner",
		})
		if err == nil {
			t.Error("expected error for empty bucket key, got nil")
		}

		_, err = bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "some-bucket",
			OwnerID:   "",
		})
		if err == nil {
			t.Error("expected error for empty owner ID, got nil")
		}

		// Test invalid bucket keys
		invalidKeys := []string{
			"my bucket",
			"my_bucket",
			"my.bucket",
			"my#bucket",
			"my/bucket",
			"bucket!",
		}
		for _, key := range invalidKeys {
			_, err = bucketRepo.Create(ctx, bucket.CreateBucketInput{
				BucketKey: key,
				OwnerID:   "some-owner",
			})
			if err == nil {
				t.Errorf("expected error for bucket key %q containing invalid character, got nil", key)
			} else if err.Error() != "bucket key can only contain alphanumeric characters and dashes" {
				t.Errorf("expected validation error message, got: %v", err)
			}
		}
	})

	t.Run("foreign key violation", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		bucketRepo := bucket.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "my-bucket",
			OwnerID:   "non-existent-client-id",
		})
		if err == nil {
			t.Error("expected foreign key violation error, got nil")
		}
	})
}

func TestBucketRepository_GetAndUpdate(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	testutil.CleanDatabase(t, db)
	clientRepo := client.NewRepository(db)
	bucketRepo := bucket.NewRepository(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c1, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client 1"})
	c2, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client 2"})

	b, err := bucketRepo.Create(ctx, bucket.CreateBucketInput{
		BucketKey: "original-key",
		OwnerID:   c1.ID,
	})
	if err != nil {
		t.Fatalf("failed to create bucket: %v", err)
	}

	t.Run("get bucket by ID", func(t *testing.T) {
		fetched, err := bucketRepo.Get(ctx, b.ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if fetched == nil {
			t.Fatal("expected bucket, got nil")
		}
		if fetched.BucketKey != "original-key" {
			t.Errorf("expected bucket key 'original-key', got %q", fetched.BucketKey)
		}
	})

	t.Run("update bucket", func(t *testing.T) {
		updated, err := bucketRepo.Update(ctx, bucket.UpdateBucketInput{
			ID:        b.ID,
			BucketKey: "updated-key",
			OwnerID:   c2.ID,
		})
		if err != nil {
			t.Fatalf("expected no error on update, got %v", err)
		}
		if updated.BucketKey != "updated-key" {
			t.Errorf("expected updated key 'updated-key', got %q", updated.BucketKey)
		}
		if updated.OwnerID != c2.ID {
			t.Errorf("expected updated owner ID %q, got %q", c2.ID, updated.OwnerID)
		}

		fetched, _ := bucketRepo.Get(ctx, b.ID)
		if fetched.BucketKey != "updated-key" {
			t.Errorf("expected database key to be 'updated-key', got %q", fetched.BucketKey)
		}
	})

	t.Run("update bucket with invalid key", func(t *testing.T) {
		_, err := bucketRepo.Update(ctx, bucket.UpdateBucketInput{
			ID:        b.ID,
			BucketKey: "invalid_key_with_underscore",
			OwnerID:   c2.ID,
		})
		if err == nil {
			t.Error("expected error updating bucket with invalid bucket key, got nil")
		} else if err.Error() != "bucket key can only contain alphanumeric characters and dashes" {
			t.Errorf("expected validation error, got: %v", err)
		}
	})
}

func TestBucketRepository_DeleteAndRestrictions(t *testing.T) {
	db := testutil.SetupTestDB(t)
	defer db.Close()

	t.Run("delete bucket", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		bucketRepo := bucket.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client"})
		b, _ := bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "bucket-to-delete",
			OwnerID:   c.ID,
		})

		err := bucketRepo.Delete(ctx, b.ID)
		if err != nil {
			t.Fatalf("expected no error on delete, got %v", err)
		}

		fetched, _ := bucketRepo.Get(ctx, b.ID)
		if fetched != nil {
			t.Error("expected bucket to be deleted, but still exists")
		}
	})

	t.Run("restricted delete on client with buckets", func(t *testing.T) {
		testutil.CleanDatabase(t, db)
		clientRepo := client.NewRepository(db)
		bucketRepo := bucket.NewRepository(db)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		c, _ := clientRepo.Create(ctx, client.CreateClientInput{Name: "Client with Bucket"})
		_, _ = bucketRepo.Create(ctx, bucket.CreateBucketInput{
			BucketKey: "bucket-preventing-delete",
			OwnerID:   c.ID,
		})

		err := clientRepo.Delete(ctx, c.ID)
		if err == nil {
			t.Error("expected error deleting client with bucket due to ON DELETE RESTRICT constraint, got nil")
		}
	})
}
