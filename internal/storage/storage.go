package storage

import (
	"context"
	"io"
)

type Driver interface {
	Put(ctx context.Context, path string, src io.Reader) (int64, error)
	Get(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
}
