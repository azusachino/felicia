package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

// Build status belongs to the publication boundary and is shared with desktop.
func (s *Server) handleJourneyBuildStatus(w http.ResponseWriter, r *http.Request) {
	journeyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	pendingIDs, count, state, err := publication.JourneyBuildStatus(r.Context(), s.repo, s.SiteOutDir(), journeyID)
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

func (s *Server) handleBuildStatus(w http.ResponseWriter, r *http.Request) {
	journeys, err := s.repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	counts := map[string]int{}
	for _, j := range journeys {
		_, count, _, err := publication.JourneyBuildStatus(r.Context(), s.repo, s.SiteOutDir(), j.ID)
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
