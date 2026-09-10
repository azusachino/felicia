package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

func TestJourneyBuildStatus(t *testing.T) {
	outDir := t.TempDir()
	journeyID := uuid.New()

	inArtifact := uuid.New()
	unpublishedSince := uuid.New() // in artifact but now authored → pending removal
	newlyPublished := uuid.New()   // published but absent from the artifact → pending add

	unchanged := &domain.Memento{ID: inArtifact, JourneyID: journeyID, Kind: "transit", Seq: 1, Title: "As built", State: domain.MementoPublished}
	withdrawn := &domain.Memento{ID: unpublishedSince, JourneyID: journeyID, Kind: "transit", Seq: 2, Title: "Was built", State: domain.MementoAuthored}

	// The artifact holds both, projected exactly as a build would write them.
	writeArtifactMementos(t, outDir, journeyID, unchanged, &domain.Memento{ID: unpublishedSince, JourneyID: journeyID, Kind: "transit", Seq: 2, Title: "Was built", State: domain.MementoPublished})
	// Manifest marks the artifact as ready.
	mustWrite(t, filepath.Join(outDir, "api", "v1", "manifest.json"), `{"files":[]}`)

	repo := newMockRepository()
	repo.journeys[journeyID] = &domain.Journey{ID: journeyID}
	repo.mementos[inArtifact] = unchanged
	repo.mementos[unpublishedSince] = withdrawn
	repo.mementos[newlyPublished] = &domain.Memento{ID: newlyPublished, JourneyID: journeyID, Kind: "transit", Seq: 3, Title: "New", State: domain.MementoPublished}

	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteOutDir: outDir}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/journeys/"+journeyID.String()+"/build-status", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d (%s)", w.Code, w.Body)
	}
	var res struct {
		PendingMementoIDs []string `json:"pending_memento_ids"`
		PendingCount      int      `json:"pending_count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)

	// Highlighted (live pending) rows: unpublishedSince + newlyPublished.
	if len(res.PendingMementoIDs) != 2 {
		t.Fatalf("pending memento ids = %v, want 2", res.PendingMementoIDs)
	}
	got := map[string]bool{res.PendingMementoIDs[0]: true, res.PendingMementoIDs[1]: true}
	if !got[unpublishedSince.String()] || !got[newlyPublished.String()] {
		t.Fatalf("unexpected pending ids: %v", res.PendingMementoIDs)
	}
	if got[inArtifact.String()] {
		t.Fatal("an already-deployed published memento must not be pending")
	}
	if res.PendingCount != 2 {
		t.Fatalf("pending count = %d, want 2", res.PendingCount)
	}
}

func TestJourneyBuildStatusWithNoArtifactReportsNeverBuilt(t *testing.T) {
	journeyID := uuid.New()
	published := uuid.New()
	repo := newMockRepository()
	repo.journeys[journeyID] = &domain.Journey{ID: journeyID}
	repo.mementos[published] = &domain.Memento{ID: published, JourneyID: journeyID, Kind: "transit", Seq: 1, State: domain.MementoPublished}

	// No artifact at all. This previously reported zero pending, which told an
	// author whose site had never been built that it was up to date.
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteOutDir: t.TempDir()}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/journeys/"+journeyID.String()+"/build-status", nil))
	var res struct {
		PendingCount int    `json:"pending_count"`
		BuildState   string `json:"build_state"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.BuildState != "never_built" {
		t.Errorf("build state = %q, want never_built", res.BuildState)
	}
	if res.PendingCount != 1 {
		t.Errorf("pending count = %d, want 1: everything publishable is pending when nothing is built", res.PendingCount)
	}
}

// TestJourneyBuildStatusSeesAnEssayOnlyEdit is the defect this tracking was
// blind to: membership comparison could not tell that a published memento's
// content had changed, so editing an essay left the site reported as current.
func TestJourneyBuildStatusSeesAnEssayOnlyEdit(t *testing.T) {
	outDir := t.TempDir()
	journeyID := uuid.New()
	mementoID := uuid.New()

	asBuilt := &domain.Memento{ID: mementoID, JourneyID: journeyID, Kind: "transit", Seq: 1, Title: "Kyoto", State: domain.MementoPublished}
	writeArtifactMementos(t, outDir, journeyID, asBuilt)
	mustWrite(t, filepath.Join(outDir, "api", "v1", "manifest.json"), `{"files":[]}`)

	edited := *asBuilt
	essay := "Rewritten after the trip."
	edited.Essay = &essay

	repo := newMockRepository()
	repo.journeys[journeyID] = &domain.Journey{ID: journeyID}
	repo.mementos[mementoID] = &edited

	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteOutDir: outDir}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/journeys/"+journeyID.String()+"/build-status", nil))
	var res struct {
		PendingMementoIDs []string `json:"pending_memento_ids"`
		PendingCount      int      `json:"pending_count"`
		BuildState        string   `json:"build_state"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.PendingCount != 1 || len(res.PendingMementoIDs) != 1 || res.PendingMementoIDs[0] != mementoID.String() {
		t.Fatalf("an essay-only edit must mark the memento pending: count=%d ids=%v", res.PendingCount, res.PendingMementoIDs)
	}
	if res.BuildState != "changed" {
		t.Errorf("build state = %q, want changed", res.BuildState)
	}
}

// writeArtifactMementos writes what a real build would have written: the full
// public projection per memento, not just its ID. Build status compares
// projected content now, so a stub entry would read as "changed" and the
// fixture would be testing the comparison against itself.
func writeArtifactMementos(t *testing.T, outDir string, journeyID uuid.UUID, mementos ...*domain.Memento) {
	t.Helper()
	entries := make([]publication.StaticMemento, 0, len(mementos))
	for _, memento := range mementos {
		entries = append(entries, publication.NewStaticMemento(memento, nil))
	}
	raw, _ := json.Marshal(entries)
	path := filepath.Join(outDir, "api", "v1", "journeys", journeyID.String(), "mementos.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
}
