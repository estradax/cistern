package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalDriver struct {
	baseDir string
}

func NewLocalDriver(baseDir string) (*LocalDriver, error) {
	if baseDir == "" {
		return nil, errors.New("base directory cannot be empty")
	}
	absDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path of base directory: %w", err)
	}
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	return &LocalDriver{baseDir: absDir}, nil
}

func (l *LocalDriver) resolvePath(path string) (string, error) {
	cleanPath := filepath.Clean(path)
	if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
		return "", errors.New("invalid storage path: must be a relative path and cannot traverse outside base directory")
	}
	fullPath := filepath.Join(l.baseDir, cleanPath)
	return fullPath, nil
}

func (l *LocalDriver) Put(ctx context.Context, path string, src io.Reader) (int64, error) {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return 0, err
	}

	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return 0, fmt.Errorf("failed to create directories for path: %w", err)
	}

	destFile, err := os.Create(fullPath)
	if err != nil {
		return 0, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if err := ctx.Err(); err != nil {
		os.Remove(fullPath)
		return 0, err
	}

	n, err := io.Copy(destFile, src)
	if err != nil {
		os.Remove(fullPath)
		return 0, fmt.Errorf("failed to copy content: %w", err)
	}

	return n, nil
}

func (l *LocalDriver) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

func (l *LocalDriver) Delete(ctx context.Context, path string) error {
	fullPath, err := l.resolvePath(path)
	if err != nil {
		return err
	}

	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}
