package intake

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/core/ports"
)

// ErrNoCandidateStore means private intake persistence was not configured.
var ErrNoCandidateStore = errors.New("stop candidate store is required")

// SourceSet is the normalized capability bundle used by both CLI and server
// compositions. Visits and media are optional; routes are required for a plan.
type SourceSet struct {
	Routes domain.RouteSource
	Visits domain.VisitSource
	Media  domain.PhotoSource
}

// PlanRequest describes one read-only intake operation.
type PlanRequest struct {
	JourneyID         uuid.UUID
	From              time.Time
	To                time.Time
	SourceFingerprint string
	Config            PlanConfig
	Sources           SourceSet
}

// Service coordinates source collection, deterministic planning, persistence,
// and review while keeping provider implementations outside the runtime.
type Service struct {
	Candidates ports.StopCandidateStore
}

// NewService creates an intake application service.
func NewService(candidates ports.StopCandidateStore) *Service {
	return &Service{Candidates: candidates}
}

// Plan collects normalized source data and returns a read-only draft plan.
func (s *Service) Plan(ctx context.Context, request PlanRequest) (DraftPlan, error) {
	if request.JourneyID == uuid.Nil {
		return DraftPlan{}, errors.New("journey ID is required")
	}
	if request.Sources.Routes == nil {
		return DraftPlan{}, errors.New("route source is required")
	}
	routes, err := request.Sources.Routes.FetchRoutes(ctx, request.From, request.To)
	if err != nil {
		return DraftPlan{}, fmt.Errorf("fetch routes: %w", err)
	}
	var visits []domain.Visit
	if request.Sources.Visits != nil {
		visits, err = request.Sources.Visits.FetchVisits(ctx, request.From, request.To)
		if err != nil {
			return DraftPlan{}, fmt.Errorf("fetch visits: %w", err)
		}
	}
	var media []domain.MediaAsset
	if request.Sources.Media != nil {
		media, err = request.Sources.Media.FetchAssets(ctx, request.From, request.To)
		if err != nil {
			return DraftPlan{}, fmt.Errorf("fetch media: %w", err)
		}
	}
	return BuildPlan(PlanInput{JourneyID: request.JourneyID, SourceFingerprint: request.SourceFingerprint, Routes: routes, Visits: visits, Media: media}, request.Config)
}

// Apply persists the plan's private stop candidates. It does not create public
// mementos and is safe to repeat because candidate identity is stable.
func (s *Service) Apply(ctx context.Context, plan DraftPlan) error {
	if s == nil || s.Candidates == nil {
		return ErrNoCandidateStore
	}
	for index := range plan.Stops {
		if err := s.Candidates.UpsertStopCandidate(ctx, &plan.Stops[index]); err != nil {
			return fmt.Errorf("apply stop %s: %w", plan.Stops[index].Identity.Key, err)
		}
	}
	return nil
}

// Review applies one explicit author decision to a private candidate.
func (s *Service) Review(ctx context.Context, patch *domain.StopReviewPatch) error {
	if s == nil || s.Candidates == nil {
		return ErrNoCandidateStore
	}
	return s.Candidates.ApplyStopReview(ctx, patch)
}

// List returns private candidates for admin review.
func (s *Service) List(ctx context.Context, journeyID uuid.UUID) ([]*domain.StopCandidate, error) {
	if s == nil || s.Candidates == nil {
		return nil, ErrNoCandidateStore
	}
	return s.Candidates.ListStopCandidatesByJourney(ctx, journeyID)
}

// Get returns one private candidate for an admin response.
func (s *Service) Get(ctx context.Context, candidateID uuid.UUID) (*domain.StopCandidate, error) {
	if s == nil || s.Candidates == nil {
		return nil, ErrNoCandidateStore
	}
	return s.Candidates.GetStopCandidate(ctx, candidateID)
}
