package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/api"
	"github.com/azusachino/felicia/internal/domain"
	"github.com/azusachino/felicia/internal/importer"
)

type fakeTrackSource struct {
	routes []domain.Route
	visits []domain.Visit
}

func (f *fakeTrackSource) FetchRoutes(context.Context, time.Time, time.Time) ([]domain.Route, error) {
	return f.routes, nil
}

func (f *fakeTrackSource) FetchVisits(context.Context, time.Time, time.Time) ([]domain.Visit, error) {
	return f.visits, nil
}

type fakePhotoSource struct {
	assets []domain.PhotoAsset
}

func (f *fakePhotoSource) FetchAssets(context.Context, time.Time, time.Time) ([]domain.PhotoAsset, error) {
	return f.assets, nil
}

// testLogger discards output so tests stay quiet.
var testLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

var _ domain.Repository = (*mockRepository)(nil)

type mockRepository struct {
	journeys     map[uuid.UUID]*domain.Journey
	mementos     map[uuid.UUID]*domain.Memento
	photos       map[uuid.UUID]*domain.MementoPhoto
	translations []*domain.Translation
	transitLegs  []*domain.TransitLeg
	createdLeg   *domain.TransitLegInput
	displayRoute orb.MultiLineString
	snappedPoint *orb.Point
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		journeys:     make(map[uuid.UUID]*domain.Journey),
		mementos:     make(map[uuid.UUID]*domain.Memento),
		photos:       make(map[uuid.UUID]*domain.MementoPhoto),
		translations: []*domain.Translation{},
	}
}

func (m *mockRepository) GetJournal(_ context.Context, id uuid.UUID) (*domain.Journal, error) {
	return &domain.Journal{ID: id, CreatedAt: time.Now()}, nil
}

func (m *mockRepository) CreateJournal(_ context.Context, _ *domain.Journal) error {
	return nil
}

func (m *mockRepository) GetJourney(_ context.Context, id uuid.UUID) (*domain.Journey, error) {
	j, ok := m.journeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return j, nil
}

