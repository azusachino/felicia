package api

import (
	"bytes"
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
// pending-build when what a build would now publish for it differs from what
// the artifact already holds. Comparison is by projected content, not by ID
// membership: the artifact's mementos.json carries the whole public projection
// — title, place, essay, price, kind_data and photos — so projecting current
// state through the same NewStaticMemento the compiler uses and comparing the
// result answers "would a build change anything" rather than only "did
// something appear or disappear". Editing an already-published essay is a
// change to the site, and membership cannot see it.
//
// This stays stateless and self-correcting: the artifact remains the record of
// what was built, so there is no dirty flag to drift.
//
// "No artifact" is reported as its own state rather than as nothing pending.
// Those are different situations — one site is current, the other has never
// been built — and collapsing them told the author their site was up to date
// when it did not exist.
//
// What this cannot know is what a remote host actually serves. A local output
// directory is evidence of a local build, so the states below describe the
// build, and deployment acknowledgement is deliberately absent rather than
// implied.

// artifactReady reports whether a compiled artifact exists to compare against.
func (s *Server) artifactReady() bool {
	return fileExists(filepath.Join(s.SiteOutDir(), filepath.FromSlash(publication.ManifestPath)))
}

// artifactMementos returns what the compiled artifact already publishes for a
// journey, keyed by memento ID, as canonical JSON per entry. A missing file
// yields an empty map — that journey has nothing built.
//
// Re-marshalling each entry rather than keeping the raw bytes normalises key
// order and whitespace, so the comparison answers "is the content different"
// and not "did the encoder space it differently".
func (s *Server) artifactMementos(journeyID uuid.UUID) (map[string][]byte, error) {
	path := filepath.Join(s.SiteOutDir(), "api", "v1", "journeys", journeyID.String(), "mementos.json")
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	var mementos []json.RawMessage
	if err := json.Unmarshal(raw, &mementos); err != nil {
		return nil, err
	}
	built := make(map[string][]byte, len(mementos))
	for _, entry := range mementos {
		var identified struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(entry, &identified); err != nil {
			return nil, err
		}
		canonical, err := canonicalJSON(entry)
		if err != nil {
			return nil, err
		}
		built[identified.ID] = canonical
	}
	return built, nil
}

// canonicalJSON re-encodes a value so two semantically equal documents compare
// equal as bytes. encoding/json sorts map keys, which is what makes this work.
func canonicalJSON(value any) ([]byte, error) {
	var intermediate any
	switch typed := value.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(typed, &intermediate); err != nil {
			return nil, err
		}
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(encoded, &intermediate); err != nil {
			return nil, err
		}
	}
	return json.Marshal(intermediate)
}

// Build states reported to the author. They are deliberately about the local
// build, never about a remote deployment, which this server has no evidence of.
const (
	buildStateNeverBuilt = "never_built"
	buildStateChanged    = "changed"
	buildStateBuilt      = "built"
)

// journeyPendingBuild computes, for one journey, which mementos a build would
// change (for highlighting), the total pending count, and the build state.
//
// The count includes artifact entries with no live row — an unpublished or
// deleted memento still present in the artifact is pending removal, and has no
// live memento to highlight.
func (s *Server) journeyPendingBuild(ctx context.Context, journeyID uuid.UUID) (pendingIDs []string, count int, state string, err error) {
	pendingIDs = []string{}
	if !s.artifactReady() {
		// Nothing has been built, so everything currently publishable is
		// pending. Reporting zero here is what told an author with no artifact
		// at all that their site was up to date.
		mementos, listErr := s.repo.ListMementosByJourney(ctx, journeyID)
		if listErr != nil {
			return nil, 0, "", listErr
		}
		for _, m := range mementos {
			if m.State == domain.MementoPublished {
				pendingIDs = append(pendingIDs, m.ID.String())
			}
		}
		return pendingIDs, len(pendingIDs), buildStateNeverBuilt, nil
	}
	built, err := s.artifactMementos(journeyID)
	if err != nil {
		return nil, 0, "", err
	}
	mementos, err := s.repo.ListMementosByJourney(ctx, journeyID)
	if err != nil {
		return nil, 0, "", err
	}
	liveIDs := make(map[string]bool, len(mementos))
	for _, memento := range mementos {
		id := memento.ID.String()
		liveIDs[id] = true
		existing, inArtifact := built[id]
		if memento.State != domain.MementoPublished {
			// Only pending if the artifact still carries it, i.e. a build would
			// withdraw it.
			if inArtifact {
				pendingIDs = append(pendingIDs, id)
			}
			continue
		}
		if !inArtifact {
			pendingIDs = append(pendingIDs, id)
			continue
		}
		projected, projectErr := s.projectedMemento(ctx, memento)
		if projectErr != nil {
			return nil, 0, "", projectErr
		}
		if !bytes.Equal(projected, existing) {
			pendingIDs = append(pendingIDs, id)
		}
	}
	count = len(pendingIDs)
	for id := range built {
		if !liveIDs[id] {
			count++
		}
	}
	if count == 0 {
		return pendingIDs, 0, buildStateBuilt, nil
	}
	return pendingIDs, count, buildStateChanged, nil
}

// projectedMemento renders one memento exactly as a build would publish it,
// through the same projection the compiler uses, so a difference here is a
// difference the reader would see.
func (s *Server) projectedMemento(ctx context.Context, memento *domain.Memento) ([]byte, error) {
	photos, err := s.repo.ListPhotosByMemento(ctx, memento.ID)
	if err != nil {
		return nil, err
	}
	projection := make([]publication.StaticPhoto, 0, len(photos))
	for _, photo := range photos {
		projection = append(projection, publication.NewStaticPhoto(photo))
	}
	return canonicalJSON(publication.NewStaticMemento(memento, projection))
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
	pendingIDs, count, state, err := s.journeyPendingBuild(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"pending_memento_ids": pendingIDs,
		"pending_count":       count,
		"build_state":         state,
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
		_, count, _, err := s.journeyPendingBuild(r.Context(), j.ID)
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
