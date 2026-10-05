package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

func TestStandaloneSampleCloseReturnsToEmptyIsolatedHandler(t *testing.T) {
	root := t.TempDir()
	repo, err := sqlite.Open(filepath.Join(root, "empty.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	h, err := NewHandler(HandlerConfig{Repo: repo, Mode: "admin", Isolated: true, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), OnPickFolder: func(string) (string, error) { t.Fatal("native picker invoked"); return "", nil }})
	if err != nil {
		t.Fatal(err)
	}
	router := NewWorkspaceRouter(h, nil)
	defer router.Close()
	for range 2 {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/desktop/sample", nil))
		if w.Code != http.StatusOK || !router.current.cfg.Sample || router.current.cfg.ReturnAvailable {
			t.Fatalf("sample open: status=%d cfg=%+v", w.Code, router.current.cfg)
		}
		sampleRoot := router.sampleRoot
		journeys, err := router.current.cfg.Repo.ListJourneys(context.Background())
		if err != nil || len(journeys) != 1 {
			t.Fatalf("seed: %v %v", journeys, err)
		}
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/desktop/sample/close", nil))
		if w.Code != http.StatusOK || router.current != h || !router.current.cfg.Isolated || router.current.cfg.Sample {
			t.Fatalf("close: %d %s", w.Code, w.Body)
		}
		journeys, err = repo.ListJourneys(context.Background())
		if err != nil || len(journeys) != 0 {
			t.Fatalf("baseline not empty: %v %v", journeys, err)
		}
		if _, err := os.Stat(sampleRoot); !os.IsNotExist(err) {
			t.Fatalf("sample root retained: %v", err)
		}
		for _, path := range []string{"/api/desktop/pick-folder", "/api/desktop/pick-file", "/api/admin/local-journeys/scan", "/api/admin/local-journeys/import", "/api/admin/site/output-dir", "/api/admin/site/directories"} {
			w = httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
			if w.Code != http.StatusForbidden {
				t.Fatalf("isolated path %s: %d %s", path, w.Code, w.Body)
			}
		}
	}
}
