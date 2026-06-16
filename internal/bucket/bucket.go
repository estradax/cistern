package bucket

import "time"

type Bucket struct {
	ID        string    `db:"id" json:"id"`
	BucketKey string    `db:"bucket_key" json:"bucket_key"`
	OwnerID   string    `db:"owner_id" json:"owner_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CreateBucketInput struct {
	BucketKey string `json:"bucket_key"`
	OwnerID   string `json:"owner_id"`
}

type UpdateBucketInput struct {
	ID        string `json:"id"`
	BucketKey string `json:"bucket_key"`
	OwnerID   string `json:"owner_id"`
}
