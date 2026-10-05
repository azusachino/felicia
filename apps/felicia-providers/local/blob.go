package local

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"

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
	if _, err := store.path(key); err != nil {
		return err
	}
	if err := os.MkdirAll(store.root, 0o700); err != nil {
		return fmt.Errorf("create media root: %w", err)
	}
	root, err := os.OpenRoot(store.root)
	if err != nil {
		return fmt.Errorf("open media root: %w", err)
	}
	defer func() { _ = root.Close() }()
	key = filepath.FromSlash(key)
	if err := root.MkdirAll(filepath.Dir(key), 0o700); err != nil {
		return fmt.Errorf("create media directory: %w", err)
	}
	temporaryName := filepath.Join(filepath.Dir(key), ".felicia-media-"+uuid.NewString())
	temporary, err := root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary media object: %w", err)
	}
	defer func() { _ = root.Remove(temporaryName) }()
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
	if err := root.Rename(temporaryName, key); err != nil {
		return fmt.Errorf("install media object: %w", err)
	}
	return nil
}

// Delete removes only a confined object; an already absent object is harmless.
func (store FileBlobStore) Delete(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := store.path(key); err != nil {
		return err
	}
	root, err := os.OpenRoot(store.root)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	if err := root.Remove(filepath.FromSlash(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Open returns a reader for a blob beneath the configured root.
func (store FileBlobStore) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if _, err := store.path(key); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(store.root)
	if err != nil {
		return nil, fmt.Errorf("open media root: %w", err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(filepath.FromSlash(key))
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
