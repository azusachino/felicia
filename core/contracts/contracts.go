// Package contracts declares versioned Felicia capability traits.
//
// The interfaces describe behavior at adapter boundaries. Canonical data
// shapes and transport schemas remain versioned artifacts under contracts/.
package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
)

const CanonicalVersion = "felicia.canonical.v1"

type Capability string

const (
	CapabilityRoutes      Capability = "routes.read"
	CapabilityVisits      Capability = "visits.read"
	CapabilityMedia       Capability = "media.read"
	CapabilityAuthoring   Capability = "mementos.write"
	CapabilityReview      Capability = "stops.review"
	CapabilitySuggestions Capability = "suggestions.propose"
	CapabilityPublication Capability = "publication.write"
)

// Suggestion is proposal data, not an authored domain mutation.
type Suggestion struct {
	ID         uuid.UUID
	Target     string
	Operation  string
	Proposal   any
	Provider   string
	Model      string
	Confidence float64
	CreatedAt  time.Time
}

// Trait lets an adapter advertise the contract version and optional behavior
// it supports. Unsupported capabilities must be reported, not guessed.
type Trait interface {
	ContractVersion() string
	Capabilities() []Capability
}

type RouteSource interface {
	Trait
	FetchRoutes(context.Context, time.Time, time.Time) ([]domain.Route, error)
}

type VisitSource interface {
	Trait
	FetchVisits(context.Context, time.Time, time.Time) ([]domain.Visit, error)
}

type MediaSource interface {
	Trait
	FetchMedia(context.Context, time.Time, time.Time) ([]domain.MediaAsset, error)
}

type AuthoringStore interface {
	Trait
	GetMemento(context.Context, uuid.UUID) (*domain.Memento, error)
	ApplyManualMementoPatch(context.Context, *domain.ManualMementoPatch) error
}

type CandidateReviewStore interface {
	Trait
	ListStopCandidatesByJourney(context.Context, uuid.UUID) ([]*domain.StopCandidate, error)
	ApplyStopReview(context.Context, *domain.StopReviewPatch) error
}

// SuggestionStore is intentionally proposal-only. Accepting a suggestion must
// go through an explicit authoring operation with revision protection.
type SuggestionStore interface {
	Trait
	ListSuggestions(context.Context, string) ([]Suggestion, error)
	CreateSuggestion(context.Context, Suggestion) error
}
