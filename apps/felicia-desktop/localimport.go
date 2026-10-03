package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
)

// localImportRequest mirrors the admin's local-journey scan/import forms. The
// workspace is a folder holding source material: Google Timeline JSON
// (Timeline.json or a Takeout *.json), optionally route.gpx, and optionally a
// photos/ directory. Unlike the web admin, no browse-root restriction applies:
// paths arrive from the app's own native folder picker.
type localImportRequest struct {
	Workspace string    `json:"workspace"`
	JourneyID uuid.UUID `json:"journey_id"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Place     string    `json:"place"`
}

type localJourneyPlan struct {
	JourneyID uuid.UUID         `json:"journey_id"`
	Workspace string            `json:"workspace"`
	Plan      intake.DraftPlan  `json:"plan"`
	Files     map[string]string `json:"files"`
}

func (h *DesktopHandler) handleScanLocalJourney(w http.ResponseWriter, r *http.Request) {
	var request localImportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	result, err := h.scanLocalJourney(r, request)
	if err != nil {
		respondError(w, localJourneyErrorStatus(err), err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (h *DesktopHandler) handleImportLocalJourney(w http.ResponseWriter, r *http.Request) {
	var request localImportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	if request.Slug == "" || request.Title == "" {
		respondError(w, http.StatusBadRequest, "slug and title are required")
		return
	}
	result, err := h.scanLocalJourney(r, request)
	if err != nil {
		respondError(w, localJourneyErrorStatus(err), err.Error())
		return
	}
	journeyID := request.JourneyID
	if journeyID == uuid.Nil {
		journeyID = result.JourneyID
	}

	start, end := result.Plan.DateStart, result.Plan.DateEnd
	if start.IsZero() || end.IsZero() {
		respondError(w, http.StatusUnprocessableEntity, "workspace has no timestamped source data")
		return
	}

	journeyID, err = h.saveImportedJourney(r, journeyID, request, start, end, result.Plan)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := h.intake.Apply(r.Context(), result.Plan); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "id": journeyID})
}

// scanLocalJourney plans from the workspace without writing anything.
func (h *DesktopHandler) scanLocalJourney(r *http.Request, request localImportRequest) (*localJourneyPlan, error) {
	workspace := request.Workspace
	if workspace == "" {
		return nil, errors.New("workspace is required")
	}
	abs, err := filepath.Abs(workspace)
	if err != nil || !dirExists(abs) {
		return nil, errors.New("workspace folder does not exist")
	}
	if request.JourneyID == uuid.Nil {
		request.JourneyID = uuid.Must(uuid.NewV7())
	}

	sources := intake.SourceSet{}
	files := map[string]string{}

	timelinePath := findTimelineJSON(abs)
	gpxPath := filepath.Join(abs, "route.gpx")
	photosDir := filepath.Join(abs, "photos")

	if timelinePath != "" {
		sources.Visits = local.NewTimelineSource(timelinePath)
		files["timeline"] = timelinePath
	}
	if fileExists(gpxPath) {
		sources.Routes = local.NewGPXSource(gpxPath)
		files["route"] = gpxPath
	}
	if dirExists(photosDir) {
		sources.Media = local.NewPhotoSource(photosDir)
		files["photos"] = photosDir
	}
	if timelinePath == "" && !fileExists(gpxPath) {
		return nil, errors.New("workspace needs Timeline.json or route.gpx")
	}

	fingerprint, err := sourceFingerprint(files)
	if err != nil {
		return nil, err
	}
	plan, err := intake.NewService(nil, nil).Plan(r.Context(), intake.PlanRequest{
		JourneyID:         request.JourneyID,
		SourceFingerprint: fingerprint,
		Sources:           sources,
	})
	if err != nil {
		return nil, err
	}
	return &localJourneyPlan{JourneyID: request.JourneyID, Workspace: abs, Plan: plan, Files: files}, nil
}

// saveImportedJourney mirrors the web admin's import write: the authoring
// service owns journey metadata, the ingest patch owns the GPS trace, and the
// plan goes through the intake service afterwards.
func (h *DesktopHandler) saveImportedJourney(r *http.Request, journeyID uuid.UUID, request localImportRequest, start, end time.Time, plan intake.DraftPlan) (uuid.UUID, error) {
	journalID := uuid.Must(uuid.NewV7())
	if sole, err := h.cfg.Repo.GetSoleJournal(r.Context()); err == nil && sole != nil {
		journalID = sole.ID
	} else if err := h.cfg.Repo.CreateJournal(r.Context(), &domain.Journal{ID: journalID, CreatedAt: time.Now().UTC()}); err != nil {
		return uuid.Nil, err
	}

	storedRoute := planRoute(plan)
	var storedMask []string
	switch stored, err := h.cfg.Repo.GetJourney(r.Context(), journeyID); {
	case errors.Is(err, domain.ErrNotFound):
	case err != nil:
		return uuid.Nil, err
	default:
		if len(stored.GPSRoute) > 0 {
			storedRoute = stored.GPSRoute
		}
		storedMask = stored.AuthoredFields
	}

	sourceRef := "local-workspace"
	journey := &domain.Journey{
		ID:             journeyID,
		JournalID:      journalID,
		Slug:           request.Slug,
		SourceRef:      &sourceRef,
		Title:          request.Title,
		Place:          request.Place,
		DateStart:      start,
		DateEnd:        end,
		GPSRoute:       storedRoute,
		AuthoredFields: domain.ClaimJourneyAuthorship(storedMask),
	}
	if err := h.journeyWriter.Save(r.Context(), journey); err != nil {
		return uuid.Nil, err
	}
	if planHasRoute(plan) {
		if err := h.cfg.Repo.ApplyIngestJourneyPatch(r.Context(), &domain.IngestJourneyPatch{
			Journey: &domain.Journey{ID: journeyID, SourceRef: &sourceRef, GPSRoute: planRoute(plan)},
			Fields:  []string{"source_ref", "gps_route"},
		}); err != nil {
			return uuid.Nil, err
		}
	}
	return journeyID, nil
}

func planRoute(plan intake.DraftPlan) orb.MultiLineString {
	var lines orb.MultiLineString
	for _, route := range plan.Routes {
		if len(route.Line) >= 2 {
			lines = append(lines, route.Line)
		}
	}
	return lines
}

func planHasRoute(plan intake.DraftPlan) bool {
	for _, route := range plan.Routes {
		if len(route.Line) >= 2 {
			return true
		}
	}
	return false
}

// findTimelineJSON accepts Timeline.json (iOS/macOS export) or any single
// Takeout month file placed in the workspace root.
func findTimelineJSON(workspace string) string {
	candidate := filepath.Join(workspace, "Timeline.json")
	if fileExists(candidate) {
		return candidate
	}
	entries, err := os.ReadDir(workspace)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		return filepath.Join(workspace, name)
	}
	return ""
}

func sourceFingerprint(files map[string]string) (string, error) {
	for _, key := range []string{"timeline", "route"} {
		if path, ok := files[key]; ok {
			data, err := os.ReadFile(path)
			if err != nil {
				return "", fmt.Errorf("read %s: %w", key, err)
			}
			sum := sha256.Sum256(data)
			return "sha256:" + hex.EncodeToString(sum[:]), nil
		}
	}
	return "local-workspace", nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func localJourneyErrorStatus(err error) int {
	message := err.Error()
	switch message {
	case "workspace is required", "workspace folder does not exist", "workspace needs Timeline.json or route.gpx", "slug and title are required":
		return http.StatusBadRequest
	default:
		return http.StatusUnprocessableEntity
	}
}
