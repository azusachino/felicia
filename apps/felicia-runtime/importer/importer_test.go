package importer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

type fakeTracks struct {
	routes  []domain.Route
	visits  []domain.Visit
	gotFrom time.Time
	gotTo   time.Time
	fetched bool
}

func (f *fakeTracks) FetchRoutes(_ context.Context, from, to time.Time) ([]domain.Route, error) {
	f.fetched = true
	f.gotFrom, f.gotTo = from, to
	return f.routes, nil
}

func (f *fakeTracks) FetchVisits(_ context.Context, from, to time.Time) ([]domain.Visit, error) {
	f.gotFrom, f.gotTo = from, to
	return f.visits, nil
}

type fakePhotos struct {
	assets []domain.PhotoAsset
}

func (f *fakePhotos) FetchAssets(_ context.Context, _, _ time.Time) ([]domain.PhotoAsset, error) {
	return f.assets, nil
}

type fakeStore struct {
	journeys map[uuid.UUID]*domain.Journey
	upserted *domain.Journey
}

type fakeObservationStore struct {
	run           *domain.ImportRun
	observations  []*domain.SourceObservation
	missingRunID  uuid.UUID
	missingSystem string
	seen          []string
	finished      domain.ImportRunStatus
}

func (s *fakeObservationStore) CreateImportRun(_ context.Context, run *domain.ImportRun) error {
	if run.ID == uuid.Nil {
		run.ID = uuid.Must(uuid.NewV7())
	}
	s.run = run
	return nil
}

func (s *fakeObservationStore) FinishImportRun(_ context.Context, _ uuid.UUID, status domain.ImportRunStatus, _ time.Time, _ *string) error {
	s.finished = status
	return nil
}

func (s *fakeObservationStore) RecordSourceObservation(_ context.Context, observation *domain.SourceObservation) error {
	s.observations = append(s.observations, observation)
	return nil
}

func (s *fakeObservationStore) MarkMissingSourceObservations(_ context.Context, runID uuid.UUID, sourceSystem string, seenExternalIDs []string) error {
	s.missingRunID, s.missingSystem, s.seen = runID, sourceSystem, seenExternalIDs
	return nil
}

func (s *fakeStore) GetJourney(_ context.Context, id uuid.UUID) (*domain.Journey, error) {
	j, ok := s.journeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return j, nil
}

// ApplyIngestJourneyPatch mirrors the providers: masked fields are applied only
// where the stored row has not claimed authorship, and the stored authored mask
// is left alone (ADR-0033).
func (s *fakeStore) ApplyIngestJourneyPatch(_ context.Context, patch *domain.IngestJourneyPatch) error {
	current, ok := s.journeys[patch.Journey.ID]
	if !ok {
		current = &domain.Journey{ID: patch.Journey.ID, JournalID: patch.Journey.JournalID}
	}
	domain.MergeIngestJourney(current, patch)
	s.journeys[current.ID] = current
	s.upserted = current
	return nil
}

func newJourney(authored ...string) *domain.Journey {
	return &domain.Journey{
		ID:             uuid.New(),
		DateStart:      time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		AuthoredFields: authored,
	}
}

func TestPersistObservationsRecordsCanonicalRunAndMarksMissing(t *testing.T) {
	store := &fakeObservationStore{}
	observedAt := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	run, err := PersistObservations(context.Background(), store, "immich", []domain.Observation{
		{Kind: domain.ObservationPhoto, Source: domain.SourceIdentity{System: "immich", ExternalID: "asset-1"}, ObservedAt: observedAt, Confidence: 0.9, Payload: domain.MediaAsset{ID: "asset-1"}},
	})
	if err != nil {
		t.Fatalf("PersistObservations: %v", err)
	}
	if run.ID == uuid.Nil || run.ID.Version() != 7 {
		t.Fatalf("run ID = %s, want UUIDv7", run.ID)
	}
	if run.Status != domain.ImportRunSucceeded || store.finished != domain.ImportRunSucceeded {
		t.Fatalf("run status = %q/%q", run.Status, store.finished)
	}
	if len(store.observations) != 1 || string(store.observations[0].Payload) == "" {
		t.Fatalf("observations = %+v", store.observations)
	}
	if store.missingRunID != run.ID || store.missingSystem != "immich" || len(store.seen) != 1 || store.seen[0] != "asset-1" {
		t.Fatalf("missing marker = %s/%s/%v", store.missingRunID, store.missingSystem, store.seen)
	}
}

func TestPersistObservationsKeepsMediaKindsAndMemoryLinks(t *testing.T) {
	store := &fakeObservationStore{}
	link := domain.MemoryLink{EntityType: "memento", EntityID: uuid.New(), Relation: "attached_to"}
	_, err := PersistObservations(context.Background(), store, "media-source", []domain.Observation{{
		Kind:       domain.ObservationMedia,
		Source:     domain.SourceIdentity{System: "media-source", ExternalID: "video-1"},
		ObservedAt: time.Now().UTC(),
		Payload: domain.MediaAsset{
			ID: "video-1", Kind: domain.MediaVideo, URI: "https://cdn.example/video.mp4", MemoryLinks: []domain.MemoryLink{link},
		},
	}})
	if err != nil {
		t.Fatalf("PersistObservations: %v", err)
	}
	var payload domain.MediaAsset
	if err := json.Unmarshal(store.observations[0].Payload, &payload); err != nil {
		t.Fatalf("decode canonical media payload: %v", err)
	}
	if payload.Kind != domain.MediaVideo || len(payload.MemoryLinks) != 1 || payload.MemoryLinks[0].Relation != "attached_to" {
		t.Fatalf("payload = %#v, want linked video asset", payload)
	}
}

