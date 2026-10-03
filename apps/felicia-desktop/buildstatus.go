package main

import (
	"net/http"

	"github.com/google/uuid"

	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func (h *DesktopHandler) handleBuildStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	h.mu.RLock()
	root := h.publicDir
	h.mu.RUnlock()
	if idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid journey UUID")
			return
		}
		ids, count, state, err := publication.JourneyBuildStatus(r.Context(), h.cfg.Repo, root, id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(w, http.StatusOK, map[string]any{"pending_memento_ids": ids, "pending_count": count, "build_state": state})
		return
	}
	journeys, err := h.cfg.Repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	counts := map[string]int{}
	for _, j := range journeys {
		_, count, _, err := publication.JourneyBuildStatus(r.Context(), h.cfg.Repo, root, j.ID)
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
