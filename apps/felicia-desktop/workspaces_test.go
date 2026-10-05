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

func TestWorkspaceRouterReturnsToOriginalAndResetsSample(t *testing.T) {
	root, err := createSampleWorkspace()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(root) }()
	repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()
	journeys, err := repo.ListJourneys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	original := journeys[0]
	original.Title = "Original synthetic workspace"
	if err = repo.UpsertJourney(ctx, original); err != nil {
		t.Fatal(err)
	}
	h, err := NewHandler(HandlerConfig{Repo: repo, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), Mode: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	router := NewWorkspaceRouter(h, nil)
	defer router.Close()
	for range 2 {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/desktop/sample", nil))
		if w.Code != http.StatusOK {
			t.Fatalf("open: %d %s", w.Code, w.Body)
		}
		if router.current == h || !router.current.cfg.Sample || !router.current.cfg.ReturnAvailable {
			t.Fatal("did not switch isolated handler")
		}
		sampleRoot := router.sampleRoot
		if sampleRoot == root {
			t.Fatal("reused original workspace")
		}
		sampleJourneys, err := router.current.cfg.Repo.ListJourneys(ctx)
		if err != nil || sampleJourneys[0].Title == original.Title || sampleJourneys[0].Title == "Changed sample" {
			t.Fatalf("sample not fresh: %v %v", sampleJourneys, err)
		}
		sampleJourneys[0].Title = "Changed sample"
		if err = router.current.cfg.Repo.UpsertJourney(ctx, sampleJourneys[0]); err != nil {
			t.Fatal(err)
		}
		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/desktop/sample/close", nil))
		if w.Code != http.StatusOK || router.current != h {
			t.Fatalf("return: %d %s", w.Code, w.Body)
		}
		if _, err = os.Stat(sampleRoot); !os.IsNotExist(err) {
			t.Fatalf("sample resources retained: %v", err)
		}
		persisted, err := repo.GetJourney(ctx, original.ID)
		if err != nil || persisted.Title != original.Title {
			t.Fatalf("original changed: %v %v", persisted, err)
		}
	}
}
