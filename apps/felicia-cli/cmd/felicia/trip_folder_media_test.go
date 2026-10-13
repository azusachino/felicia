package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
)

func TestInstallMediaFilesCompanionFailures(t *testing.T) {
	t.Run("absent companion", func(t *testing.T) {
		photos := t.TempDir()
		if err := os.WriteFile(filepath.Join(photos, "photo.jpg"), fixtureJPEG(t), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := installMediaFiles(context.Background(), photos, filepath.Join(t.TempDir(), "media"), nil)
		if err != nil || len(got) != 1 || got[0].VideoKey != "" {
			t.Fatalf("installMediaFiles() = (%+v, %v), want one photo without video", got, err)
		}
	})

	t.Run("companion read error", func(t *testing.T) {
		photos := t.TempDir()
		if err := os.WriteFile(filepath.Join(photos, "photo.jpg"), fixtureJPEG(t), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(photos, "photo.MOV"), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := installMediaFiles(context.Background(), photos, filepath.Join(t.TempDir(), "media"), nil)
		if err == nil {
			t.Fatal("installMediaFiles succeeded with an unreadable linked companion")
		}
	})

	t.Run("companion write error", func(t *testing.T) {
		photos := t.TempDir()
		photo := fixtureJPEG(t)
		video := []byte("synthetic video bytes")
		if err := os.WriteFile(filepath.Join(photos, "photo.jpg"), photo, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(photos, "photo.MOV"), video, 0o600); err != nil {
			t.Fatal(err)
		}
		mediaRoot := filepath.Join(t.TempDir(), "media")
		// Block creation of the video object's content-addressed parent only.
		videoKeyDigest := importer.MediaDigest(video)
		blockedDir := filepath.Join(mediaRoot, "media", videoKeyDigest)
		if err := os.MkdirAll(filepath.Dir(blockedDir), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(blockedDir, []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		_, err := installMediaFiles(context.Background(), photos, mediaRoot, nil)
		if err == nil {
			t.Fatal("installMediaFiles succeeded despite companion install failure")
		}
	})
}
