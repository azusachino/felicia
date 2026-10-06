package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// desktopCreateMementoRequest is the body of POST /api/admin/mementos/create
// — the automatic new-memento creation path (docs/contracts/
// automatic-sequence-allocation.md). It carries the caller-chosen identity
// and authored values only: seq and expected_revision have no field in the
// request at all (unknown fields, including nulls, are rejected): positions
// are server-allocated atomically and edit-revision checks belong to the
// explicit upsert path. The authorship mask is server-derived, never
// client-supplied.
// Unknown fields are rejected so a client aiming the old upsert payload at
// this endpoint fails loudly instead of silently dropping fields.
type desktopCreateMementoRequest struct {
	ID         uuid.UUID       `json:"id"`
	JourneyID  uuid.UUID       `json:"journey_id"`
	Kind       string          `json:"kind"`
	Title      string          `json:"title"`
	Place      string          `json:"place"`
	OccurredAt string          `json:"occurred_at"`
	OccurredTZ string          `json:"occurred_tz"`
	KindData   json.RawMessage `json:"kind_data"`
	State      string          `json:"state"`
}

// handleCreateMemento persists a new draft memento with an automatically
// allocated journey-local position. Creation never edits an existing row: a
// same-ID retry with matching values returns the stored row unchanged, and
// any incompatible retry is a 409 conflict.
func (h *DesktopHandler) handleCreateMemento(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var req desktopCreateMementoRequest
	if err := decoder.Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid create request: "+err.Error())
		return
	}
	if req.ID == uuid.Nil {
		respondError(w, http.StatusBadRequest, "id is required for creation")
		return
	}
	if req.JourneyID == uuid.Nil {
		respondError(w, http.StatusBadRequest, "journey_id is required for creation")
		return
	}
	if req.Kind == "" {
		respondError(w, http.StatusBadRequest, "kind is required for creation")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		respondError(w, http.StatusBadRequest, "title is required for creation")
		return
	}
	// Creation is new-draft only: an existing row's lifecycle moves through
	// the edit path, never through creation.
	if req.State == "" {
		req.State = string(domain.MementoDraft)
	}
	if domain.MementoState(req.State) != domain.MementoDraft {
		respondError(w, http.StatusBadRequest, "creation only supports draft state")
		return
	}
	tpl, ok := h.cfg.Registry.Template(req.Kind)
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "kind template not registered",
			"issues": []domain.Issue{{Code: "kind_not_registered"}},
		})
		return
	}
	var occurred time.Time
	if req.OccurredAt != "" {
		var parseErr error
		occurred, parseErr = time.Parse(time.RFC3339, req.OccurredAt)
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "invalid occurred_at timestamp format (RFC3339)")
			return
		}
	}
	if req.OccurredTZ == "" {
		// Host-independent creation default: an omitted authored zone is UTC,
		// never inferred from an existing row.
		req.OccurredTZ = "UTC"
	}
	if issues := domain.ValidateOccurredTimezone(req.OccurredTZ); len(issues) > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"issues": issues,
		})
		return
	}
	if len(req.KindData) == 0 {
		req.KindData = []byte(`{}`)
	}
	var dataMap map[string]any
	if err := json.Unmarshal(req.KindData, &dataMap); err != nil || dataMap == nil {
		// JSON null (or any non-object) is rejected; an omitted body was
		// already normalized to {} above.
		respondError(w, http.StatusBadRequest, "kind_data must be a JSON object")
		return
	}
	if issues := domain.ValidateForState(tpl, dataMap, domain.MementoState(req.State)); len(issues) > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"issues": issues,
		})
		return
	}
	created, err := h.mementoWriter.CreateManualMemento(r.Context(), &domain.Memento{
		ID:         req.ID,
		JourneyID:  req.JourneyID,
		Kind:       req.Kind,
		OccurredAt: occurred,
		OccurredTZ: req.OccurredTZ,
		Title:      req.Title,
		Place:      req.Place,
		KindData:   req.KindData,
		State:      domain.MementoState(req.State),
	})
	if err != nil {
		if errors.Is(err, domain.ErrWriteConflict) {
			respondError(w, http.StatusConflict, "memento already exists with different creation values")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, created)
}
