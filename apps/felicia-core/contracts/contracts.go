// Package contracts declares versioned Felicia capability traits.
//
// The interfaces describe behavior at adapter boundaries. Canonical data
// shapes and transport schemas remain versioned artifacts under contracts/.
package contracts

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// CanonicalVersion identifies the stable Felicia-owned semantic contract.
const CanonicalVersion = "felicia.canonical.v1"

// Capability identifies an optional behavior an adapter can provide.
type Capability string

const (
	// CapabilityRoutes reads normalized route evidence.
	CapabilityRoutes Capability = "routes.read"
	// CapabilityVisits reads normalized visit evidence.
	CapabilityVisits Capability = "visits.read"
	// CapabilityMedia reads normalized media evidence.
	CapabilityMedia Capability = "media.read"
	// CapabilityAuthoring writes authored mementos.
	CapabilityAuthoring Capability = "mementos.write"
	// CapabilityReview applies explicit stop review decisions.
	CapabilityReview Capability = "stops.review"
	// CapabilitySuggestions proposes non-mutating authoring changes.
	CapabilitySuggestions Capability = "suggestions.propose"
	// CapabilityPublication writes a public projection.
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

// RouteSource reads normalized route evidence for a time range.
type RouteSource interface {
	Trait
	FetchRoutes(context.Context, time.Time, time.Time) ([]domain.Route, error)
}

// VisitSource reads normalized visit evidence for a time range.
type VisitSource interface {
	Trait
	FetchVisits(context.Context, time.Time, time.Time) ([]domain.Visit, error)
}

// MediaSource reads normalized media evidence for a time range.
type MediaSource interface {
	Trait
	FetchMedia(context.Context, time.Time, time.Time) ([]domain.MediaAsset, error)
}

// AuthoringStore applies revision-protected authored memento changes.
type AuthoringStore interface {
	Trait
	GetMemento(context.Context, uuid.UUID) (*domain.Memento, error)
	ApplyManualMementoPatch(context.Context, *domain.ManualMementoPatch) error
}

// CandidateReviewStore exposes private stop candidates and explicit reviews.
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
