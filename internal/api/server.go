// Package api implements the Go API server router and handlers.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/importer"
	"github.com/azusachino/felicia/internal/publication"
)

// Server represents the API server.
type Server struct {
	repo                  domain.Repository
	registry              *domain.Registry
	cache                 *CacheManager
	logger                *slog.Logger
	importer              *importer.Importer // may be nil when no sources are configured
	transitSegmentLengthM float64
}

const defaultTransitSegmentLengthM = 100000

// RouteConfig controls the route-curation values used by HTTP handlers.
type RouteConfig struct {
	TransitSegmentLengthM float64
}

// NewServer creates a new API server instance. A nil logger falls back to
// slog.Default. A nil importer disables the ingest endpoints (503).
func NewServer(repo domain.Repository, registry *domain.Registry, cache *CacheManager, logger *slog.Logger, imp *importer.Importer, routeConfig RouteConfig) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if routeConfig.TransitSegmentLengthM <= 0 {
		routeConfig.TransitSegmentLengthM = defaultTransitSegmentLengthM
	}
	return &Server{
		repo:                  repo,
		registry:              registry,
		cache:                 cache,
		logger:                logger,
		importer:              imp,
		transitSegmentLengthM: routeConfig.TransitSegmentLengthM,
	}
}

// requestLogger is a chi middleware that emits one structured slog line per
// request with method, path, status, size, and latency.
func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			defer func() {
				logger.Info("http request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"bytes", ww.BytesWritten(),
					"duration_ms", time.Since(start).Milliseconds(),
				)
			}()
			next.ServeHTTP(ww, r)
		})
	}
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(requestLogger(s.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	// Public Read-Only Query API (Valkey Cached)
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/journeys", s.handleGetPublicJourneys)
		r.Get("/journeys/{id}", s.handleGetPublicJourneyDetails)
		r.Get("/journeys/{id}/mementos", s.handleGetPublicMementos)
	})

	// Authoring Admin API (Valkey Invalidation on Write)
	r.Route("/api/admin", func(r chi.Router) {
		r.Get("/templates", s.handleGetTemplates)
		r.Post("/journals", s.handleCreateJournal)
		r.Post("/journals/{id}/reset-mock", s.handleResetMockJournal)
		r.Get("/journeys", s.handleListJourneys)
		r.Get("/journeys/{id}", s.handleGetJourney)
		r.Post("/journeys", s.handleUpsertJourney)
		r.Post("/journeys/{id}/legs", s.handleCreateTransitLeg)
		r.Post("/journeys/{id}/snap", s.handleSnapToRoute)
		r.Get("/journeys/{id}/mementos", s.handleListMementos)
		r.Get("/mementos/{id}", s.handleGetMemento)
		r.Post("/mementos", s.handleUpsertMemento)
		r.Post("/photos", s.handleUpsertPhoto)
		r.Get("/mementos/{id}/translations", s.handleListTranslations)
		r.Post("/translations", s.handleUpsertTranslation)

		// Ingest triggers (auto-seed from the configured sources)
		r.Post("/journeys/{id}/sync-route", s.handleSyncRoute)
		r.Get("/journeys/{id}/visits", s.handleSyncVisits)
		r.Get("/journeys/{id}/tray", s.handlePhotoTray)
	})

	return r
}

type createTransitLegRequest struct {
	Origin      []float64 `json:"origin"`
	Dest        []float64 `json:"dest"`
	OriginLabel *string   `json:"origin_label,omitempty"`
	DestLabel   *string   `json:"dest_label,omitempty"`
}

