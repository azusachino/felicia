package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/local"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
)

type localJourneyRequest struct {
	Workspace string    `json:"workspace"`
	JourneyID uuid.UUID `json:"journey_id"`
	JournalID uuid.UUID `json:"journal_id"`
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

func (s *Server) handleScanLocalJourney(w http.ResponseWriter, r *http.Request) {
	var request localJourneyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	result, err := s.scanLocalJourney(r, request)
	if err != nil {
		respondError(w, localJourneyErrorStatus(err), err.Error())
		return
	}
	respondJSON(w, http.StatusOK, result)
}

func (s *Server) handleImportLocalJourney(w http.ResponseWriter, r *http.Request) {
	var request localJourneyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	if request.JourneyID == uuid.Nil || request.Slug == "" || request.Title == "" {
		respondError(w, http.StatusBadRequest, "journey_id, slug, and title are required")
		return
	}
	result, err := s.scanLocalJourney(r, request)
	if err != nil {
		respondError(w, localJourneyErrorStatus(err), err.Error())
		return
	}
	journalID := request.JournalID
	if journalID == uuid.Nil {
		journal, getErr := s.repo.GetSoleJournal(r.Context())
		if getErr != nil {
			if !errors.Is(getErr, domain.ErrNotFound) {
				respondError(w, http.StatusInternalServerError, getErr.Error())
				return
			}
			journalID = uuid.Must(uuid.NewV7())
			if err := s.repo.CreateJournal(r.Context(), &domain.Journal{ID: journalID, CreatedAt: time.Now().UTC()}); err != nil {
				respondError(w, http.StatusInternalServerError, err.Error())
				return
			}
		} else {
			journalID = journal.ID
		}
	}
	start, end := result.Plan.DateStart, result.Plan.DateEnd
	if start.IsZero() || end.IsZero() {
		respondError(w, http.StatusUnprocessableEntity, "workspace has no timestamped route or photo data")
		return
	}
	route := make(orb.MultiLineString, 0, len(result.Plan.Routes))
	for _, source := range result.Plan.Routes {
		if len(source.Line) >= 2 {
			route = append(route, source.Line)
		}
	}
	sourceRef := "route.gpx"
	journey := &domain.Journey{ID: request.JourneyID, JournalID: journalID, Slug: request.Slug, SourceRef: &sourceRef, Title: request.Title, Place: request.Place, DateStart: start, DateEnd: end, GPSRoute: route}
	if err := s.journeyWriter.Save(r.Context(), journey); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.intakeService.Apply(r.Context(), result.Plan); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]any{"journey": journey, "plan": result.Plan, "status": "imported"})
}

func (s *Server) scanLocalJourney(r *http.Request, request localJourneyRequest) (*localJourneyPlan, error) {
	workspace, err := s.localWorkspace(request.Workspace)
	if err != nil {
		return nil, err
	}
	if request.JourneyID == uuid.Nil {
		request.JourneyID = uuid.Must(uuid.NewV7())
	}
	routePath := filepath.Join(workspace, "route.gpx")
	photosPath := filepath.Join(workspace, "photos")
	if _, err := os.Stat(routePath); err != nil {
		return nil, fmt.Errorf("route.gpx is required")
	}
	if info, err := os.Stat(photosPath); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("photos directory is required")
	}
	planService := intake.NewService(nil, nil)
	fingerprint, err := fileFingerprint(routePath)
	if err != nil {
		return nil, err
	}
	plan, err := planService.Plan(r.Context(), intake.PlanRequest{JourneyID: request.JourneyID, SourceFingerprint: fingerprint, Sources: intake.SourceSet{Routes: local.NewGPXSource(routePath), Media: local.NewPhotoSourceWithSidecar(photosPath, filepath.Join(workspace, "photos.jsonl"))}})
	if err != nil {
		return nil, err
	}
	return &localJourneyPlan{JourneyID: request.JourneyID, Workspace: workspace, Plan: plan, Files: map[string]string{"route": routePath, "photos": photosPath}}, nil
}

func (s *Server) localWorkspace(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("workspace is required")
	}
	root := s.siteBrowseRoot
	if root == "" {
		if home, err := os.UserHomeDir(); err == nil {
			root = home
		}
	}
	absRoot, err := resolvePath(root)
	if err != nil {
		return "", fmt.Errorf("browse root is unavailable")
	}
	absWorkspace, err := resolvePath(value)
	if err != nil || !withinRoot(absRoot, absWorkspace) {
		return "", fmt.Errorf("workspace is outside the configured browse root")
	}
	return absWorkspace, nil
}

func fileFingerprint(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", filename, err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func localJourneyErrorStatus(err error) int {
	message := err.Error()
	if len(message) >= len("workspace is outside") && (message == "workspace is required" || message == "route.gpx is required" || message == "photos directory is required" || message == "workspace is outside the configured browse root") {
		return http.StatusBadRequest
	}
	return http.StatusUnprocessableEntity
}