func (m *mockRepository) GetJourneyBySlug(_ context.Context, slug string) (*domain.Journey, error) {
	for _, j := range m.journeys {
		if j.Slug == slug {
			return j, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockRepository) ListJourneys(_ context.Context) ([]*domain.Journey, error) {
	var list []*domain.Journey
	for _, j := range m.journeys {
		list = append(list, j)
	}
	return list, nil
}

func (m *mockRepository) UpsertJourney(_ context.Context, journey *domain.Journey) error {
	m.journeys[journey.ID] = journey
	return nil
}

func (m *mockRepository) GetMemento(_ context.Context, id uuid.UUID) (*domain.Memento, error) {
	mem, ok := m.mementos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return mem, nil
}

func (m *mockRepository) ListMementosByJourney(_ context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	var list []*domain.Memento
	for _, mem := range m.mementos {
		if mem.JourneyID == journeyID {
			list = append(list, mem)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertMemento(_ context.Context, memento *domain.Memento) error {
	m.mementos[memento.ID] = memento
	return nil
}

func (m *mockRepository) CreateTransitLeg(_ context.Context, leg *domain.TransitLegInput) error {
	m.createdLeg = leg
	return nil
}

func (m *mockRepository) ListTransitLegsByJourney(_ context.Context, _ uuid.UUID) ([]*domain.TransitLeg, error) {
	return m.transitLegs, nil
}

func (m *mockRepository) DeleteTransitLeg(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockRepository) GetDisplayRoute(_ context.Context, _ uuid.UUID) (orb.MultiLineString, error) {
	return m.displayRoute, nil
}

func (m *mockRepository) SnapToRoute(_ context.Context, _ uuid.UUID, _ orb.Point) (*orb.Point, error) {
	return m.snappedPoint, nil
}

func (m *mockRepository) GetPhoto(_ context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	ph, ok := m.photos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ph, nil
}

func (m *mockRepository) ListPhotosByMemento(_ context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	var list []*domain.MementoPhoto
	for _, ph := range m.photos {
		if ph.MementoID == mementoID {
			list = append(list, ph)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertPhoto(_ context.Context, photo *domain.MementoPhoto) error {
	m.photos[photo.ID] = photo
	return nil
}

func (m *mockRepository) ListTranslations(_ context.Context, ownerType string, ownerID uuid.UUID) ([]*domain.Translation, error) {
	var list []*domain.Translation
	for _, t := range m.translations {
		if t.OwnerType == ownerType && t.OwnerID == ownerID {
			list = append(list, t)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertTranslation(_ context.Context, translation *domain.Translation) error {
	for i, t := range m.translations {
		if t.OwnerType == translation.OwnerType && t.OwnerID == translation.OwnerID && t.Lang == translation.Lang && t.Field == translation.Field {
			m.translations[i] = translation
			return nil
		}
	}
	m.translations = append(m.translations, translation)
	return nil
}

func TestServerGetTemplates(t *testing.T) {
	reg, err := domain.LoadRegistry(os.DirFS("../../kinds"))
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}

	repo := newMockRepository()
	srv := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/api/admin/templates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]domain.Template
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := res["transit"]; !ok {
		t.Error("expected transit template to be returned")
	}
}

func TestServerUpsertMementoValidation(t *testing.T) {
	reg, err := domain.LoadRegistry(os.DirFS("../../kinds"))
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}

	repo := newMockRepository()
	srv := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	handler := srv.Handler()

	// 1. Submit invalid memento (missing required transit fields 'operator', 'from', 'to')
	payload := map[string]any{
		"id":          uuid.New(),
		"journey_id":  uuid.New(),
		"kind":        "transit",
		"seq":         1,
		"occurred_at": time.Now().Format(time.RFC3339),
		"occurred_tz": "Asia/Tokyo",
		"title":       "Invalid Transit",
		"place":       "Somewhere",
		"kind_data":   map[string]any{"line": "Yamanote Line"}, // missing 'operator', 'from', 'to'
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}

	var res map[string]any
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res["error"] != "validation failed" {
		t.Errorf("expected validation failed error, got: %v", res["error"])
	}
	issues, ok := res["issues"].([]any)
	if !ok || len(issues) == 0 {
		t.Error("expected validation issues in response")
	}
}

func TestServerIngestEndpoints(t *testing.T) {
	reg, err := domain.LoadRegistry(os.DirFS("../../kinds"))
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}
	repo := newMockRepository()
	jid := uuid.New()
	repo.journeys[jid] = &domain.Journey{
		ID:             jid,
		DateStart:      time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		AuthoredFields: []string{},
	}
	tracks := &fakeTrackSource{
		routes: []domain.Route{{Line: orb.LineString{{139.0, 35.0}, {139.7, 35.6}}}},
		visits: []domain.Visit{{Label: "明治神宮", Coord: orb.Point{139.6993, 35.6764}, SourceRef: "dawarich:visit:7"}},
	}
	photos := &fakePhotoSource{assets: []domain.PhotoAsset{{ID: "asset-1", SourceRef: "immich:asset:asset-1"}}}
	imp := importer.New(tracks, photos, repo, 0)
	handler := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, imp, api.RouteConfig{}).Handler()

	base := "/api/admin/journeys/" + jid.String()

	// sync-route writes gps_route
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", base+"/sync-route", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("sync-route: expected 200, got %d (%s)", w.Code, w.Body)
	}
	if len(repo.journeys[jid].GPSRoute) == 0 {
		t.Error("expected gps_route to be populated after sync-route")
	}

	// visits returns the derived-place candidates
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", base+"/visits", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("visits: expected 200, got %d", w.Code)
	}
	var visits []map[string]any
	_ = json.NewDecoder(w.Body).Decode(&visits)
	if len(visits) != 1 {
		t.Errorf("expected 1 visit, got %d", len(visits))
	}

	// tray returns the photo candidates
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", base+"/tray", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("tray: expected 200, got %d", w.Code)
	}
}

func TestServerIngestNotConfigured(t *testing.T) {
	reg, _ := domain.LoadRegistry(os.DirFS("../../kinds"))
	repo := newMockRepository()
	// nil importer -> ingest endpoints are unavailable.
	handler := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("POST", "/api/admin/journeys/"+uuid.NewString()+"/sync-route", nil))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when ingest is unconfigured, got %d", w.Code)
	}
}

func TestServerCreateTransitLeg(t *testing.T) {
	repo := newMockRepository()
	journeyID := uuid.New()
	repo.transitLegs = []*domain.TransitLeg{{Seq: 2}}
	repo.displayRoute = orb.MultiLineString{
		{{139.7671, 35.6812}, {139.7003, 35.6895}},
		{{139.7798, 35.5494}, {135.2381, 34.4342}},
	}
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{TransitSegmentLengthM: 50000}).Handler()

	body := bytes.NewBufferString(`{"origin":[139.7798,35.5494],"dest":[135.2381,34.4342],"origin_label":"HND","dest_label":"KIX"}`)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/journeys/"+journeyID.String()+"/legs", body)
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body)
	}
	if repo.createdLeg == nil {
		t.Fatal("expected transit leg to be created")
	}
	if repo.createdLeg.JourneyID != journeyID || repo.createdLeg.Seq != 3 {
		t.Errorf("unexpected created leg: %+v", repo.createdLeg)
	}
	if repo.createdLeg.Origin != (orb.Point{139.7798, 35.5494}) || repo.createdLeg.Dest != (orb.Point{135.2381, 34.4342}) {
		t.Errorf("unexpected leg endpoints: %+v", repo.createdLeg)
	}
	if repo.createdLeg.SegmentLenM != 50000 {
		t.Errorf("expected configured 50000m segments, got %v", repo.createdLeg.SegmentLenM)
	}

	var response struct {
		Status       string `json:"status"`
		DisplayRoute struct {
			Type        string        `json:"type"`
			Coordinates [][][]float64 `json:"coordinates"`
		} `json:"display_route"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Status != "ok" || response.DisplayRoute.Type != "MultiLineString" || len(response.DisplayRoute.Coordinates) != 2 {
		t.Errorf("unexpected response: %+v", response)
	}
}

func TestServerCreateTransitLegRejectsInvalidCoordinates(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/journeys/"+uuid.NewString()+"/legs", bytes.NewBufferString(`{"origin":[139.7],"dest":[135.2,34.4]}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid coordinates, got %d (%s)", w.Code, w.Body)
	}
}