// handleCreateTransitLeg adds an authored transit segment. PostGIS builds its
// great-circle geometry; the response returns the composed display route.
func (s *Server) handleCreateTransitLeg(w http.ResponseWriter, r *http.Request) {
	journeyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}

	var req createTransitLegRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	origin, err := parseCoordinate(req.Origin, "origin")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	dest, err := parseCoordinate(req.Dest, "dest")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	legs, err := s.repo.ListTransitLegsByJourney(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	seq := 0
	for _, leg := range legs {
		seq = max(seq, leg.Seq+1)
	}
	leg := &domain.TransitLegInput{
		ID:          uuid.New(),
		JourneyID:   journeyID,
		Seq:         seq,
		OriginLabel: req.OriginLabel,
		DestLabel:   req.DestLabel,
		Origin:      origin,
		Dest:        dest,
		SegmentLenM: s.transitSegmentLengthM,
	}
	if err := s.repo.CreateTransitLeg(r.Context(), leg); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.cache.InvalidateAll(r.Context())

	route, err := s.repo.GetDisplayRoute(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"status":        "ok",
		"leg_id":        leg.ID,
		"display_route": toGeoJSON(route),
	})
}

func parseCoordinate(coord []float64, name string) (orb.Point, error) {
	if len(coord) != 2 {
		return orb.Point{}, fmt.Errorf("%s must be [lng, lat]", name)
	}
	lng, lat := coord[0], coord[1]
	if math.IsNaN(lng) || math.IsInf(lng, 0) || math.IsNaN(lat) || math.IsInf(lat, 0) || lng < -180 || lng > 180 || lat < -90 || lat > 90 {
		return orb.Point{}, fmt.Errorf("%s has invalid longitude or latitude", name)
	}
	return orb.Point{lng, lat}, nil
}

type snapToRouteRequest struct {
	Point []float64 `json:"point"`
}

// handleSnapToRoute projects a proposed memento coordinate onto the journey's
// composed route (GPS track plus authored transit legs).
func (s *Server) handleSnapToRoute(w http.ResponseWriter, r *http.Request) {
	journeyID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}

	var req snapToRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	point, err := parseCoordinate(req.Point, "point")
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	snapped, err := s.repo.SnapToRoute(r.Context(), journeyID, point)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if snapped == nil {
		respondError(w, http.StatusUnprocessableEntity, "journey has no route to snap to")
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"point": toGeoJSON(*snapped)})
}

// Templates endpoint

func (s *Server) handleGetTemplates(w http.ResponseWriter, _ *http.Request) {
	kinds := s.registry.Kinds()
	tpls := make(map[string]domain.Template)
	for _, k := range kinds {
		if tpl, ok := s.registry.Template(k); ok {
			tpls[k] = tpl
		}
	}
	respondJSON(w, http.StatusOK, tpls)
}

// Journey handlers (Admin)

type createJournalRequest struct {
	ID uuid.UUID `json:"id"`
}

func (s *Server) handleCreateJournal(w http.ResponseWriter, r *http.Request) {
	var req createJournalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}
	if req.ID == uuid.Nil {
		req.ID = uuid.Must(uuid.NewV7())
	}

	journal := &domain.Journal{ID: req.ID, CreatedAt: time.Now().UTC()}
	if err := s.repo.CreateJournal(r.Context(), journal); err != nil {
		// Seed data is intentionally rerunnable. Treat an existing journal as
		// success and let the journey upserts refresh its contents.
		if existing, getErr := s.repo.GetJournal(r.Context(), req.ID); getErr == nil {
			respondJSON(w, http.StatusOK, existing)
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, journal)
}

func (s *Server) handleListJourneys(w http.ResponseWriter, r *http.Request) {
	js, err := s.repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if js == nil {
		js = make([]*domain.Journey, 0)
	}
	respondJSON(w, http.StatusOK, js)
}

func (s *Server) handleResetMockJournal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journal UUID")
		return
	}
	if err := s.repo.ResetMockJournal(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetJourney(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	j, err := s.repo.GetJourney(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "journey not found")
		return
	}
	respondJSON(w, http.StatusOK, j)
}

type upsertJourneyRequest struct {
	ID             uuid.UUID     `json:"id"`
	JournalID      uuid.UUID     `json:"journal_id"`
	Slug           string        `json:"slug"`
	SourceRef      *string       `json:"source_ref,omitempty"`
	Title          string        `json:"title"`
	Place          string        `json:"place"`
	Country        *string       `json:"country,omitempty"`
	Region         *string       `json:"region,omitempty"`
	DateStart      string        `json:"date_start"`
	DateEnd        string        `json:"date_end"`
	GPSRoute       [][][]float64 `json:"gps_route,omitempty"`
	AuthoredFields []string      `json:"authored_fields"`
}

