package object

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/estradax/cistern/internal/storage"
	"github.com/google/uuid"
)

type Service struct {
	repo          *Repository
	storage       storage.Driver
	presignSecret string
}

func NewService(repo *Repository, store storage.Driver, presignSecret string) *Service {
	return &Service{
		repo:          repo,
		storage:       store,
		presignSecret: presignSecret,
	}
}

func (s *Service) Upload(ctx context.Context, bucketID string, key string, contentType string, reader io.Reader) (*Object, error) {
	fileUUID := uuid.New().String()
	storagePath := fmt.Sprintf("%s/%s", bucketID, fileUUID)

	hash := md5.New()
	teeReader := io.TeeReader(reader, hash)

	size, err := s.storage.Put(ctx, storagePath, teeReader)
	if err != nil {
		return nil, fmt.Errorf("failed to store object payload: %w", err)
	}

	etag := fmt.Sprintf("%x", hash.Sum(nil))

	if contentType == "" {
		contentType = "application/octet-stream"
	}

	obj, err := s.repo.Create(ctx, CreateObjectInput{
		BucketID:    bucketID,
		ObjectKey:   key,
		Size:        size,
		ContentType: contentType,
		ETag:        etag,
		StoragePath: storagePath,
	})
	if err != nil {
		_ = s.storage.Delete(ctx, storagePath)
		return nil, fmt.Errorf("failed to create object record: %w", err)
	}

	return obj, nil
}

func (s *Service) Download(ctx context.Context, id string) (*Object, io.ReadCloser, error) {
	obj, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if obj == nil {
		return nil, nil, fmt.Errorf("object not found: %s", id)
	}

	reader, err := s.storage.Get(ctx, obj.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve object payload: %w", err)
	}

	return obj, reader, nil
}

func (s *Service) GetByBucketAndKey(ctx context.Context, bucketID string, key string) (*Object, error) {
	return s.repo.GetByBucketAndKey(ctx, bucketID, key)
}

func (s *Service) GetByKey(ctx context.Context, key string) (*Object, error) {
	return s.repo.GetByKey(ctx, key)
}

func (s *Service) Get(ctx context.Context, id string) (*Object, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) ListByBucket(ctx context.Context, bucketID string) ([]Object, error) {
	return s.repo.ListByBucket(ctx, bucketID)
}

func (s *Service) DownloadByKey(ctx context.Context, key string) (*Object, io.ReadCloser, error) {
	obj, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return nil, nil, err
	}
	if obj == nil {
		return nil, nil, fmt.Errorf("object not found: %s", key)
	}

	reader, err := s.storage.Get(ctx, obj.StoragePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to retrieve object payload: %w", err)
	}

	return obj, reader, nil
}

func (s *Service) DeleteByKey(ctx context.Context, key string) error {
	obj, err := s.repo.GetByKey(ctx, key)
	if err != nil {
		return err
	}
	if obj == nil {
		return nil
	}

	if err := s.repo.Delete(ctx, obj.ID); err != nil {
		return fmt.Errorf("failed to delete object record: %w", err)
	}

	if err := s.storage.Delete(ctx, obj.StoragePath); err != nil {
		return fmt.Errorf("failed to delete object physical payload: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	obj, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}
	if obj == nil {
		return nil
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete object record: %w", err)
	}

	if err := s.storage.Delete(ctx, obj.StoragePath); err != nil {
		return fmt.Errorf("failed to delete object physical payload: %w", err)
	}

	return nil
}

func (s *Service) GeneratePresignedURL(baseURL, method, bucketKey, objectKey string, expiresInSeconds int64) (string, error) {
	if expiresInSeconds <= 0 {
		expiresInSeconds = 3600
	}
	expires := time.Now().Add(time.Duration(expiresInSeconds) * time.Second).Unix()
	sig := GenerateSignature(s.presignSecret, method, bucketKey, objectKey, expires)

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	u.Path = fmt.Sprintf("/api/v1/presigned/objects/%s", objectKey)

	q := u.Query()
	if method == "POST" {
		q.Set("bucket_key", bucketKey)
	}
	q.Set("expires", fmt.Sprintf("%d", expires))
	q.Set("signature", sig)
	u.RawQuery = q.Encode()

	return u.String(), nil
}

func (s *Service) VerifyPresignedURL(method, bucketKey, objectKey string, expires int64, signature string) bool {
	return VerifySignature(s.presignSecret, method, bucketKey, objectKey, expires, signature)
}
