package importer

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/domain"
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

type fakeStore struct {
	journeys map[uuid.UUID]*domain.Journey
	upserted *domain.Journey
}

func (s *fakeStore) GetJourney(_ context.Context, id uuid.UUID) (*domain.Journey, error) {
	j, ok := s.journeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return j, nil
}

func (s *fakeStore) UpsertJourney(_ context.Context, j *domain.Journey) error {
	s.journeys[j.ID] = j
	s.upserted = j
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

	im := New(tracks, store, 0) // 0 -> DefaultEpsilon
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

	im := New(tracks, store, 0)
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

	im := New(tracks, store, 0)
	got, err := im.SyncVisits(context.Background(), j.ID)
	if err != nil {
		t.Fatalf("SyncVisits: %v", err)
	}
	if len(got) != 2 || got[0].Label != "明治神宮" || got[1].SourceRef != "dawarich:visit:8" {
		t.Errorf("unexpected visits: %+v", got)
	}
}
