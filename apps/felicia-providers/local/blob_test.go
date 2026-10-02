package local

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
)

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

	for _, invalid := range []string{"../outside", "/absolute", "media/../../outside"} {
		if err := store.Put(context.Background(), invalid, []byte("bad")); err == nil {
			t.Errorf("Put(%q) succeeded, want invalid-key error", invalid)
		}
	}
}
