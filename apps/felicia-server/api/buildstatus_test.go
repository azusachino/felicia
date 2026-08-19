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
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

func TestJourneyBuildStatus(t *testing.T) {
	outDir := t.TempDir()
	journeyID := uuid.New()

	// Deployed artifact: one published memento (id "in-artifact").
	inArtifact := uuid.New()
	unpublishedSince := uuid.New() // in artifact but now authored → pending removal
	writeArtifactMementos(t, outDir, journeyID, inArtifact, unpublishedSince)
	// Manifest marks the artifact as ready.
	mustWrite(t, filepath.Join(outDir, "api", "v1", "manifest.json"), `{"files":[]}`)

	repo := newMockRepository()
	repo.journeys[journeyID] = &domain.Journey{ID: journeyID}
	// DB state: inArtifact still published (not pending); unpublishedSince now
	// authored (pending removal); newlyPublished published but absent from the
	// artifact (pending add).
	newlyPublished := uuid.New()
	repo.mementos[inArtifact] = &domain.Memento{ID: inArtifact, JourneyID: journeyID, State: domain.MementoPublished}
	repo.mementos[unpublishedSince] = &domain.Memento{ID: unpublishedSince, JourneyID: journeyID, State: domain.MementoAuthored}
	repo.mementos[newlyPublished] = &domain.Memento{ID: newlyPublished, JourneyID: journeyID, State: domain.MementoPublished}

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

func TestJourneyBuildStatusNoArtifactIsZero(t *testing.T) {
	journeyID := uuid.New()
	repo := newMockRepository()
	repo.journeys[journeyID] = &domain.Journey{ID: journeyID}
	repo.mementos[uuid.New()] = &domain.Memento{ID: uuid.New(), JourneyID: journeyID, State: domain.MementoPublished}

	// No SiteOutDir artifact → nothing is pending (first build sets the baseline).
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil,
		api.RouteConfig{SiteOutDir: t.TempDir()}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/journeys/"+journeyID.String()+"/build-status", nil))
	var res struct {
		PendingCount int `json:"pending_count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res.PendingCount != 0 {
		t.Fatalf("pending count without artifact = %d, want 0", res.PendingCount)
	}
}

func writeArtifactMementos(t *testing.T, outDir string, journeyID uuid.UUID, ids ...uuid.UUID) {
	t.Helper()
	type entry struct {
		ID string `json:"id"`
	}
	entries := make([]entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, entry{ID: id.String()})
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
