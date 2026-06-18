package object

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomSuffix(length int) string {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumeric))))
		if err != nil {
			b[i] = alphanumeric[time.Now().UnixNano()%int64(len(alphanumeric))]
		} else {
			b[i] = alphanumeric[num.Int64()]
		}
	}
	return string(b)
}

func TransformObjectKey(key string) string {
	var sb strings.Builder
	for _, r := range key {
		if r == ' ' {
			sb.WriteRune('-')
		} else if r >= 0 && r <= 127 {
			sb.WriteRune(r)
		}
	}
	return sb.String() + generateRandomSuffix(5)
}

func (r *Repository) Create(ctx context.Context, input CreateObjectInput) (*Object, error) {
	if input.BucketID == "" {
		return nil, errors.New("bucket ID cannot be empty")
	}
	if input.ObjectKey == "" {
		return nil, errors.New("object key cannot be empty")
	}

	transformedKey := TransformObjectKey(input.ObjectKey)
	if len(transformedKey) > 500 {
		return nil, errors.New("object key length cannot exceed 500 characters")
	}
	if input.StoragePath == "" {
		return nil, errors.New("storage path cannot be empty")
	}

	obj := &Object{
		ID:          uuid.New().String(),
		BucketID:    input.BucketID,
		ObjectKey:   transformedKey,
		Size:        input.Size,
		ContentType: input.ContentType,
		ETag:        input.ETag,
		StoragePath: input.StoragePath,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `INSERT INTO objects (id, bucket_id, object_key, size, content_type, etag, storage_path, created_at, updated_at) 
	          VALUES (:id, :bucket_id, :object_key, :size, :content_type, :etag, :storage_path, :created_at, :updated_at)`
	_, err := r.db.NamedExecContext(ctx, query, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func (r *Repository) Get(ctx context.Context, id string) (*Object, error) {
	var obj Object
	query := `SELECT id, bucket_id, object_key, size, content_type, etag, storage_path, created_at, updated_at 
	          FROM objects WHERE id = ?`
	err := r.db.GetContext(ctx, &obj, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}

func (r *Repository) GetByBucketAndKey(ctx context.Context, bucketID string, key string) (*Object, error) {
	var obj Object
	query := `SELECT id, bucket_id, object_key, size, content_type, etag, storage_path, created_at, updated_at 
	          FROM objects WHERE bucket_id = ? AND object_key = ?`
	err := r.db.GetContext(ctx, &obj, query, bucketID, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}

func (r *Repository) GetByKey(ctx context.Context, key string) (*Object, error) {
	var obj Object
	query := `SELECT id, bucket_id, object_key, size, content_type, etag, storage_path, created_at, updated_at 
	          FROM objects WHERE object_key = ?`
	err := r.db.GetContext(ctx, &obj, query, key)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &obj, nil
}


func (r *Repository) Delete(ctx context.Context, id string) error {
	if id == "" {
		return errors.New("object ID cannot be empty")
	}
	_, err := r.db.ExecContext(ctx, "DELETE FROM objects WHERE id = ?", id)
	return err
}

func (r *Repository) ListByBucket(ctx context.Context, bucketID string) ([]Object, error) {
	if bucketID == "" {
		return nil, errors.New("bucket ID cannot be empty")
	}
	var objects []Object
	query := `SELECT id, bucket_id, object_key, size, content_type, etag, storage_path, created_at, updated_at 
	          FROM objects WHERE bucket_id = ? ORDER BY object_key ASC`
	err := r.db.SelectContext(ctx, &objects, query, bucketID)
	if err != nil {
		return nil, err
	}
	return objects, nil
}
