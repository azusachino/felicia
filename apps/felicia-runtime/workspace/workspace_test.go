package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_Explicit(t *testing.T) {
	temp := t.TempDir()
	uncleaned := filepath.Join(temp, "foo", "..", "explicit-ws", ".")
	expectedRoot := filepath.Clean(uncleaned)

	// Set env and create a cwd .felicia to verify explicit root takes highest precedence.
	t.Setenv("FELICIA_WORKSPACE", filepath.Join(temp, "env-ws"))
	cwdDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwdDir, ".felicia"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwdDir)

	ws, err := Resolve(uncleaned)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.Root != expectedRoot {
		t.Errorf("expected Root %q, got %q", expectedRoot, ws.Root)
	}
	if expected := filepath.Join(expectedRoot, "felicia.sqlite"); ws.Database != expected {
		t.Errorf("expected Database %q, got %q", expected, ws.Database)
	}
	if expected := filepath.Join(expectedRoot, "media"); ws.MediaRoot != expected {
		t.Errorf("expected MediaRoot %q, got %q", expected, ws.MediaRoot)
	}
	if expected := filepath.Join(expectedRoot, "site"); ws.PublicDir != expected {
		t.Errorf("expected PublicDir %q, got %q", expected, ws.PublicDir)
	}
}

func TestResolve_EnvVar(t *testing.T) {
	temp := t.TempDir()
	uncleaned := filepath.Join(temp, "a", "..", "env-workspace", ".")
	expectedRoot := filepath.Clean(uncleaned)

	t.Setenv("FELICIA_WORKSPACE", uncleaned)

	// Create a cwd .felicia to ensure env var takes precedence over local cwd.
	cwdDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(cwdDir, ".felicia"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwdDir)

	ws, err := Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.Root != expectedRoot {
		t.Errorf("expected Root %q, got %q", expectedRoot, ws.Root)
	}
	if expected := filepath.Join(expectedRoot, "felicia.sqlite"); ws.Database != expected {
		t.Errorf("expected Database %q, got %q", expected, ws.Database)
	}
	if expected := filepath.Join(expectedRoot, "media"); ws.MediaRoot != expected {
		t.Errorf("expected MediaRoot %q, got %q", expected, ws.MediaRoot)
	}
	if expected := filepath.Join(expectedRoot, "site"); ws.PublicDir != expected {
		t.Errorf("expected PublicDir %q, got %q", expected, ws.PublicDir)
	}
}

func TestResolve_LocalCwdFallback(t *testing.T) {
	t.Setenv("FELICIA_WORKSPACE", "")

	cwdDir := t.TempDir()
	localFelicia := filepath.Join(cwdDir, ".felicia")
	if err := os.Mkdir(localFelicia, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwdDir)

	fakeHome := filepath.Join(t.TempDir(), "fake-home")
	t.Setenv("HOME", fakeHome)

	ws, err := Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ws.Root != localFelicia {
		t.Errorf("expected Root %q, got %q", localFelicia, ws.Root)
	}
	if expected := filepath.Join(localFelicia, "felicia.sqlite"); ws.Database != expected {
		t.Errorf("expected Database %q, got %q", expected, ws.Database)
	}
	if expected := filepath.Join(localFelicia, "media"); ws.MediaRoot != expected {
		t.Errorf("expected MediaRoot %q, got %q", expected, ws.MediaRoot)
	}
	if expected := filepath.Join(localFelicia, "site"); ws.PublicDir != expected {
		t.Errorf("expected PublicDir %q, got %q", expected, ws.PublicDir)
	}
}

func TestResolve_LocalCwdFileIgnored(t *testing.T) {
	t.Setenv("FELICIA_WORKSPACE", "")

	cwdDir := t.TempDir()
	// Create .felicia as a regular file, which should NOT match IsDir().
	if err := os.WriteFile(filepath.Join(cwdDir, ".felicia"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwdDir)

	fakeHome := filepath.Join(t.TempDir(), "fake-home")
	t.Setenv("HOME", fakeHome)

	ws, err := Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedRoot := filepath.Join(fakeHome, ".felicia")
	if ws.Root != expectedRoot {
		t.Errorf("expected fallback to home %q, got %q", expectedRoot, ws.Root)
	}
}

func TestResolve_HomeDirFallback(t *testing.T) {
	t.Setenv("FELICIA_WORKSPACE", "")

	cwdDir := t.TempDir()
	t.Chdir(cwdDir)

	fakeHome := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", fakeHome)

	ws, err := Resolve("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedRoot := filepath.Join(fakeHome, ".felicia")
	if ws.Root != expectedRoot {
		t.Errorf("expected Root %q, got %q", expectedRoot, ws.Root)
	}
	if expected := filepath.Join(expectedRoot, "felicia.sqlite"); ws.Database != expected {
		t.Errorf("expected Database %q, got %q", expected, ws.Database)
	}
	if expected := filepath.Join(expectedRoot, "media"); ws.MediaRoot != expected {
		t.Errorf("expected MediaRoot %q, got %q", expected, ws.MediaRoot)
	}
	if expected := filepath.Join(expectedRoot, "site"); ws.PublicDir != expected {
		t.Errorf("expected PublicDir %q, got %q", expected, ws.PublicDir)
	}
}

func TestPathCleaning(t *testing.T) {
	temp := t.TempDir()
	dirty := filepath.Join(temp, "foo", "..", "bar", ".", "baz", "..")
	expected := filepath.Clean(dirty)

	ws := New(dirty)
	if ws.Root != expected {
		t.Errorf("expected cleaned Root %q, got %q", expected, ws.Root)
	}
	if ws.Database != filepath.Join(expected, "felicia.sqlite") {
		t.Errorf("expected cleaned Database path, got %q", ws.Database)
	}
}

func TestEnsureDirs(t *testing.T) {
	temp := t.TempDir()
	root := filepath.Join(temp, "workspace-to-create")
	ws := New(root)

	// Verify dirs do not exist initially.
	if _, err := os.Stat(ws.Root); !os.IsNotExist(err) {
		t.Fatalf("expected Root not to exist initially")
	}

	if err := ws.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	for _, path := range []string{ws.Root, ws.MediaRoot, ws.PublicDir} {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("directory %s was not created: %v", path, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("path %s is not a directory", path)
		}
	}

	// Idempotency: calling EnsureDirs again should succeed.
	if err := ws.EnsureDirs(); err != nil {
		t.Errorf("EnsureDirs second call failed: %v", err)
	}
}