func (s *Server) handleUpsertJourney(w http.ResponseWriter, r *http.Request) {
	var req upsertJourneyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}

	if req.AuthoredFields == nil {
		req.AuthoredFields = []string{}
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

	var gpsRoute orb.MultiLineString
	if len(req.GPSRoute) > 0 {
		for _, segment := range req.GPSRoute {
			var ls orb.LineString
			for _, coord := range segment {
				if len(coord) >= 2 {
					ls = append(ls, orb.Point{coord[0], coord[1]})
				}
			}
			gpsRoute = append(gpsRoute, ls)
		}
	}

	journey := &domain.Journey{
		ID:             req.ID,
		JournalID:      req.JournalID,
		Slug:           req.Slug,
		SourceRef:      req.SourceRef,
		Title:          req.Title,
		Place:          req.Place,
		Country:        req.Country,
		Region:         req.Region,
		DateStart:      start,
		DateEnd:        end,
		GPSRoute:       gpsRoute,
		AuthoredFields: req.AuthoredFields,
	}

	if err := s.repo.UpsertJourney(r.Context(), journey); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Memento handlers (Admin)

func (s *Server) handleListMementos(w http.ResponseWriter, r *http.Request) {
	journeyIDStr := chi.URLParam(r, "id")
	journeyID, err := uuid.Parse(journeyIDStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	ms, err := s.repo.ListMementosByJourney(r.Context(), journeyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, ms)
}

func (s *Server) handleGetMemento(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memento UUID")
		return
	}
	m, err := s.repo.GetMemento(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, "memento not found")
		return
	}
	respondJSON(w, http.StatusOK, m)
}

type mementoGeom struct {
	Type        string `json:"type"` // "Point" or "LineString"
	Coordinates any    `json:"coordinates"`
}

type upsertMementoRequest struct {
	ID             uuid.UUID      `json:"id"`
	JourneyID      uuid.UUID      `json:"journey_id"`
	Kind           string         `json:"kind"`
	Seq            int            `json:"seq"`
	OccurredAt     string         `json:"occurred_at"` // RFC3339
	OccurredTZ     string         `json:"occurred_tz"`
	Geom           *mementoGeom   `json:"geom"`
	Title          string         `json:"title"`
	Place          string         `json:"place"`
	Vendor         *string        `json:"vendor,omitempty"`
	Essay          *string        `json:"essay,omitempty"`
	PriceAmount    *int64         `json:"price_amount,omitempty"`
	PriceCurrency  *string        `json:"price_currency,omitempty"`
	KindData       map[string]any `json:"kind_data"`
	SourceRef      *string        `json:"source_ref,omitempty"`
	AuthoredFields []string       `json:"authored_fields"`
	OrphanedAt     *string        `json:"orphaned_at,omitempty"`
}

