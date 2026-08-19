package api

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

// Pending-build tracking (docs/contracts/memento-lifecycle.md §6). A memento is
// pending-build when its current published visibility differs from the deployed
// artifact: the DB's set of published memento IDs for a journey is compared
// against the IDs present in that journey's compiled mementos.json. This is
// stateless and self-correcting — the artifact is the source of truth for what
// is deployed, so there is no dirty flag to drift. When no artifact exists yet
// nothing is pending; the first build establishes the baseline.

// artifactReady reports whether a compiled artifact exists to compare against.
func (s *Server) artifactReady() bool {
	return fileExists(filepath.Join(s.SiteOutDir(), filepath.FromSlash(publication.ManifestPath)))
}

// artifactMementoIDs returns the memento IDs present in a journey's compiled
// mementos.json (the deployed/published set). A missing file yields an empty
// set — the journey has no deployed mementos.
func (s *Server) artifactMementoIDs(journeyID uuid.UUID) (map[string]bool, error) {
	path := filepath.Join(s.SiteOutDir(), "api", "v1", "journeys", journeyID.String(), "mementos.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	var mementos []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw, &mementos); err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(mementos))
	for _, m := range mementos {
		ids[m.ID] = true
	}
	return ids, nil
}

// journeyPendingBuild computes, for one journey, the set of existing mementos
// whose published visibility differs from the artifact (for highlighting) and
// the total pending count (which also includes artifact entries with no live
// row — e.g. an unpublished-then-deleted memento still listed in the artifact).
func (s *Server) journeyPendingBuild(ctx context.Context, journeyID uuid.UUID) (pendingIDs []string, count int, err error) {
	if !s.artifactReady() {
		return nil, 0, nil
	}
	artifactIDs, err := s.artifactMementoIDs(journeyID)
	if err != nil {
		return nil, 0, err
	}
	mementos, err := s.repo.ListMementosByJourney(ctx, journeyID)
	if err != nil {
		return nil, 0, err
	}
	liveIDs := make(map[string]bool, len(mementos))
	pendingIDs = []string{}
	for _, m := range mementos {
		id := m.ID.String()
		liveIDs[id] = true
		publishedNow := m.State == domain.MementoPublished
		inArtifact := artifactIDs[id]
		if publishedNow != inArtifact {
			pendingIDs = append(pendingIDs, id)
		}
	}
	count = len(pendingIDs)
	// Artifact entries with no live row are stale deployments pending removal.
	for id := range artifactIDs {
		if !liveIDs[id] {
			count++
		}
	}
	return pendingIDs, count, nil
}

// handleJourneyBuildStatus returns the pending-build state for one journey:
// the IDs of live mementos to highlight, and the total pending count for the
// journey-detail Build action's (N) badge.
func (s *Server) handleJourneyBuildStatus(w http.ResponseWriter, r *http.Request) {
	journeyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	pendingIDs, count, err := s.journeyPendingBuild(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"pending_memento_ids": pendingIDs,
		"pending_count":       count,
	})
}

// handleBuildStatus returns a per-journey pending count across all journeys,
// for the journeys-list Build action's (N) badge and row highlighting.
func (s *Server) handleBuildStatus(w http.ResponseWriter, r *http.Request) {
	journeys, err := s.repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	counts := map[string]int{}
	for _, j := range journeys {
		_, count, err := s.journeyPendingBuild(r.Context(), j.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if count > 0 {
			counts[j.ID.String()] = count
		}
	}
	respondJSON(w, http.StatusOK, map[string]any{"pending_by_journey": counts})
}
