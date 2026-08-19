package intake

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

type serviceRouteSource struct{}

func (serviceRouteSource) FetchRoutes(context.Context, time.Time, time.Time) ([]domain.Route, error) {
	start := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	return []domain.Route{{Line: orb.LineString{{135, 35}, {135.001, 35.001}}, Points: []domain.TrackPoint{{Coord: orb.Point{135, 35}, At: start}, {Coord: orb.Point{135.001, 35.001}, At: start.Add(time.Hour)}}}}, nil
}

type serviceStore struct {
	candidates map[string]*domain.StopCandidate
	reviews    int
}

func (s *serviceStore) GetStopCandidate(_ context.Context, id uuid.UUID) (*domain.StopCandidate, error) {
	for _, c := range s.candidates {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (s *serviceStore) ListStopCandidatesByJourney(_ context.Context, journeyID uuid.UUID) ([]*domain.StopCandidate, error) {
	var out []*domain.StopCandidate
	for _, c := range s.candidates {
		if c.JourneyID == journeyID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (s *serviceStore) UpsertStopCandidate(_ context.Context, candidate *domain.StopCandidate) error {
	if candidate.ID == uuid.Nil {
		candidate.ID = uuid.New()
	}
	s.candidates[candidate.Identity.Key] = candidate
	return nil
}
func (s *serviceStore) ApplyStopReview(_ context.Context, _ *domain.StopReviewPatch) error {
	s.reviews++
	return nil
}

// serviceJourneyStore is the narrow JourneySyncStore seam Apply uses to
// persist derived date bounds.
type serviceJourneyStore struct {
	journey *domain.Journey
	upserts int
}

func (s *serviceJourneyStore) GetJourney(_ context.Context, id uuid.UUID) (*domain.Journey, error) {
	if s.journey == nil || s.journey.ID != id {
		return nil, domain.ErrNotFound
	}
	return s.journey, nil
}

func (s *serviceJourneyStore) ApplyIngestJourneyPatch(_ context.Context, patch *domain.IngestJourneyPatch) error {
	current := s.journey
	if current == nil || current.ID != patch.Journey.ID {
		current = &domain.Journey{ID: patch.Journey.ID, JournalID: patch.Journey.JournalID}
	}
	domain.MergeIngestJourney(current, patch)
	s.journey = current
	s.upserts++
	return nil
}

func TestServicePlansAppliesAndReviewsThroughPorts(t *testing.T) {
	journeyID := uuid.New()
	store := &serviceStore{candidates: make(map[string]*domain.StopCandidate)}
	service := NewService(store, nil)
	plan, err := service.Plan(context.Background(), PlanRequest{JourneyID: journeyID, Sources: SourceSet{Routes: serviceRouteSource{}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Stops) != 1 {
		t.Fatalf("stops = %d, want 1", len(plan.Stops))
	}
	if err := service.Apply(context.Background(), plan); err != nil {
		t.Fatal(err)
	}
	if len(store.candidates) != 1 {
		t.Fatalf("persisted candidates = %d, want 1", len(store.candidates))
	}
	if err := service.Review(context.Background(), &domain.StopReviewPatch{CandidateID: plan.Stops[0].ID, State: domain.CandidateKept}); err != nil {
		t.Fatal(err)
	}
	if store.reviews != 1 {
		t.Fatalf("reviews = %d, want 1", store.reviews)
	}
}

// Acceptance criterion 2 of issue #57: a date the author set by hand survives
// ingest, while the one they left alone picks up the derived value. Asserted
// at the runtime seam so both providers inherit the same behaviour — SQLite's
// journey upsert has no per-field guard of its own.
func TestServiceApplyRespectsAuthoredDateBounds(t *testing.T) {
	authored := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	derivedStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name           string
		authoredFields []string
		wantStart      time.Time
		wantUpserts    int
	}{
		{"unauthored dates adopt the derivation", nil, derivedStart, 1},
		{"an authored date_start is preserved", []string{"date_start"}, authored, 1},
		{"authoring both dates writes nothing at all", []string{"date_start", "date_end"}, authored, 0},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			journeyID := uuid.New()
			journeys := &serviceJourneyStore{journey: &domain.Journey{
				ID: journeyID, DateStart: authored, DateEnd: authored, AuthoredFields: testCase.authoredFields,
			}}
			service := NewService(&serviceStore{candidates: make(map[string]*domain.StopCandidate)}, journeys)

			plan, err := service.Plan(context.Background(), PlanRequest{JourneyID: journeyID, Sources: SourceSet{Routes: serviceRouteSource{}}})
			if err != nil {
				t.Fatal(err)
			}
			if !plan.DateStart.Equal(derivedStart) {
				t.Fatalf("plan.DateStart = %s, want the route's day %s", plan.DateStart, derivedStart)
			}
			if err := service.Apply(context.Background(), plan); err != nil {
				t.Fatal(err)
			}
			if got := journeys.journey.DateStart; !got.Equal(testCase.wantStart) {
				t.Errorf("journey.DateStart = %s, want %s", got, testCase.wantStart)
			}
			if journeys.upserts != testCase.wantUpserts {
				t.Errorf("journey upserts = %d, want %d", journeys.upserts, testCase.wantUpserts)
			}
		})
	}
}

// A composition that never wired a journey store (the CLI's plan-only path)
// must still apply cleanly rather than fail or panic.
func TestServiceApplyWithoutJourneyStoreSkipsDateBounds(t *testing.T) {
	store := &serviceStore{candidates: make(map[string]*domain.StopCandidate)}
	service := NewService(store, nil)
	plan, err := service.Plan(context.Background(), PlanRequest{JourneyID: uuid.New(), Sources: SourceSet{Routes: serviceRouteSource{}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Apply(context.Background(), plan); err != nil {
		t.Fatalf("apply without a journey store: %v", err)
	}
}