func (s *Server) handleUpsertMemento(w http.ResponseWriter, r *http.Request) {
	var req upsertMementoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}

	if req.AuthoredFields == nil {
		req.AuthoredFields = []string{}
	}

	// 1. Template registry validation
	tpl, ok := s.registry.Template(req.Kind)
	if !ok {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "kind template not registered",
			"issues": []domain.Issue{{Code: "kind_not_registered"}},
		})
		return
	}

	issues := domain.Validate(tpl, req.KindData)
	if len(issues) > 0 {
		respondJSON(w, http.StatusBadRequest, map[string]any{
			"error":  "validation failed",
			"issues": issues,
		})
		return
	}

	occurred, err := time.Parse(time.RFC3339, req.OccurredAt)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid occurred_at timestamp format (RFC3339)")
		return
	}

	var orphaned *time.Time
	if req.OrphanedAt != nil {
		o, err := time.Parse(time.RFC3339, *req.OrphanedAt)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid orphaned_at timestamp format")
			return
		}
		orphaned = &o
	}

	var geom orb.Geometry
	if req.Geom != nil {
		switch req.Geom.Type {
		case "Point":
			if coords, ok := req.Geom.Coordinates.([]any); ok && len(coords) >= 2 {
				x, _ := coords[0].(float64)
				y, _ := coords[1].(float64)
				geom = orb.Point{x, y}
			}
		case "LineString":
			if coords, ok := req.Geom.Coordinates.([]any); ok {
				var ls orb.LineString
				for _, val := range coords {
					if ptVal, ok := val.([]any); ok && len(ptVal) >= 2 {
						x, _ := ptVal[0].(float64)
						y, _ := ptVal[1].(float64)
						ls = append(ls, orb.Point{x, y})
					}
				}
				geom = ls
			}
		}
	}

	kindDataRaw, err := json.Marshal(req.KindData)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to serialize kind_data")
		return
	}

	memento := &domain.Memento{
		ID:             req.ID,
		JourneyID:      req.JourneyID,
		Kind:           req.Kind,
		Seq:            req.Seq,
		OccurredAt:     occurred,
		OccurredTZ:     req.OccurredTZ,
		Geom:           geom,
		Title:          req.Title,
		Place:          req.Place,
		Vendor:         req.Vendor,
		Essay:          req.Essay,
		PriceAmount:    req.PriceAmount,
		PriceCurrency:  req.PriceCurrency,
		KindData:       kindDataRaw,
		SourceRef:      req.SourceRef,
		AuthoredFields: req.AuthoredFields,
		OrphanedAt:     orphaned,
	}

	if err := s.repo.UpsertMemento(r.Context(), memento); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Photo handler (Admin)

type upsertPhotoRequest struct {
	ID          uuid.UUID  `json:"id"`
	MementoID   uuid.UUID  `json:"memento_id"`
	ObjectKey   string     `json:"object_key"`
	ContentHash string     `json:"content_hash"`
	Caption     *string    `json:"caption,omitempty"`
	Seq         int        `json:"seq"`
	TakenAt     *time.Time `json:"taken_at,omitempty"`
	SourceRef   *string    `json:"source_ref,omitempty"`
}

func (s *Server) handleUpsertPhoto(w http.ResponseWriter, r *http.Request) {
	var req upsertPhotoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}

	photo := &domain.MementoPhoto{
		ID:          req.ID,
		MementoID:   req.MementoID,
		ObjectKey:   req.ObjectKey,
		ContentHash: req.ContentHash,
		Caption:     req.Caption,
		Seq:         req.Seq,
		TakenAt:     req.TakenAt,
		SourceRef:   req.SourceRef,
	}

	if err := s.repo.UpsertPhoto(r.Context(), photo); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Translation handlers (Admin)

func (s *Server) handleListTranslations(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid memento UUID")
		return
	}
	ts, err := s.repo.ListTranslations(r.Context(), "memento", id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, ts)
}

type upsertTranslationRequest struct {
	ID         uuid.UUID `json:"id"`
	OwnerType  string    `json:"owner_type"`
	OwnerID    uuid.UUID `json:"owner_id"`
	Lang       string    `json:"lang"`
	Field      string    `json:"field"`
	Value      string    `json:"value"`
	Provenance string    `json:"provenance"`
}

func (s *Server) handleUpsertTranslation(w http.ResponseWriter, r *http.Request) {
	var req upsertTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request JSON")
		return
	}

	translation := &domain.Translation{
		ID:         req.ID,
		OwnerType:  req.OwnerType,
		OwnerID:    req.OwnerID,
		Lang:       req.Lang,
		Field:      req.Field,
		Value:      req.Value,
		Provenance: req.Provenance,
	}

	if err := s.repo.UpsertTranslation(r.Context(), translation); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.cache.InvalidateAll(r.Context())
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Ingest handlers (Admin) — trigger the auto-seed importer from the sources.

func (s *Server) ingestReady(w http.ResponseWriter) bool {
	if s.importer == nil {
		respondError(w, http.StatusServiceUnavailable, "ingest sources not configured")
		return false
	}
	return true
}

