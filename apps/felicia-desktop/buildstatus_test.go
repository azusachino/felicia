package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDesktopBuildStatusRoutes(t *testing.T) {
	h, _, _ := setupTestHandler(t, "admin", "")
	created := httptest.NewRecorder()
	h.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		`{"slug":"status-trip","title":"Status Trip","place":"Kyoto","date_start":"2026-05-01","date_end":"2026-05-03"}`)))
	if created.Code != http.StatusOK {
		t.Fatalf("create journey: %d %s", created.Code, created.Body)
	}
	var journey struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &journey); err != nil {
		t.Fatal(err)
	}
	url := "/api/admin/journeys/" + journey.ID + "/build-status"
	check := func(state string, count int) {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, url, nil))
		var result struct {
			State string `json:"build_state"`
			Count int    `json:"pending_count"`
		}
		if w.Code != http.StatusOK {
			t.Fatalf("build status: %d %s", w.Code, w.Body)
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if result.State != state || result.Count != count {
			t.Fatalf("status=%+v, want state %q count %d", result, state, count)
		}
	}
	check("never_built", 0)
	root := h.cfg.PublicDir
	artifact := filepath.Join(root, "api", "v1", "journeys", journey.ID)
	if err := os.MkdirAll(artifact, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "v1", "manifest.json"), []byte(`{"files":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	check("built", 0)
	// A removed row still in public output is pending withdrawal, not zero.
	if err := os.WriteFile(filepath.Join(artifact, "mementos.json"), []byte(`[{"id":"`+uuid.NewString()+`"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	check("changed", 1)
	all := httptest.NewRecorder()
	h.ServeHTTP(all, httptest.NewRequest(http.MethodGet, "/api/admin/build-status", nil))
	var aggregate struct {
		Counts map[string]int `json:"pending_by_journey"`
	}
	if err := json.Unmarshal(all.Body.Bytes(), &aggregate); err != nil {
		t.Fatal(err)
	}
	if all.Code != http.StatusOK || aggregate.Counts[journey.ID] != 1 {
		t.Fatalf("aggregate: %d %s", all.Code, all.Body)
	}
}
