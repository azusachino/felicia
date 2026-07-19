package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/server/api"
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
