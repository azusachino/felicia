package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	core "github.com/azusachino/felicia/apps/felicia-core"
	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
	journeyruntime "github.com/azusachino/felicia/apps/felicia-runtime/journey"
	mementoruntime "github.com/azusachino/felicia/apps/felicia-runtime/memento"
)

//go:embed all:assets
var embeddedAssets embed.FS

var accentPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

type HandlerConfig struct {
	Repo          domain.Repository
	Registry      *domain.Registry
	MediaRoot     string
	PublicDir     string
	Mode          string // "admin" or "reader"
	Token         string // optional; tests only. The packaged app runs tokenless on the webview-only scheme.
	PreviewPort   string
	PreviewPortFn func() string
	OnPickFolder  func(title string) (string, error)
	OnPickFile    func(title string) (string, error)
}

type DesktopHandler struct {
	cfg           HandlerConfig
	journeyWriter *journeyruntime.Service
	mementoWriter *mementoruntime.Service
	intake        *intake.Service
	adminFS       fs.FS
	readerFS      fs.FS
	mu            sync.RWMutex
	publicDir     string
}

func NewHandler(cfg HandlerConfig) (*DesktopHandler, error) {
	if cfg.Registry == nil {
		kindsFS, err := fs.Sub(core.KindsFS, "kinds")
		if err != nil {
			return nil, fmt.Errorf("load kinds fs: %w", err)
		}
		reg, err := domain.LoadRegistry(kindsFS)
		if err != nil {
			return nil, fmt.Errorf("load kinds registry: %w", err)
		}
		cfg.Registry = reg
	}

	adminSub, _ := fs.Sub(embeddedAssets, "assets/admin")
	readerSub, _ := fs.Sub(embeddedAssets, "assets/reader")

	h := &DesktopHandler{
		cfg:           cfg,
		journeyWriter: journeyruntime.New(cfg.Repo),
		mementoWriter: mementoruntime.New(cfg.Repo),
		intake:        intake.NewService(candidateStore(cfg.Repo), cfg.Repo),
		adminFS:       adminSub,
		readerFS:      readerSub,
		publicDir:     cfg.PublicDir,
	}
	return h, nil
}

func candidateStore(repo domain.Repository) ports.StopCandidateStore {
	if store, ok := repo.(ports.StopCandidateStore); ok {
		return store
	}
	return nil
}

func (h *DesktopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cleanPath := path.Clean(r.URL.Path)

	// In reader mode: strictly block admin APIs and serve static compiled JSON
	if h.cfg.Mode == "reader" {
		if strings.HasPrefix(cleanPath, "/api/admin/") {
			http.Error(w, "reader mode does not allow admin operations", http.StatusForbidden)
			return
		}
		if strings.HasPrefix(cleanPath, "/api/") {
			h.serveStaticArtifact(w, r)
			return
		}
	}

	// In admin mode: pre-paint bootstrap script
	if cleanPath == "/boot.js" {
		h.handleBoot(w, r)
		return
	}

	// Desktop native integration endpoints
	if cleanPath == "/api/desktop/pick-folder" {
		if !h.authorizedMutation(w, r) {
			return
		}
		h.handlePick(w, r, true)
		return
	}
	if cleanPath == "/api/desktop/pick-file" {
		if !h.authorizedMutation(w, r) {
			return
		}
		h.handlePick(w, r, false)
		return
	}

	// Admin API routing
	if strings.HasPrefix(cleanPath, "/api/admin/") {
		if !h.authorizedMutation(w, r) {
			return
		}
		if cleanPath == "/api/admin/local-journeys/scan" {
			h.handleScanLocalJourney(w, r)
			return
		}
		if cleanPath == "/api/admin/local-journeys/import" {
			h.handleImportLocalJourney(w, r)
			return
		}
		h.routeAdmin(w, r, cleanPath)
		return
	}

	// Fall through to embedded static UI
	h.serveStaticUI(w, r)
}

func (h *DesktopHandler) authorizedMutation(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		return true
	}
	if h.cfg.Token != "" {
		token := r.Header.Get("X-Desktop-Token")
		if token == "" {
			token = r.Header.Get("X-Spike-Token")
		}
		if token != h.cfg.Token {
			http.Error(w, "forbidden: missing or invalid session token", http.StatusForbidden)
			return false
		}
	}
	return true
}

