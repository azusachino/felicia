// Package api implements the Go API server router and handlers.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/domain"
)

// Server represents the API server.
type Server struct {
	repo     domain.Repository
	registry *domain.Registry
}

// NewServer creates a new API server instance.
func NewServer(repo domain.Repository, registry *domain.Registry) *Server {
	return &Server{
		repo:     repo,
		registry: registry,
	}
}

// Handler returns the HTTP handler for the server.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	r.Route("/api/admin", func(r chi.Router) {
		r.Get("/templates", s.handleGetTemplates)
		r.Get("/journeys", s.handleListJourneys)
		r.Get("/journeys/{id}", s.handleGetJourney)
		r.Post("/journeys", s.handleUpsertJourney)
		r.Get("/journeys/{id}/mementos", s.handleListMementos)
		r.Get("/mementos/{id}", s.handleGetMemento)
		r.Post("/mementos", s.handleUpsertMemento)
		r.Post("/photos", s.handleUpsertPhoto)
		r.Get("/mementos/{id}/translations", s.handleListTranslations)
		r.Post("/translations", s.handleUpsertTranslation)
	})

	return r
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

// Journey handlers

func (s *Server) handleListJourneys(w http.ResponseWriter, r *http.Request) {
	js, err := s.repo.ListJourneys(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, js)
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

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Memento handlers

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

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Photo handler

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

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Translation handlers

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

	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
