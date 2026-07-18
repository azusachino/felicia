package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core"
	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/runtime/importer"
	"github.com/azusachino/felicia/server/api"
)

func loadKinds(t *testing.T) *domain.Registry {
	t.Helper()
	subFS, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		t.Fatalf("failed to locate embedded kinds: %v", err)
	}
	reg, err := domain.LoadRegistry(subFS)
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}
	return reg
}

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
	transitLegs  []*domain.TransitLeg
	createdLeg   *domain.TransitLegInput
	displayRoute orb.MultiLineString
	snappedPoint *orb.Point
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		journeys: make(map[uuid.UUID]*domain.Journey),
		mementos: make(map[uuid.UUID]*domain.Memento),
		photos:   make(map[uuid.UUID]*domain.MementoPhoto),
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

func (m *mockRepository) GetMementoBySourceIdentity(_ context.Context, source domain.SourceIdentity) (*domain.Memento, error) {
	for _, mem := range m.mementos {
		if mem.SourceIdentity != nil && *mem.SourceIdentity == source {
			return mem, nil
		}
	}
	return nil, domain.ErrNotFound
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

func (m *mockRepository) ApplyManualMementoPatch(_ context.Context, patch *domain.ManualMementoPatch) error {
	if existing, ok := m.mementos[patch.Memento.ID]; ok {
		if patch.ExpectedRevision != nil && *patch.ExpectedRevision != existing.Revision {
			return domain.ErrWriteConflict
		}
		patch.Memento.Revision = existing.Revision + 1
	} else {
		patch.Memento.Revision = 1
	}
	memento := patch.Memento
	if patch.State != "" {
		memento.State = patch.State
	}
	for _, field := range patch.Fields {
		found := false
		for _, existing := range memento.AuthoredFields {
			if existing == field {
				found = true
				break
			}
		}
		if !found {
			memento.AuthoredFields = append(memento.AuthoredFields, field)
		}
	}
	m.mementos[memento.ID] = memento
	return nil
}

func (m *mockRepository) ApplyIngestMementoPatch(_ context.Context, patch *domain.IngestMementoPatch) error {
	m.mementos[patch.Memento.ID] = patch.Memento
	return nil
}

func (m *mockRepository) ResetMockJournal(_ context.Context, journalID uuid.UUID) error {
	for id, journey := range m.journeys {
		if journey.JournalID == journalID {
			delete(m.journeys, id)
		}
	}
	for id := range m.mementos {
		delete(m.mementos, id)
	}
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

func TestServerGetTemplates(t *testing.T) {
	reg := loadKinds(t)

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
	reg := loadKinds(t)

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
		"state":       "published",
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

func TestServerAllowsIncompleteDraftButRejectsInvalidCompleteGeometry(t *testing.T) {
	reg := loadKinds(t)
	repo := newMockRepository()
	srv := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})

	draft := map[string]any{
		"id": uuid.New(), "kind": "live", "state": "draft", "kind_data": map[string]any{},
	}
	body, _ := json.Marshal(draft)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("incomplete draft status = %d, body = %s", w.Code, w.Body)
	}

	complete := map[string]any{
		"id": uuid.New(), "kind": "live", "state": "published",
		"occurred_at": "2026-03-20T10:00:00Z", "occurred_tz": "Asia/Tokyo",
		"kind_data": map[string]any{
			"artist": "羊文学",
			"venue":  map[string]any{"name": "日本武道館", "coords": []float64{139.7495, 35.6933}},
			"date":   "2026-03-22T18:30:00+09:00",
		},
		"geom": map[string]any{"type": "Point", "coordinates": []float64{181, 35}},
	}
	body, _ = json.Marshal(complete)
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid complete geometry status = %d, body = %s", w.Code, w.Body)
	}
}

func TestServerManualMementoPatchOwnsFieldsServerSide(t *testing.T) {
	reg := loadKinds(t)
	repo := newMockRepository()
	srv := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	journeyID := uuid.New()
	mementoID := uuid.New()
	payload := map[string]any{
		"id":          mementoID,
		"journey_id":  journeyID,
		"kind":        "live",
		"seq":         1,
		"occurred_at": "2026-03-20T10:00:00Z",
		"occurred_tz": "Asia/Tokyo",
		"title":       "Live show",
		"place":       "Tokyo",
		"kind_data": map[string]any{
			"artist": "羊文学",
			"venue":  map[string]any{"name": "日本武道館", "coords": []float64{139.7495, 35.6933}},
			"date":   "2026-03-22T18:30:00+09:00",
		},
		// This client-controlled list is intentionally ignored by the handler.
		"authored_fields": []string{"source_ref"},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body)
	}
	memento := repo.mementos[mementoID]
	if memento.State != domain.MementoDraft {
		t.Errorf("state = %q, want draft", memento.State)
	}
	for _, field := range memento.AuthoredFields {
		if field == "source_ref" {
			t.Fatal("client must not be able to author source_ref")
		}
	}
	if len(memento.AuthoredFields) == 0 {
		t.Fatal("manual patch should record server-derived authored fields")
	}
}

func TestServerRejectsStaleMementoRevision(t *testing.T) {
	reg := loadKinds(t)
	repo := newMockRepository()
	mementoID := uuid.New()
	repo.mementos[mementoID] = &domain.Memento{ID: mementoID, Revision: 3}
	srv := api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	payload := map[string]any{
		"id": mementoID, "kind": "live", "state": "draft", "expected_revision": int64(2),
		"occurred_at": "2026-03-20T10:00:00Z", "occurred_tz": "Asia/Tokyo", "kind_data": map[string]any{},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("stale revision status = %d, body = %s", w.Code, w.Body)
	}
}

