package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"

	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
)

// Match actions before the generic journey route, which accepts only a UUID.
func (h *DesktopHandler) handleJourneyAction(w http.ResponseWriter, r *http.Request, path string) bool {
	if !strings.HasPrefix(path, "/api/admin/journeys/") {
		return false
	}
	idText, action, ok := strings.Cut(strings.TrimPrefix(path, "/api/admin/journeys/"), "/")
	if !ok {
		return false
	}
	method := http.MethodPost
	switch action {
	case "visits", "tray", "stop-candidates":
		method = http.MethodGet
	case "sync-route", "intake/plan", "snap":
	default:
		return false
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return true
	}
	id, err := uuid.Parse(idText)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return true
	}
	journey, err := h.cfg.Repo.GetJourney(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "journey not found")
		return true
	}
	if action == "snap" {
		var body struct {
			Point []float64 `json:"point"`
		}
		if err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&body); err != nil || len(body.Point) != 2 || body.Point[0] < -180 || body.Point[0] > 180 || body.Point[1] < -90 || body.Point[1] > 90 {
			respondError(w, http.StatusBadRequest, "invalid point coordinates")
			return true
		}
		point, err := h.cfg.Repo.SnapToRoute(r.Context(), id, orb.Point{body.Point[0], body.Point[1]})
		switch {
		case err != nil:
			respondError(w, http.StatusInternalServerError, err.Error())
		case point == nil:
			respondError(w, http.StatusUnprocessableEntity, "journey has no route to snap to")
		default:
			respondJSON(w, http.StatusOK, map[string]any{"point": geojson.NewGeometry(*point)})
		}
		return true
	}
	if action == "stop-candidates" {
		candidates, err := h.intake.List(r.Context(), id)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
		} else {
			respondJSON(w, http.StatusOK, candidates)
		}
		return true
	}
	if !h.cfg.Sample {
		respondError(w, http.StatusNotImplemented, "external sources are not configured in this desktop build")
		return true
	}
	source := sampleSources{repo: h.cfg.Repo}
	im := importer.New(source, source, h.cfg.Repo, 0.0001)
	switch action {
	case "sync-route":
		err = im.SyncRoute(r.Context(), id)
		if err != nil {
			break
		}
		route, e := h.cfg.Repo.GetDisplayRoute(r.Context(), id)
		err = e
		if err == nil {
			respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "gps_route": geojson.NewGeometry(route)})
		}
	case "visits":
		visits, e := im.SyncVisits(r.Context(), id)
		err = e
		if err != nil {
			break
		}
		out := []map[string]any{}
		for _, v := range visits {
			out = append(out, map[string]any{"coord": v.Coord, "label": v.Label, "arrive": v.Arrive.Format(time.RFC3339), "depart": v.Depart.Format(time.RFC3339), "confidence": v.Confidence, "source_ref": v.SourceRef})
		}
		respondJSON(w, http.StatusOK, out)
	case "tray":
		photos, e := im.SyncPhotoTray(r.Context(), id)
		err = e
		if err != nil {
			break
		}
		out := []map[string]any{}
		for _, p := range photos {
			item := map[string]any{"id": p.ID, "at": p.At.Format(time.RFC3339), "checksum": p.Checksum, "source_ref": p.SourceRef}
			if p.Coord != nil {
				item["coord"] = *p.Coord
			}
			out = append(out, item)
		}
		respondJSON(w, http.StatusOK, out)
	case "intake/plan":
		plan, e := h.intake.Plan(r.Context(), intake.PlanRequest{JourneyID: id, From: journey.DateStart, To: journey.DateEnd.AddDate(0, 0, 1).Add(-time.Second), SourceFingerprint: "sample:kyoto", Sources: intake.SourceSet{Routes: source, Visits: source, Media: source}})
		err = e
		if err != nil {
			break
		}
		err = h.intake.Apply(r.Context(), plan)
		if err == nil {
			respondJSON(w, http.StatusOK, map[string]any{"journey_id": plan.JourneyID, "stops": plan.Stops, "issues": plan.Issues})
		}
	}
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
	}
	return true
}