func (s *Server) writeImporterErr(w http.ResponseWriter, err error) {
	if errors.Is(err, importer.ErrNoTrackSource) || errors.Is(err, importer.ErrNoPhotoSource) {
		respondError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	respondError(w, http.StatusInternalServerError, err.Error())
}

// handleSyncRoute pulls the journey's Dawarich tracks, RDP-simplifies, and
// writes gps_route, then returns the composed display route.
func (s *Server) handleSyncRoute(w http.ResponseWriter, r *http.Request) {
	if !s.ingestReady(w) {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	if err := s.importer.SyncRoute(r.Context(), id); err != nil {
		s.writeImporterErr(w, err)
		return
	}
	s.cache.InvalidateAll(r.Context())

	route, err := s.repo.GetDisplayRoute(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"status": "ok", "gps_route": toGeoJSON(route)})
}

type visitResponse struct {
	Coord      []float64 `json:"coord"` // [lng, lat]
	Label      string    `json:"label"`
	Arrive     string    `json:"arrive"`
	Depart     string    `json:"depart"`
	Confidence float64   `json:"confidence"`
	SourceRef  string    `json:"source_ref"`
}

// handleSyncVisits returns the journey's Dawarich visits as derived-place
// candidates for curation (not persisted).
func (s *Server) handleSyncVisits(w http.ResponseWriter, r *http.Request) {
	if !s.ingestReady(w) {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	visits, err := s.importer.SyncVisits(r.Context(), id)
	if err != nil {
		s.writeImporterErr(w, err)
		return
	}
	out := make([]visitResponse, 0, len(visits))
	for _, v := range visits {
		out = append(out, visitResponse{
			Coord:      []float64{v.Coord.X(), v.Coord.Y()},
			Label:      v.Label,
			Arrive:     v.Arrive.Format(time.RFC3339),
			Depart:     v.Depart.Format(time.RFC3339),
			Confidence: v.Confidence,
			SourceRef:  v.SourceRef,
		})
	}
	respondJSON(w, http.StatusOK, out)
}

type photoTrayItem struct {
	ID        string    `json:"id"`
	At        string    `json:"at"`
	Coord     []float64 `json:"coord,omitempty"` // omitted when the photo has no GPS
	Checksum  string    `json:"checksum"`
	SourceRef string    `json:"source_ref"`
}

// handlePhotoTray returns the journey's Immich photos as drag-to-snap tray
// candidates (not persisted).
func (s *Server) handlePhotoTray(w http.ResponseWriter, r *http.Request) {
	if !s.ingestReady(w) {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}
	assets, err := s.importer.SyncPhotoTray(r.Context(), id)
	if err != nil {
		s.writeImporterErr(w, err)
		return
	}
	out := make([]photoTrayItem, 0, len(assets))
	for _, a := range assets {
		item := photoTrayItem{
			ID:        a.ID,
			At:        a.At.Format(time.RFC3339),
			Checksum:  a.Checksum,
			SourceRef: a.SourceRef,
		}
		if a.Coord != nil {
			item.Coord = []float64{a.Coord.X(), a.Coord.Y()}
		}
		out = append(out, item)
	}
	respondJSON(w, http.StatusOK, out)
}

// Public Read-Only Query APIs (Valkey Cached)

func (s *Server) handleGetPublicJourneys(w http.ResponseWriter, r *http.Request) {
	cacheKey := "felicia:public:journeys"
	if cached, err := s.cache.Get(r.Context(), cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}

	journeys, err := s.repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var list []publication.JourneyListItem
	for _, j := range journeys {
		mementos, err := s.repo.ListMementosByJourney(r.Context(), j.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		list = append(list, publication.NewJourneyListItem(j, mementos))
	}

	jsonData, err := json.Marshal(list)
	if err == nil {
		_ = s.cache.Set(r.Context(), cacheKey, string(jsonData))
	}

	respondJSON(w, http.StatusOK, list)
}

type publicJourney struct {
	ID             uuid.UUID                 `json:"id"`
	JournalID      uuid.UUID                 `json:"journal_id"`
	Slug           string                    `json:"slug"`
	SourceRef      *string                   `json:"source_ref,omitempty"`
	Title          string                    `json:"title"`
	Place          string                    `json:"place"`
	Country        *string                   `json:"country,omitempty"`
	Region         *string                   `json:"region,omitempty"`
	DateStart      string                    `json:"date_start"`
	DateEnd        string                    `json:"date_end"`
	GPSRoute       *geoJSONGeom              `json:"gps_route,omitempty"`
	AuthoredFields []string                  `json:"authored_fields"`
	Translations   map[string]map[string]any `json:"translations,omitempty"`
}

func (s *Server) handleGetPublicJourneyDetails(w http.ResponseWriter, r *http.Request) {
	id, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}

	cacheKey := fmt.Sprintf("felicia:public:journey:%s", id.String())
	if cached, err := s.cache.Get(r.Context(), cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}
	j, err := s.repo.GetJourney(r.Context(), id)

	if err != nil {
		respondError(w, http.StatusNotFound, "journey not found")
		return
	}

	trans, err := s.repo.ListTranslations(r.Context(), "journey", j.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	pj := publicJourney{
		ID:             j.ID,
		JournalID:      j.JournalID,
		Slug:           j.Slug,
		SourceRef:      j.SourceRef,
		Title:          j.Title,
		Place:          j.Place,
		Country:        j.Country,
		Region:         j.Region,
		DateStart:      j.DateStart.Format("2006-01-02"),
		DateEnd:        j.DateEnd.Format("2006-01-02"),
		GPSRoute:       toGeoJSON(j.GPSRoute),
		AuthoredFields: j.AuthoredFields,
		Translations:   buildAPITranslationMap(trans),
	}

	jsonData, err := json.Marshal(pj)
	if err == nil {
		_ = s.cache.Set(r.Context(), cacheKey, string(jsonData))
	}

	respondJSON(w, http.StatusOK, pj)
}

type publicMemento struct {
	ID            uuid.UUID                 `json:"id"`
	JourneyID     uuid.UUID                 `json:"journey_id"`
	Kind          string                    `json:"kind"`
	Seq           int                       `json:"seq"`
	OccurredAt    string                    `json:"occurred_at"`
	OccurredTZ    string                    `json:"occurred_tz"`
	Geom          *geoJSONGeom              `json:"geom,omitempty"`
	Title         string                    `json:"title"`
	Place         string                    `json:"place"`
	Vendor        *string                   `json:"vendor,omitempty"`
	Essay         *string                   `json:"essay,omitempty"`
	PriceAmount   *int64                    `json:"price_amount,omitempty"`
	PriceCurrency *string                   `json:"price_currency,omitempty"`
	KindData      json.RawMessage           `json:"kind_data,omitempty"`
	SourceRef     *string                   `json:"source_ref,omitempty"`
	Photos        []publicPhoto             `json:"photos,omitempty"`
	Translations  map[string]map[string]any `json:"translations,omitempty"`
}

type publicPhoto struct {
	ID          uuid.UUID `json:"id"`
	MementoID   uuid.UUID `json:"memento_id"`
	ObjectKey   string    `json:"object_key"`
	ContentHash string    `json:"content_hash"`
	Caption     *string   `json:"caption,omitempty"`
	Seq         int       `json:"seq"`
	TakenAt     *string   `json:"taken_at,omitempty"`
	SourceRef   *string   `json:"source_ref,omitempty"`
}

func (s *Server) handleGetPublicMementos(w http.ResponseWriter, r *http.Request) {
	id, parseErr := uuid.Parse(chi.URLParam(r, "id"))
	if parseErr != nil {
		respondError(w, http.StatusBadRequest, "invalid journey UUID")
		return
	}

	cacheKey := fmt.Sprintf("felicia:public:mementos:%s", id.String())
	if cached, err := s.cache.Get(r.Context(), cacheKey); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(cached))
		return
	}
	j, err := s.repo.GetJourney(r.Context(), id)

	if err != nil {
		respondError(w, http.StatusNotFound, "journey not found")
		return
	}

	mementos, err := s.repo.ListMementosByJourney(r.Context(), j.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var list []publicMemento
	for _, m := range mementos {
		mTrans, err := s.repo.ListTranslations(r.Context(), "memento", m.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		photos, err := s.repo.ListPhotosByMemento(r.Context(), m.ID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var sPhotos []publicPhoto
		for _, ph := range photos {
			var takenAtStr *string
			if ph.TakenAt != nil {
				tStr := ph.TakenAt.Format(time.RFC3339)
				takenAtStr = &tStr
			}
			sPhotos = append(sPhotos, publicPhoto{
				ID:          ph.ID,
				MementoID:   ph.MementoID,
				ObjectKey:   ph.ObjectKey,
				ContentHash: ph.ContentHash,
				Caption:     ph.Caption,
				Seq:         ph.Seq,
				TakenAt:     takenAtStr,
				SourceRef:   ph.SourceRef,
			})
		}

		var kindData json.RawMessage
		if len(m.KindData) > 0 {
			kindData = m.KindData
		}

		list = append(list, publicMemento{
			ID:            m.ID,
			JourneyID:     m.JourneyID,
			Kind:          m.Kind,
			Seq:           m.Seq,
			OccurredAt:    m.OccurredAt.Format(time.RFC3339),
			OccurredTZ:    m.OccurredTZ,
			Geom:          toGeoJSON(m.Geom),
			Title:         m.Title,
			Place:         m.Place,
			Vendor:        m.Vendor,
			Essay:         m.Essay,
			PriceAmount:   m.PriceAmount,
			PriceCurrency: m.PriceCurrency,
			KindData:      kindData,
			SourceRef:     m.SourceRef,
			Photos:        sPhotos,
			Translations:  buildAPITranslationMap(mTrans),
		})
	}

	jsonData, err := json.Marshal(list)
	if err == nil {
		_ = s.cache.Set(r.Context(), cacheKey, string(jsonData))
	}

	respondJSON(w, http.StatusOK, list)
}

// Helpers

type geoJSONGeom struct {
	Type        string      `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

func toGeoJSON(geom orb.Geometry) *geoJSONGeom {
	if geom == nil {
		return nil
	}
	switch g := geom.(type) {
	case orb.Point:
		return &geoJSONGeom{
			Type:        "Point",
			Coordinates: []float64{g.X(), g.Y()},
		}
	case orb.LineString:
		if len(g) == 0 {
			return nil
		}
		var coords [][]float64
		for _, pt := range g {
			coords = append(coords, []float64{pt.X(), pt.Y()})
		}
		return &geoJSONGeom{
			Type:        "LineString",
			Coordinates: coords,
		}
	case orb.MultiLineString:
		if len(g) == 0 {
			return nil
		}
		var coords [][][]float64
		for _, ls := range g {
			var lsCoords [][]float64
			for _, pt := range ls {
				lsCoords = append(lsCoords, []float64{pt.X(), pt.Y()})
			}
			coords = append(coords, lsCoords)
		}
		return &geoJSONGeom{
			Type:        "MultiLineString",
			Coordinates: coords,
		}
	default:
		return nil
	}
}

func buildAPITranslationMap(translations []*domain.Translation) map[string]map[string]any {
	m := make(map[string]map[string]any)
	for _, t := range translations {
		if _, ok := m[t.Lang]; !ok {
			m[t.Lang] = make(map[string]any)
		}
		if strings.HasPrefix(t.Field, "kind_data.") {
			parts := strings.Split(t.Field, ".")
			if len(parts) == 2 {
				kindDataMap, ok := m[t.Lang]["kind_data"].(map[string]any)
				if !ok {
					kindDataMap = make(map[string]any)
					m[t.Lang]["kind_data"] = kindDataMap
				}
				kindDataMap[parts[1]] = t.Value
			} else {
				m[t.Lang][t.Field] = t.Value
			}
		} else {
			m[t.Lang][t.Field] = t.Value
		}
	}
	return m
}

// Helper responders

func respondJSON(w http.ResponseWriter, status int, val any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(val)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respondJSON(w, status, map[string]string{"error": msg})
}
