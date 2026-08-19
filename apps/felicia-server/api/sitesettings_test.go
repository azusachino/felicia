package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

func TestBrowseDirectoriesStaysWithinRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sites", "personal"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteBrowseRoot: root}).Handler()

	// Root listing hides dotfiles and reports no parent.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/browse", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("browse status = %d (%s)", w.Code, w.Body)
	}
	var res struct {
		Parent string `json:"parent"`
		Dirs   []struct {
			Name string `json:"name"`
			Path string `json:"path"`
		} `json:"dirs"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.Parent != "" {
		t.Errorf("root parent = %q, want empty", res.Parent)
	}
	if len(res.Dirs) != 1 || res.Dirs[0].Name != "sites" {
		t.Fatalf("root dirs = %v, want [sites]", res.Dirs)
	}

	// A path outside the root is refused.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/browse?path="+filepath.Dir(root), nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("escape status = %d, want 400", w.Code)
	}
}

// A symlink under the browse root that points outside it must not let the
// picker escape the root: filepath.Abs/Rel are lexical, so the handler has to
// evaluate symlinks before confining the path.
func TestBrowseDirectoriesRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(outside, "secret"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unsupported on this platform: %v", err)
	}
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteBrowseRoot: root}).Handler()

	// Navigating straight into the symlinked path is refused.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/browse?path="+link, nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("symlink escape status = %d, want 400 (%s)", w.Code, w.Body)
	}

	// And the escaping symlink is never offered in the root listing.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/browse", nil))
	var res struct {
		Dirs []struct {
			Name string `json:"name"`
		} `json:"dirs"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)
	for _, d := range res.Dirs {
		if d.Name == "escape" {
			t.Fatalf("symlink escaping the root was listed: %v", res.Dirs)
		}
	}
}

// Repointing output at a non-empty directory that is not a prior felicia site
// output is refused, so a stray path cannot be scattered with compiler files or
// reconciled unexpectedly. An empty dir or one holding a manifest is accepted.
func TestPutSiteRejectsUnrelatedNonEmptyDir(t *testing.T) {
	srv := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{})
	handler := srv.Handler()
	putOutDir := func(dir string) int {
		body, _ := json.Marshal(map[string]string{"out_dir": dir})
		req := httptest.NewRequest(http.MethodPut, "/api/admin/site", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Code
	}

	// Non-empty, no manifest: refused.
	unrelated := t.TempDir()
	if err := os.WriteFile(filepath.Join(unrelated, "important.txt"), []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := putOutDir(unrelated); code != http.StatusBadRequest {
		t.Fatalf("put into unrelated non-empty dir = %d, want 400", code)
	}
	if _, err := os.Stat(filepath.Join(unrelated, "important.txt")); err != nil {
		t.Fatalf("pre-existing file must be left untouched: %v", err)
	}

	// A directory that already holds a felicia artifact manifest is accepted.
	priorArtifact := t.TempDir()
	manifest := filepath.Join(priorArtifact, filepath.FromSlash(publication.ManifestPath))
	if err := os.MkdirAll(filepath.Dir(manifest), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, []byte(`{"files":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := putOutDir(priorArtifact); code != http.StatusOK {
		t.Fatalf("put into prior artifact dir = %d, want 200", code)
	}
}

func TestPutSiteUpdatesOutDirAndPersists(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "felicia.toml")
	newOut := filepath.Join(t.TempDir(), "site-out")
	srv := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteOutDir: ".felicia/site", ConfigPath: configPath})
	handler := srv.Handler()

	body, _ := json.Marshal(map[string]string{"out_dir": newOut})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("put site status = %d (%s)", w.Code, w.Body)
	}

	// The running server reflects the change immediately.
	if srv.SiteOutDir() != newOut {
		t.Fatalf("SiteOutDir() = %q, want %q", srv.SiteOutDir(), newOut)
	}
	// And it was persisted to the TOML config.
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if !bytes.Contains(raw, []byte(newOut)) {
		t.Fatalf("config does not contain the new out_dir: %s", raw)
	}
}
