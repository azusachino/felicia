package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func TestSampleWorkspaceIsFreshAndNeverTouchesAuthorHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	private := filepath.Join(home, ".felicia")
	if err := os.Mkdir(private, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(private, "felicia.sqlite")
	if err := os.WriteFile(sentinel, []byte("private sentinel"), 0o600); err != nil {
		t.Fatal(err)
	}
	var firstTitle string
	for range 2 {
		root, err := createSampleWorkspace()
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = os.RemoveAll(root) }()
		if filepath.Dir(root) == home || root == private {
			t.Fatal("sample reused author path")
		}
		repo, err := sqlite.Open(filepath.Join(root, "felicia.sqlite"))
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		journeys, err := repo.ListJourneys(ctx)
		if err != nil || len(journeys) != 1 {
			t.Fatalf("journeys=%v err=%v", journeys, err)
		}
		if firstTitle != "" && journeys[0].Title != firstTitle {
			t.Fatal("sample changed across openings")
		}
		firstTitle = journeys[0].Title
		mementos, err := repo.ListMementosByJourney(ctx, journeys[0].ID)
		if err != nil || len(mementos) != 3 {
			t.Fatalf("mementos=%v err=%v", mementos, err)
		}
		if _, err = os.Stat(filepath.Join(root, "site", filepath.FromSlash(publication.ManifestPath))); err != nil {
			t.Fatal(err)
		}
		h, err := NewHandler(HandlerConfig{Repo: repo, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), Mode: "admin", Sample: true})
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range mementos {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/mementos/"+item.ID.String()+"/photos", nil))
			if w.Code != http.StatusOK {
				t.Fatalf("photos status=%d body=%s", w.Code, w.Body)
			}
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/desktop/workspace", nil))
		var workspace map[string]bool
		if err = json.Unmarshal(w.Body.Bytes(), &workspace); err != nil || !workspace["sample"] {
			t.Fatalf("workspace=%s err=%v", w.Body, err)
		}
		for _, path := range []string{"/api/desktop/pick-folder", "/api/desktop/pick-file", "/api/admin/local-journeys/scan"} {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, path, nil))
			if w.Code != http.StatusForbidden {
				t.Fatalf("sample allowed %s: %d", path, w.Code)
			}
		}
		_ = repo.Close()
	}
	unchanged, err := os.ReadFile(sentinel)
	if err != nil || string(unchanged) != "private sentinel" {
		t.Fatalf("author data changed: %q, %v", unchanged, err)
	}
}