func TestServerIngestEndpoints(t *testing.T) {
	reg := loadKinds(t)
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
	reg := loadKinds(t)
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
	published := &domain.Memento{
		ID: uuid.New(), JourneyID: journey.ID, Kind: "stamp", Place: "Tokyo",
		OccurredAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), Geom: orb.Point{139.7, 35.6},
		State: domain.MementoPublished,
	}
	repo.mementos[published.ID] = published
	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+journey.ID.String()+".json", nil))
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

func TestGetPublicJourneysExcludesEmpty(t *testing.T) {
	repo := newMockRepository()
	base := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	withMementos := &domain.Journey{ID: uuid.New(), JournalID: uuid.New(), Slug: "kyushu", Title: "Kyushu"}
	empty := &domain.Journey{ID: uuid.New(), JournalID: uuid.New(), Slug: "bare", Title: "Bare"}
	repo.journeys[withMementos.ID] = withMementos
	repo.journeys[empty.ID] = empty
	repo.mementos[uuid.New()] = &domain.Memento{
		ID: uuid.New(), JourneyID: withMementos.ID, Kind: "stamp", Place: "Kagoshima",
		OccurredAt: base, Geom: orb.Point{130.6, 31.6}, State: domain.MementoPublished,
	}

	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys.json", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body)
	}

	var list []struct {
		Slug         string `json:"slug"`
		MementoCount int    `json:"memento_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&list); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected only non-empty journeys, got %d: %s", len(list), w.Body)
	}
	if list[0].Slug != "kyushu" || list[0].MementoCount == 0 {
		t.Errorf("expected the journey with mementos, got %+v", list[0])
	}
}

func TestPublicEndpointsExposeOnlyPublishedMementos(t *testing.T) {
	repo := newMockRepository()
	base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)

	publishedJourney := &domain.Journey{ID: uuid.New(), JournalID: uuid.New(), Slug: "tokyo", Title: "Tokyo", Place: "Tokyo",
		DateStart: base, DateEnd: base.AddDate(0, 0, 2), AuthoredFields: []string{}}
	draftJourney := &domain.Journey{ID: uuid.New(), JournalID: publishedJourney.JournalID, Slug: "osaka-draft", Title: "Osaka draft", Place: "Osaka",
		DateStart: base.AddDate(0, 1, 0), DateEnd: base.AddDate(0, 1, 1), AuthoredFields: []string{}}
	repo.journeys[publishedJourney.ID] = publishedJourney
	repo.journeys[draftJourney.ID] = draftJourney

	essay := "published essay"
	draftEssay := "draft essay must not leak"
	published := &domain.Memento{ID: uuid.New(), JourneyID: publishedJourney.ID, Kind: "stamp", Seq: 1,
		OccurredAt: base.Add(12 * time.Hour), OccurredTZ: "Asia/Tokyo", Geom: orb.Point{139.7, 35.6},
		Title: "published stub", Place: "Akihabara", Essay: &essay, State: domain.MementoPublished}
	draft := &domain.Memento{ID: uuid.New(), JourneyID: publishedJourney.ID, Kind: "stamp", Seq: 2,
		OccurredAt: base.Add(13 * time.Hour), OccurredTZ: "Asia/Tokyo", Geom: orb.Point{139.8, 35.7},
		Title: "secret draft", Place: "Ueno", Essay: &draftEssay, State: domain.MementoDraft}
	draftOnly := &domain.Memento{ID: uuid.New(), JourneyID: draftJourney.ID, Kind: "stamp", Seq: 1,
		OccurredAt: base.AddDate(0, 1, 0), OccurredTZ: "Asia/Tokyo", Geom: orb.Point{135.5, 34.7},
		Title: "unpublished", Place: "Namba", State: domain.MementoDraft}
	repo.mementos[published.ID] = published
	repo.mementos[draft.ID] = draft
	repo.mementos[draftOnly.ID] = draftOnly

	handler := api.NewServer(repo, nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	// The index lists only journeys with published content, counting
	// published mementos only.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("index status = %d (%s)", w.Code, w.Body)
	}
	var index []struct {
		Slug         string `json:"slug"`
		MementoCount int    `json:"memento_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&index); err != nil {
		t.Fatalf("decode index: %v", err)
	}
	if len(index) != 1 || index[0].Slug != "tokyo" {
		t.Fatalf("index = %+v, want only the journey with published content", index)
	}
	if index[0].MementoCount != 1 {
		t.Errorf("memento_count = %d, want published-only count 1", index[0].MementoCount)
	}

	// The mementos endpoint returns only published mementos, with the
	// authored essay present.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+publishedJourney.ID.String()+"/mementos", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("mementos status = %d (%s)", w.Code, w.Body)
	}
	body := w.Body.String()
	if bytes.Contains([]byte(body), []byte(draftEssay)) {
		t.Errorf("draft memento leaked on the public API: %s", body)
	}
	var mementos []struct {
		Title string  `json:"title"`
		Essay *string `json:"essay"`
	}
	if err := json.Unmarshal([]byte(body), &mementos); err != nil {
		t.Fatalf("decode mementos: %v", err)
	}
	if len(mementos) != 1 || mementos[0].Title != "published stub" {
		t.Fatalf("mementos = %+v, want the single published memento", mementos)
	}
	if mementos[0].Essay == nil || *mementos[0].Essay != essay {
		t.Errorf("essay = %v, want the authored essay", mementos[0].Essay)
	}

	// A journey without published mementos has no public projection.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+draftJourney.ID.String()+"/mementos", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("draft-only journey mementos status = %d, want 404 (%s)", w.Code, w.Body)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/journeys/"+draftJourney.ID.String(), nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("draft-only journey detail status = %d, want 404 (%s)", w.Code, w.Body)
	}
}
