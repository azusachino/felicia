package intake

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
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

func TestServicePlansAppliesAndReviewsThroughPorts(t *testing.T) {
	journeyID := uuid.New()
	store := &serviceStore{candidates: make(map[string]*domain.StopCandidate)}
	service := NewService(store)
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
