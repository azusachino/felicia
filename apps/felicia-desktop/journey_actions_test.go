package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

func TestSampleJourneyActionsUseIsolatedSourcesAndPersistRealPlans(t *testing.T) {
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
	h, err := NewHandler(HandlerConfig{Repo: repo, MediaRoot: filepath.Join(root, "media"), PublicDir: filepath.Join(root, "site"), Mode: "admin", Sample: true})
	if err != nil {
		t.Fatal(err)
	}
	base := "/api/admin/journeys/0190cbde-f300-7000-8000-000000000002/"
	request := func(method, action, body string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, base+action, strings.NewReader(body)))
		return w
	}
	for _, action := range []string{"visits", "tray"} {
		w := request(http.MethodGet, action, "")
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", action, w.Code, w.Body)
		}
		var items []map[string]any
		if err = json.Unmarshal(w.Body.Bytes(), &items); err != nil || len(items) == 0 {
			t.Fatalf("%s: %s %v", action, w.Body, err)
		}
	}
	journeyID := uuid.MustParse("0190cbde-f300-7000-8000-000000000002")
	seed, err := repo.GetJourney(context.Background(), journeyID)
	if err != nil {
		t.Fatal(err)
	}
	originalTitle := seed.Title
	seed.GPSRoute = nil
	if err = repo.UpsertJourney(context.Background(), seed); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"sync-route", "intake/plan"} {
		w := request(http.MethodPost, action, "")
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", action, w.Code, w.Body)
		}
	}
	updated, err := repo.GetJourney(context.Background(), journeyID)
	if err != nil || len(updated.GPSRoute) == 0 || updated.Title != originalTitle {
		t.Fatal("sync did not restore the real stored route while preserving authored title", err)
	}
	first := request(http.MethodGet, "stop-candidates", "")
	var stops []map[string]any
	if err = json.Unmarshal(first.Body.Bytes(), &stops); err != nil || len(stops) == 0 {
		t.Fatalf("no persisted stops: %d %s %v", first.Code, first.Body, err)
	}
	ids := map[string]bool{}
	for _, stop := range stops {
		ids[stop["id"].(string)] = true
	}
	if w := request(http.MethodPost, "intake/plan", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	second := request(http.MethodGet, "stop-candidates", "")
	var again []map[string]any
	_ = json.Unmarshal(second.Body.Bytes(), &again)
	if len(again) != len(stops) {
		t.Fatalf("duplicate stops: %d vs %d", len(again), len(stops))
	}
	for _, stop := range again {
		if !ids[stop["id"].(string)] {
			t.Fatal("unstable stop ID")
		}
	}
	w := request(http.MethodPost, "snap", `{"point":[135.7672,34.9859]}`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"Point"`) {
		t.Fatalf("snap: %d %s", w.Code, w.Body)
	}
	if w = request(http.MethodPost, "snap", `{"point":[200,100]}`); w.Code != 400 {
		t.Fatal("invalid point accepted")
	}
	if w = request(http.MethodGet, "sync-route", ""); w.Code != 405 {
		t.Fatal("sync allowed GET")
	}
	before, err := repo.ListMementosByJourney(context.Background(), uuid.MustParse("0190cbde-f300-7000-8000-000000000002"))
	if err != nil || len(before) != 3 {
		t.Fatal("source previews/plans created public mementos", err, len(before))
	}
	h.cfg.Sample = false
	if w = request(http.MethodGet, "visits", ""); w.Code != 501 {
		t.Fatal("real workspace silently used sample sources", w.Code)
	}
}