func (h *DesktopHandler) handleBoot(w http.ResponseWriter, _ *http.Request) {
	settings, err := publication.ResolveSiteSettings(context.Background(), h.cfg.Repo)
	if err != nil {
		settings = domain.DefaultSiteSettings(uuid.Nil)
	}
	boot := map[string]any{
		"lang":    settings.DefaultLanguage,
		"theme":   settings.DefaultTheme,
		"desktop": true,
	}
	b, _ := json.Marshal(boot)
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(append(append([]byte("window.bootPrefs = "), b...), ";\n"...))
}

func (h *DesktopHandler) handlePick(w http.ResponseWriter, r *http.Request, directories bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&body)
	if body.Title == "" {
		body.Title = "Choose a folder"
		if !directories {
			body.Title = "Choose a file"
		}
	}
	pick := h.cfg.OnPickFolder
	if !directories {
		pick = h.cfg.OnPickFile
	}
	if pick == nil {
		respondJSON(w, http.StatusOK, map[string]any{"selected": false, "path": "", "basename": ""})
		return
	}
	chosen, err := pick(body.Title)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"selected": chosen != "",
		"path":     chosen,
		"basename": filepath.Base(chosen),
	})
}

func (h *DesktopHandler) routeAdmin(w http.ResponseWriter, r *http.Request, reqPath string) {
	switch {
	case reqPath == "/api/admin/journeys":
		switch r.Method {
		case http.MethodGet:
			h.handleListJourneys(w, r)
		case http.MethodPost:
			h.handleUpsertJourney(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case strings.HasPrefix(reqPath, "/api/admin/journeys/") && strings.HasSuffix(reqPath, "/mementos"):
		idStr := strings.TrimSuffix(strings.TrimPrefix(reqPath, "/api/admin/journeys/"), "/mementos")
		h.handleListJourneyMementos(w, r, idStr)
	case strings.HasPrefix(reqPath, "/api/admin/journeys/") && strings.HasSuffix(reqPath, "/stop-candidates"):
		respondJSON(w, http.StatusOK, []any{})
	case strings.HasPrefix(reqPath, "/api/admin/journeys/") && strings.HasSuffix(reqPath, "/build-status"):
		idStr := strings.TrimSuffix(strings.TrimPrefix(reqPath, "/api/admin/journeys/"), "/build-status")
		h.handleBuildStatus(w, r, idStr)
	case strings.HasPrefix(reqPath, "/api/admin/journeys/"):
		idStr := strings.TrimPrefix(reqPath, "/api/admin/journeys/")
		switch r.Method {
		case http.MethodGet:
			h.handleGetJourney(w, r, idStr)
		case http.MethodDelete:
			h.handleDeleteJourney(w, r, idStr)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case reqPath == "/api/admin/mementos":
		if r.Method == http.MethodPost {
			h.handleUpsertMemento(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case strings.HasPrefix(reqPath, "/api/admin/mementos/"):
		idStr := strings.TrimPrefix(reqPath, "/api/admin/mementos/")
		switch r.Method {
		case http.MethodGet:
			h.handleGetMemento(w, r, idStr)
		case http.MethodDelete:
			h.handleDeleteMemento(w, r, idStr)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case reqPath == "/api/admin/site-settings":
		switch r.Method {
		case http.MethodGet:
			h.handleGetSiteSettings(w, r)
		case http.MethodPut:
			h.handlePutSiteSettings(w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case reqPath == "/api/admin/site":
		h.handleSiteInfo(w, r)
	case reqPath == "/api/admin/build-status":
		h.handleBuildStatus(w, r, "")
	case reqPath == "/api/admin/compile":
		if r.Method == http.MethodPost {
			h.handleCompile(w, r)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	case reqPath == "/api/admin/templates":
		h.handleTemplates(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (h *DesktopHandler) handleListJourneys(w http.ResponseWriter, r *http.Request) {
	js, err := h.cfg.Repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if js == nil {
		js = make([]*domain.Journey, 0)
	}
	respondJSON(w, http.StatusOK, js)
}

func (h *DesktopHandler) handleGetJourney(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	j, err := h.cfg.Repo.GetJourney(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "journey not found")
		return
	}
	respondJSON(w, http.StatusOK, j)
}

type desktopUpsertJourneyRequest struct {
	ID               uuid.UUID `json:"id"`
	JournalID        uuid.UUID `json:"journal_id"`
	Slug             string    `json:"slug"`
	SourceRef        *string   `json:"source_ref,omitempty"`
	Title            string    `json:"title"`
	Place            string    `json:"place"`
	Country          *string   `json:"country,omitempty"`
	Region           *string   `json:"region,omitempty"`
	DateStart        string    `json:"date_start"`
	DateEnd          string    `json:"date_end"`
	ExpectedRevision *int64    `json:"expected_revision,omitempty"`
}

func (h *DesktopHandler) handleUpsertJourney(w http.ResponseWriter, r *http.Request) {
	var req desktopUpsertJourneyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	start, err := time.Parse("2006-01-02", req.DateStart)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid date_start format (YYYY-MM-DD)")
		return
	}
	end, err := time.Parse("2006-01-02", req.DateEnd)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid date_end format (YYYY-MM-DD)")
		return
	}
	if end.Before(start) {
		respondError(w, http.StatusBadRequest, "End date must not be before start date.")
		return
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.Must(uuid.NewV7())
	}
	existing, lookupErr := h.cfg.Repo.GetJourneyBySlug(r.Context(), req.Slug)
	if lookupErr != nil && !errors.Is(lookupErr, domain.ErrNotFound) && !errors.Is(lookupErr, sql.ErrNoRows) {
		respondError(w, http.StatusInternalServerError, lookupErr.Error())
		return
	}
	if existing != nil && existing.ID != req.ID {
		respondError(w, http.StatusConflict, "A journey with this slug already exists. Choose another slug.")
		return
	}
	if req.JournalID == uuid.Nil {
		sole, getErr := h.cfg.Repo.GetSoleJournal(r.Context())
		if getErr == nil && sole != nil {
			req.JournalID = sole.ID
		} else {
			newJournal := &domain.Journal{ID: uuid.Must(uuid.NewV7()), CreatedAt: time.Now().UTC()}
			if err := h.cfg.Repo.CreateJournal(r.Context(), newJournal); err == nil {
				req.JournalID = newJournal.ID
			}
		}
	}

	var storedRoute orb.MultiLineString
	var storedMask []string
	switch stored, getErr := h.cfg.Repo.GetJourney(r.Context(), req.ID); {
	case errors.Is(getErr, domain.ErrNotFound):
	case getErr != nil:
		respondError(w, http.StatusInternalServerError, getErr.Error())
		return
	default:
		storedRoute, storedMask = stored.GPSRoute, stored.AuthoredFields
	}

	journey := &domain.Journey{
		ID:               req.ID,
		JournalID:        req.JournalID,
		Slug:             req.Slug,
		SourceRef:        req.SourceRef,
		Title:            req.Title,
		Place:            req.Place,
		Country:          req.Country,
		Region:           req.Region,
		DateStart:        start,
		DateEnd:          end,
		GPSRoute:         storedRoute,
		AuthoredFields:   domain.ClaimJourneyAuthorship(storedMask),
		ExpectedRevision: req.ExpectedRevision,
	}
	if err := h.journeyWriter.Save(r.Context(), journey); err != nil {
		if errors.Is(err, domain.ErrWriteConflict) {
			respondError(w, http.StatusConflict, "journey was modified by another writer")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	persisted, err := h.cfg.Repo.GetJourney(r.Context(), journey.ID)
	if err != nil {
		respondJSON(w, http.StatusOK, journey)
		return
	}
	respondJSON(w, http.StatusOK, persisted)
}

func (h *DesktopHandler) handleDeleteJourney(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	if err := h.cfg.Repo.DeleteJourney(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "journey not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *DesktopHandler) handleListJourneyMementos(w http.ResponseWriter, r *http.Request, idStr string) {
	journeyID, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	ms, err := h.cfg.Repo.ListMementosByJourney(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ms == nil {
		ms = make([]*domain.Memento, 0)
	}
	respondJSON(w, http.StatusOK, ms)
}

func (h *DesktopHandler) handleGetMemento(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memento UUID")
		return
	}
	m, err := h.cfg.Repo.GetMemento(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "memento not found")
		return
	}
	respondJSON(w, http.StatusOK, m)
}

type desktopUpsertMementoRequest struct {
	ID               uuid.UUID       `json:"id"`
	JourneyID        uuid.UUID       `json:"journey_id"`
	Kind             string          `json:"kind"`
	Seq              int             `json:"seq"`
	OccurredAt       string          `json:"occurred_at"`
	OccurredTZ       string          `json:"occurred_tz"`
	Title            string          `json:"title"`
	Place            string          `json:"place"`
	Vendor           *string         `json:"vendor,omitempty"`
	Essay            *string         `json:"essay,omitempty"`
	PriceAmount      *int64          `json:"price_amount,omitempty"`
	PriceCurrency    *string         `json:"price_currency,omitempty"`
	KindData         json.RawMessage `json:"kind_data"`
	SourceRef        *string         `json:"source_ref,omitempty"`
	State            string          `json:"state"`
	ExpectedRevision *int64          `json:"expected_revision,omitempty"`
}

func (h *DesktopHandler) handleUpsertMemento(w http.ResponseWriter, r *http.Request) {
	var req desktopUpsertMementoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.Must(uuid.NewV7())
	}
	state := domain.MementoState(req.State)
	if state == "" {
		if existing, err := h.cfg.Repo.GetMemento(r.Context(), req.ID); err == nil {
			state = existing.State
		} else {
			state = domain.MementoDraft
		}
	}
	tpl, ok := h.cfg.Registry.Template(req.Kind)
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "kind template not registered",
			"issues": []domain.Issue{{Code: "kind_not_registered"}},
		})
		return
	}
	var dataMap map[string]any
	if len(req.KindData) > 0 {
		_ = json.Unmarshal(req.KindData, &dataMap)
	}
	if issues := domain.ValidateForState(tpl, dataMap, state); len(issues) > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"issues": issues,
		})
		return
	}
	var occurred time.Time
	if req.OccurredAt != "" {
		var parseErr error
		occurred, parseErr = time.Parse(time.RFC3339, req.OccurredAt)
		if parseErr != nil {
			respondError(w, http.StatusBadRequest, "invalid occurred_at timestamp format")
			return
		}
	}
	memento := &domain.Memento{
		ID:            req.ID,
		JourneyID:     req.JourneyID,
		Kind:          req.Kind,
		Seq:           req.Seq,
		OccurredAt:    occurred,
		OccurredTZ:    req.OccurredTZ,
		Title:         req.Title,
		Place:         req.Place,
		Vendor:        req.Vendor,
		Essay:         req.Essay,
		PriceAmount:   req.PriceAmount,
		PriceCurrency: req.PriceCurrency,
		KindData:      req.KindData,
		SourceRef:     req.SourceRef,
		State:         state,
	}
	fields := []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "title", "place", "kind_data", "source_ref"}
	if err := h.mementoWriter.ApplyManualPatch(r.Context(), &domain.ManualMementoPatch{
		Memento:          memento,
		Fields:           fields,
		State:            state,
		ExpectedRevision: req.ExpectedRevision,
	}); err != nil {
		if errors.Is(err, domain.ErrWriteConflict) {
			respondError(w, http.StatusConflict, "memento revision conflict")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	persisted, err := h.cfg.Repo.GetMemento(r.Context(), memento.ID)
	if err != nil {
		respondJSON(w, http.StatusOK, memento)
		return
	}
	respondJSON(w, http.StatusOK, persisted)
}

func (h *DesktopHandler) handleDeleteMemento(w http.ResponseWriter, r *http.Request, idStr string) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memento UUID")
		return
	}
	if err := h.cfg.Repo.DeleteMemento(r.Context(), id); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			respondError(w, http.StatusNotFound, "memento not found")
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *DesktopHandler) handleGetSiteSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := publication.ResolveSiteSettings(r.Context(), h.cfg.Repo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, publication.NewStaticSiteSettings(settings))
}

type desktopPutSiteSettingsRequest struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Design          *string `json:"design"`
	DefaultLanguage *string `json:"default_language"`
	DefaultTheme    *string `json:"default_theme"`
	Accent          *string `json:"accent"`
}

func (h *DesktopHandler) handlePutSiteSettings(w http.ResponseWriter, r *http.Request) {
	var req desktopPutSiteSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	var issues []domain.Issue
	if req.Accent != nil && *req.Accent != "" && !accentPattern.MatchString(*req.Accent) {
		issues = append(issues, domain.Issue{Field: "accent", Code: "invalid_format"})
	}
	if len(issues) > 0 {
		respondJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "validation failed", "issues": issues})
		return
	}
	current, err := publication.ResolveSiteSettings(r.Context(), h.cfg.Repo)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if req.Title != nil {
		current.Title = *req.Title
	}
	if req.Description != nil {
		current.Description = *req.Description
	}
	if req.Design != nil {
		current.Design = *req.Design
	}
	if req.DefaultLanguage != nil {
		current.DefaultLanguage = *req.DefaultLanguage
	}
	if req.DefaultTheme != nil {
		current.DefaultTheme = *req.DefaultTheme
	}
	if req.Accent != nil {
		current.Accent = *req.Accent
	}
	if err := h.cfg.Repo.UpsertSiteSettings(r.Context(), &current); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, publication.NewStaticSiteSettings(current))
}