func TestPersistObservationsRejectsMixedSourceRun(t *testing.T) {
	store := &fakeObservationStore{}
	_, err := PersistObservations(context.Background(), store, "immich", []domain.Observation{{
		Kind: domain.ObservationVisit, Source: domain.SourceIdentity{System: "dawarich", ExternalID: "visit-1"}, Payload: map[string]any{},
	}})
	if err == nil || store.finished != domain.ImportRunFailed {
		t.Fatalf("error/status = %v/%q, want failed run", err, store.finished)
	}
}

func TestSyncRouteSimplifiesAndPersists(t *testing.T) {
	j := newJourney()
	// Six near-collinear points (deviations well under the 0.0001 epsilon):
	// RDP should collapse them to the two endpoints.
	route := domain.Route{Line: orb.LineString{
		{139.0, 35.0}, {139.00002, 35.1}, {139.00001, 35.2},
		{139.00002, 35.3}, {139.00001, 35.4}, {139.0, 35.5},
	}}
	tracks := &fakeTracks{routes: []domain.Route{route}}
	store := &fakeStore{journeys: map[uuid.UUID]*domain.Journey{j.ID: j}}

	im := New(tracks, nil, store, 0) // 0 -> DefaultEpsilon
	if err := im.SyncRoute(context.Background(), j.ID); err != nil {
		t.Fatalf("SyncRoute: %v", err)
	}

	if store.upserted == nil {
		t.Fatal("expected the journey to be upserted")
	}
	if len(store.upserted.GPSRoute) != 1 {
		t.Fatalf("expected 1 route segment, got %d", len(store.upserted.GPSRoute))
	}
	if got := len(store.upserted.GPSRoute[0]); got != 2 {
		t.Errorf("expected RDP to reduce 6 points to 2, got %d", got)
	}
	// The fetch window covers the whole last day.
	if !tracks.gotFrom.Equal(j.DateStart) {
		t.Errorf("from = %v, want %v", tracks.gotFrom, j.DateStart)
	}
	if !tracks.gotTo.After(j.DateEnd) {
		t.Errorf("to = %v, want after %v", tracks.gotTo, j.DateEnd)
	}
}

func TestSyncRouteSkipsAuthoredRoute(t *testing.T) {
	j := newJourney("gps_route")
	existing := orb.MultiLineString{{{1, 1}, {2, 2}}}
	j.GPSRoute = existing
	tracks := &fakeTracks{routes: []domain.Route{{Line: orb.LineString{{9, 9}, {8, 8}}}}}
	store := &fakeStore{journeys: map[uuid.UUID]*domain.Journey{j.ID: j}}

	im := New(tracks, nil, store, 0)
	if err := im.SyncRoute(context.Background(), j.ID); err != nil {
		t.Fatalf("SyncRoute: %v", err)
	}

	if tracks.fetched {
		t.Error("should not fetch tracks when gps_route is authored")
	}
	if store.upserted != nil {
		t.Error("should not upsert when gps_route is authored")
	}
	if len(j.GPSRoute) != 1 || len(j.GPSRoute[0]) != 2 || j.GPSRoute[0][0] != (orb.Point{1, 1}) {
		t.Error("authored gps_route must remain untouched")
	}
}

func TestSyncVisitsReturnsCandidates(t *testing.T) {
	j := newJourney()
	visits := []domain.Visit{
		{Label: "明治神宮", Coord: orb.Point{139.6993, 35.6764}, SourceRef: "dawarich:visit:7"},
		{Label: "道頓堀", Coord: orb.Point{135.5013, 34.6687}, SourceRef: "dawarich:visit:8"},
	}
	tracks := &fakeTracks{visits: visits}
	store := &fakeStore{journeys: map[uuid.UUID]*domain.Journey{j.ID: j}}

	im := New(tracks, nil, store, 0)
	got, err := im.SyncVisits(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("SyncVisits: %v", err)
	}
	if len(got) != 2 || got[0].Label != "明治神宮" || got[1].SourceRef != "dawarich:visit:8" {
		t.Errorf("unexpected visits: %+v", got)
	}
}

func TestSyncPhotoTrayReturnsAssets(t *testing.T) {
	j := newJourney()
	photos := &fakePhotos{assets: []domain.PhotoAsset{
		{ID: "asset-1", Coord: &orb.Point{139.6993, 35.6764}, SourceRef: "immich:asset:asset-1"},
		{ID: "asset-2", SourceRef: "immich:asset:asset-2"},
	}}
	store := &fakeStore{journeys: map[uuid.UUID]*domain.Journey{j.ID: j}}

	im := New(nil, photos, store, 0)
	got, err := im.SyncPhotoTray(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("SyncPhotoTray: %v", err)
	}
	if len(got) != 2 || got[1].Coord != nil {
		t.Errorf("unexpected tray: %+v", got)
	}
}

func TestSyncWithoutSourcesErrors(t *testing.T) {
	j := newJourney()
	store := &fakeStore{journeys: map[uuid.UUID]*domain.Journey{j.ID: j}}
	im := New(nil, nil, store, 0)

	if err := im.SyncRoute(context.Background(), j.ID); !errors.Is(err, ErrNoTrackSource) {
		t.Errorf("SyncRoute: expected ErrNoTrackSource, got %v", err)
	}
	if _, err := im.SyncPhotoTray(context.Background(), j.ID); !errors.Is(err, ErrNoPhotoSource) {
		t.Errorf("SyncPhotoTray: expected ErrNoPhotoSource, got %v", err)
	}
}
