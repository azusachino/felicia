package publication

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeArtifacts(t *testing.T, root string, jsonFiles []string, mediaFiles []string) *FileArtifactWriter {
	t.Helper()
	writer := &FileArtifactWriter{Root: root}
	for _, name := range jsonFiles {
		if err := writer.WriteJSON(name, map[string]string{"file": name}); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	for _, name := range mediaFiles {
		if err := writer.WriteMedia(name, strings.NewReader("media-bytes")); err != nil {
			t.Fatalf("write media %s: %v", name, err)
		}
	}
	return writer
}

func mustNotExist(t *testing.T, root, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); !os.IsNotExist(err) {
		t.Errorf("%s should have been removed (stat err = %v)", name, err)
	}
}

func mustExist(t *testing.T, root, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(name))); err != nil {
		t.Errorf("%s should exist: %v", name, err)
	}
}

func TestFinalizeRemovesStaleArtifactsAndPrunesDirs(t *testing.T) {
	root := t.TempDir()
	// A co-located SPA build must never be touched by cleanup.
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("<html>"), 0o644); err != nil {
		t.Fatalf("seed spa file: %v", err)
	}

	first := writeArtifacts(t, root,
		[]string{"api/v1/journeys.json", "api/v1/journeys/a.json", "api/v1/journeys/a/mementos.json"},
		[]string{"workflow/live.jpg"},
	)
	if removed, err := first.Finalize(); err != nil || len(removed) != 0 {
		t.Fatalf("first finalize = (%v, %v), want no removals", removed, err)
	}

	// The journey is unpublished: the second compile emits only the index.
	second := writeArtifacts(t, root, []string{"api/v1/journeys.json"}, nil)
	removed, err := second.Finalize()
	if err != nil {
		t.Fatalf("second finalize: %v", err)
	}
	if len(removed) != 3 {
		t.Fatalf("removed = %v, want the journey detail, mementos, and media", removed)
	}
	mustNotExist(t, root, "api/v1/journeys/a.json")
	mustNotExist(t, root, "api/v1/journeys/a/mementos.json")
	mustNotExist(t, root, "workflow/live.jpg")
	mustNotExist(t, root, "workflow")
	mustNotExist(t, root, "api/v1/journeys/a")
	mustExist(t, root, "index.html")
	mustExist(t, root, "api/v1/journeys.json")
	mustExist(t, root, ManifestPath)
}

func TestFinalizeWithoutPreviousManifestRemovesNothing(t *testing.T) {
	root := t.TempDir()
	stray := filepath.Join(root, "api", "v1", "stray.json")
	if err := os.MkdirAll(filepath.Dir(stray), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(stray, []byte("{}"), 0o644); err != nil {
		t.Fatalf("seed stray: %v", err)
	}

	writer := writeArtifacts(t, root, []string{"api/v1/journeys.json"}, nil)
	removed, err := writer.Finalize()
	if err != nil || len(removed) != 0 {
		t.Fatalf("finalize = (%v, %v), want no removals without a manifest", removed, err)
	}
	mustExist(t, root, "api/v1/stray.json")
}

func TestFinalizeIgnoresMalformedManifest(t *testing.T) {
	root := t.TempDir()
	manifest := filepath.Join(root, filepath.FromSlash(ManifestPath))
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(manifest, []byte("not json"), 0o644); err != nil {
		t.Fatalf("seed manifest: %v", err)
	}

	writer := writeArtifacts(t, root, []string{"api/v1/journeys.json"}, nil)
	removed, err := writer.Finalize()
	if err != nil || len(removed) != 0 {
		t.Fatalf("finalize = (%v, %v), want cleanup skipped on malformed manifest", removed, err)
	}
}

func TestSafeJoinRejectsUnsafePaths(t *testing.T) {
	for _, unsafe := range []string{"", "/abs/path", "../escape", "a/../../b", `a\b`, "."} {
		if _, err := SafeJoin("/root", unsafe); err == nil {
			t.Errorf("SafeJoin(%q) should fail", unsafe)
		}
	}
	if joined, err := SafeJoin("/root", "api/v1/journeys.json"); err != nil || !strings.HasSuffix(joined, filepath.FromSlash("api/v1/journeys.json")) {
		t.Errorf("SafeJoin valid path = (%q, %v)", joined, err)
	}
}
