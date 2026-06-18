package bucket

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

var bucketKeyRegex = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)

func isValidBucketKey(key string) bool {
	return bucketKeyRegex.MatchString(key)
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(ctx context.Context, input CreateBucketInput) (*Bucket, error) {
	if input.BucketKey == "" {
		return nil, errors.New("bucket key cannot be empty")
	}
	if !isValidBucketKey(input.BucketKey) {
		return nil, errors.New("bucket key can only contain alphanumeric characters and dashes")
	}
	if input.OwnerID == "" {
		return nil, errors.New("owner ID cannot be empty")
	}

	bucket := &Bucket{
		ID:        uuid.New().String(),
		BucketKey: input.BucketKey,
		OwnerID:   input.OwnerID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	query := `INSERT INTO buckets (id, bucket_key, owner_id, created_at, updated_at) VALUES (:id, :bucket_key, :owner_id, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*Bucket, error) {
	var bucket Bucket
	err := r.db.GetContext(ctx, &bucket, "SELECT id, bucket_key, owner_id, created_at, updated_at FROM buckets WHERE id = ?", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bucket, nil
}

func (r *Repository) GetByKey(ctx context.Context, key string) (*Bucket, error) {
	var bucket Bucket
	err := r.db.GetContext(ctx, &bucket, "SELECT id, bucket_key, owner_id, created_at, updated_at FROM buckets WHERE bucket_key = ?", key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &bucket, nil
}

func (r *Repository) Update(ctx context.Context, input UpdateBucketInput) (*Bucket, error) {
	if input.ID == "" {
		return nil, errors.New("bucket ID cannot be empty")
	}
	if input.BucketKey == "" {
		return nil, errors.New("bucket key cannot be empty")
	}
	if !isValidBucketKey(input.BucketKey) {
		return nil, errors.New("bucket key can only contain alphanumeric characters and dashes")
	}
	if input.OwnerID == "" {
		return nil, errors.New("owner ID cannot be empty")
	}

	bucket, err := r.Get(ctx, input.ID)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, errors.New("bucket not found")
	}

	bucket.BucketKey = input.BucketKey
	bucket.OwnerID = input.OwnerID
	bucket.UpdatedAt = time.Now()

	query := `UPDATE buckets SET bucket_key = :bucket_key, owner_id = :owner_id, updated_at = :updated_at WHERE id = :id`
	_, err = r.db.NamedExecContext(ctx, query, bucket)
	if err != nil {
		return nil, err
	}

	return bucket, nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("bucket ID cannot be empty")
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM buckets WHERE id = ?", id)
	return err
}