func (h *DesktopHandler) handleSiteInfo(w http.ResponseWriter, _ *http.Request) {
	h.mu.RLock()
	outDir := h.publicDir
	h.mu.RUnlock()
	previewPort := h.cfg.PreviewPort
	if h.cfg.PreviewPortFn != nil {
		previewPort = h.cfg.PreviewPortFn()
	}
	manifestFile := filepath.Join(outDir, filepath.FromSlash(publication.ManifestPath))
	_, statErr := os.Stat(manifestFile)
	respondJSON(w, http.StatusOK, map[string]any{
		"out_dir":        outDir,
		"preview_port":   previewPort,
		"spa_ready":      true,
		"artifact_ready": statErr == nil,
	})
}

func (h *DesktopHandler) handleCompile(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	outDir := h.publicDir
	h.mu.RUnlock()

	writer := &publication.FileArtifactWriter{Root: outDir}
	media := publication.FileMediaSource{Root: h.cfg.MediaRoot}
	report, err := (publication.StaticCompiler{}).Compile(r.Context(), publication.Input{}, h.cfg.Repo, media, writer)
	if err != nil {
		_ = writer.Abort()
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	removed, err := writer.Finalize()
	if err != nil {
		_ = writer.Abort()
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	report.Removed = len(removed)
	respondJSON(w, http.StatusOK, report)
}

func (h *DesktopHandler) handleTemplates(w http.ResponseWriter, _ *http.Request) {
	kinds := h.cfg.Registry.Kinds()
	tpls := make(map[string]domain.Template, len(kinds))
	for _, k := range kinds {
		if tpl, ok := h.cfg.Registry.Template(k); ok {
			tpls[k] = tpl
		}
	}
	respondJSON(w, http.StatusOK, tpls)
}

func (h *DesktopHandler) serveStaticArtifact(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	outDir := h.publicDir
	h.mu.RUnlock()
	http.FileServer(http.Dir(outDir)).ServeHTTP(w, r)
}

func (h *DesktopHandler) serveStaticUI(w http.ResponseWriter, r *http.Request) {
	targetFS := h.adminFS
	if h.cfg.Mode == "reader" {
		targetFS = h.readerFS
	}
	clean := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if clean == "" {
		clean = "index.html"
	}
	data, err := fs.ReadFile(targetFS, clean)
	if err != nil {
		// SPA fallback: return index.html for unknown client routes
		indexData, indexErr := fs.ReadFile(targetFS, "index.html")
		if indexErr != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexData)
		return
	}
	ctype := "application/octet-stream"
	switch filepath.Ext(clean) {
	case ".html":
		ctype = "text/html; charset=utf-8"
	case ".js", ".mjs":
		ctype = "text/javascript; charset=utf-8"
	case ".css":
		ctype = "text/css; charset=utf-8"
	case ".json":
		ctype = "application/json"
	case ".svg":
		ctype = "image/svg+xml"
	case ".png":
		ctype = "image/png"
	}
	w.Header().Set("Content-Type", ctype)
	_, _ = w.Write(data)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
