package object

import "time"

type Object struct {
	ID          string    `db:"id" json:"id"`
	BucketID    string    `db:"bucket_id" json:"bucket_id"`
	ObjectKey   string    `db:"object_key" json:"object_key"`
	Size        int64     `db:"size" json:"size"`
	ContentType string    `db:"content_type" json:"content_type"`
	ETag        string    `db:"etag" json:"etag"`
	StoragePath string    `db:"storage_path" json:"storage_path"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

type CreateObjectInput struct {
	BucketID    string `json:"bucket_id"`
	ObjectKey   string `json:"object_key"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	ETag        string `json:"etag"`
	StoragePath string `json:"storage_path"`
}

type GeneratePresignedURLInput struct {
	Method    string `json:"method"`
	ExpiresIn int64  `json:"expires_in"`
	BucketKey string `json:"bucket_key"`
}

type GeneratePresignedURLResponse struct {
	URL string `json:"url"`
}

