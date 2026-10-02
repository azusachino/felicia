package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/azusachino/felicia/apps/felicia-core/ports"
)

// FileBlobStore stores private media objects beneath a configured root.
type FileBlobStore struct {
	root string
}

var _ ports.BlobStore = FileBlobStore{}

// NewFileBlobStore creates a private filesystem-backed blob store rooted at root.
func NewFileBlobStore(root string) FileBlobStore {
	return FileBlobStore{root: root}
}

// Put atomically writes a blob beneath the configured root with owner-only permissions.
func (store FileBlobStore) Put(ctx context.Context, key string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	filename, err := store.path(key)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(filename), 0o700); err != nil {
		return fmt.Errorf("create media directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(filename), ".felicia-media-*")
	if err != nil {
		return fmt.Errorf("create temporary media object: %w", err)
	}
	temporaryName := temporary.Name()
	defer func() { _ = os.Remove(temporaryName) }()
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write media object: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync media object: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close media object: %w", err)
	}
	if err := os.Rename(temporaryName, filename); err != nil {
		return fmt.Errorf("install media object: %w", err)
	}
	return nil
}

// Open returns a reader for a blob beneath the configured root.
func (store FileBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	filename, err := store.path(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open media object: %w", err)
	}
	return file, nil
}

func (store FileBlobStore) path(key string) (string, error) {
	if key == "" || !filepath.IsLocal(filepath.FromSlash(key)) {
		return "", fmt.Errorf("invalid media object key")
	}
	return filepath.Join(store.root, filepath.FromSlash(key)), nil
}