func TestServerSnapToRoute(t *testing.T) {
	repo := newMockRepository()
	snapped := orb.Point{139.7003, 35.6895}
	repo.snappedPoint = &snapped
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/journeys/"+uuid.NewString()+"/snap", bytes.NewBufferString(`{"point":[139.71,35.69]}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body)
	}

	var response struct {
		Point struct {
			Type        string    `json:"type"`
			Coordinates []float64 `json:"coordinates"`
		} `json:"point"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Point.Type != "Point" || len(response.Point.Coordinates) != 2 || response.Point.Coordinates[0] != snapped.X() || response.Point.Coordinates[1] != snapped.Y() {
		t.Errorf("unexpected snap response: %+v", response)
	}
}

func TestServerSnapToRouteRejectsEmptyRouteAndInvalidPoint(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	emptyRoute := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/journeys/"+uuid.NewString()+"/snap", bytes.NewBufferString(`{"point":[139.71,35.69]}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(emptyRoute, req)
	if emptyRoute.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for an empty route, got %d (%s)", emptyRoute.Code, emptyRoute.Body)
	}

	invalidPoint := httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/admin/journeys/"+uuid.NewString()+"/snap", bytes.NewBufferString(`{"point":[200,35.69]}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(invalidPoint, req)
	if invalidPoint.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid point, got %d (%s)", invalidPoint.Code, invalidPoint.Body)
	}
}

func TestPublicJourneyOmitsEmptyGPSRoute(t *testing.T) {
	repo := newMockRepository()
	journey := &domain.Journey{
		ID:             uuid.New(),
		JournalID:      uuid.New(),
		Slug:           "empty-route",
		Title:          "Empty Route",
		Place:          "Tokyo",
		DateStart:      time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		AuthoredFields: []string{},
	}
	repo.journeys[journey.ID] = journey
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys/empty-route", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body)
	}

	var response map[string]json.RawMessage
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := response["gps_route"]; ok {
		t.Error("expected empty GPS route to be omitted")
	}
}
