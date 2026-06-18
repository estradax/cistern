package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDriver(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cistern-storage-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	driver, err := NewLocalDriver(tempDir)
	if err != nil {
		t.Fatalf("failed to create local driver: %v", err)
	}

	ctx := context.Background()
	content := []byte("hello, world")
	path := "testbucket/hello.txt"

	written, err := driver.Put(ctx, path, bytes.NewReader(content))
	if err != nil {
		t.Fatalf("failed to put file: %v", err)
	}
	if written != int64(len(content)) {
		t.Errorf("expected written bytes %d, got %d", len(content), written)
	}

	physicalPath := filepath.Join(tempDir, path)
	if _, err := os.Stat(physicalPath); err != nil {
		t.Errorf("physical file does not exist: %v", err)
	}

	reader, err := driver.Get(ctx, path)
	if err != nil {
		t.Fatalf("failed to get file: %v", err)
	}
	defer reader.Close()

	readContent, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("failed to read file content: %v", err)
	}
	if string(readContent) != string(content) {
		t.Errorf("expected content %q, got %q", content, readContent)
	}

	err = driver.Delete(ctx, path)
	if err != nil {
		t.Fatalf("failed to delete file: %v", err)
	}

	if _, err := os.Stat(physicalPath); !os.IsNotExist(err) {
		t.Errorf("expected physical file to be deleted, stat err: %v", err)
	}

	_, err = driver.resolvePath("../outside.txt")
	if err == nil {
		t.Error("expected error when path traverses outside base directory, got nil")
	}

	_, err = driver.resolvePath("/absolute/path.txt")
	if err == nil {
		t.Error("expected error when path is absolute, got nil")
	}
}
