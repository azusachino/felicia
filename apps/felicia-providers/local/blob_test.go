package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestFileBlobStoreConfinesSymlinkReadsAndWrites(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	sentinel := filepath.Join(outside, "original.jpg")
	if err := os.WriteFile(sentinel, []byte("unchanged private sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "media")); err != nil {
		t.Fatal(err)
	}
	store := NewFileBlobStore(root)
	if err := store.Put(context.Background(), "media/original.jpg", []byte("must not escape")); err == nil {
		t.Fatal("write escaped through symlink")
	}
	if reader, err := store.Open(context.Background(), "media/original.jpg"); err == nil {
		_ = reader.Close()
		t.Fatal("read escaped through symlink")
	}
	if err := store.Delete(context.Background(), "media/original.jpg"); err == nil {
		t.Fatal("delete escaped through symlink")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "unchanged private sentinel" {
		t.Fatalf("sentinel changed: %q %v", data, err)
	}
}

func TestFileBlobStoreStoresPrivateObjectAndRejectsEscapingKeys(t *testing.T) {
	root := t.TempDir()
	store := NewFileBlobStore(root)
	key := "media/abc/original.jpg"
	if err := store.Put(context.Background(), key, []byte("private original")); err != nil {
		t.Fatal(err)
	}
	storedPath := filepath.Join(root, filepath.FromSlash(key))
	info, err := os.Stat(storedPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions = %o, want 600", info.Mode().Perm())
	}
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil || string(data) != "private original" {
		t.Fatalf("read = %q, err = %v", data, err)
	}

	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), key); err != nil {
		t.Fatalf("idempotent delete: %v", err)
	}
	for _, invalid := range []string{"../outside", "/absolute", "media/../../outside"} {
		if err := store.Put(context.Background(), invalid, []byte("bad")); err == nil {
			t.Errorf("Put(%q) succeeded, want invalid-key error", invalid)
		}
		if err := store.Delete(context.Background(), invalid); err == nil {
			t.Errorf("Delete(%q) accepted invalid key", invalid)
		}
	}
}
