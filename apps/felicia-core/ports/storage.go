// Package ports defines the narrow persistence contracts consumed by runtime
// use cases. Provider packages implement these interfaces; transport packages
// must not depend on a concrete provider.
package ports

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// JournalStore persists the journal root used by authoring workflows.
type JournalStore interface {
	GetJournal(ctx context.Context, id uuid.UUID) (*domain.Journal, error)
	CreateJournal(ctx context.Context, journal *domain.Journal) error
	ResetMockJournal(ctx context.Context, id uuid.UUID) error
	// GetSoleJournal returns the single journal row expected in this
	// single-tenant deployment. Returns domain.ErrNotFound when the
	// database has not been bootstrapped yet.
	GetSoleJournal(ctx context.Context) (*domain.Journal, error)
}

// SiteSettingsStore persists the journal-scoped public site identity/style
// settings (ADMIN-02 M2).
type SiteSettingsStore interface {
	GetSiteSettings(ctx context.Context, journalID uuid.UUID) (*domain.SiteSettings, error) // domain.ErrNotFound if absent
	UpsertSiteSettings(ctx context.Context, settings *domain.SiteSettings) error
}

// JourneyStore is the narrow journey seam required by runtime workflows.
type JourneyStore interface {
	GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error)
	GetJourneyBySlug(ctx context.Context, slug string) (*domain.Journey, error)
	ListJourneys(ctx context.Context) ([]*domain.Journey, error)
	UpsertJourney(ctx context.Context, journey *domain.Journey) error
}

// JourneySyncStore is the smaller seam used by source synchronization. It
// keeps import workflows independent from admin listing and lookup behavior.
// It writes exclusively through the ingest seam so an import can never take or
// clear authored ownership (ADR-0033).
type JourneySyncStore interface {
	GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error)
	ApplyIngestJourneyPatch(ctx context.Context, patch *domain.IngestJourneyPatch) error
}

// MementoStore owns memento lifecycle and optimistic-concurrency persistence.
type MementoStore interface {
	GetMemento(ctx context.Context, id uuid.UUID) (*domain.Memento, error)
	GetMementoBySourceIdentity(ctx context.Context, source domain.SourceIdentity) (*domain.Memento, error)
	ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.Memento, error)
	UpsertMemento(ctx context.Context, memento *domain.Memento) error
	ApplyManualMementoPatch(ctx context.Context, patch *domain.ManualMementoPatch) error
	ApplyIngestMementoPatch(ctx context.Context, patch *domain.IngestMementoPatch) error
	DeleteMemento(ctx context.Context, id uuid.UUID) error
}

// MediaStore persists memento media.
type MediaStore interface {
	GetPhoto(ctx context.Context, id uuid.UUID) (*domain.MementoPhoto, error)
	ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error)
	UpsertPhoto(ctx context.Context, photo *domain.MementoPhoto) error
}

// RouteStore persists authored route additions and computes the display route.
type RouteStore interface {
	CreateTransitLeg(ctx context.Context, leg *domain.TransitLegInput) error
	ListTransitLegsByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.TransitLeg, error)
	DeleteTransitLeg(ctx context.Context, id uuid.UUID) error
	GetDisplayRoute(ctx context.Context, journeyID uuid.UUID) (orb.MultiLineString, error)
	SnapToRoute(ctx context.Context, journeyID uuid.UUID, point orb.Point) (*orb.Point, error)
}

// ObservationStore persists source history independently of authored content.
type ObservationStore interface {
	CreateImportRun(ctx context.Context, run *domain.ImportRun) error
	FinishImportRun(ctx context.Context, id uuid.UUID, status domain.ImportRunStatus, finishedAt time.Time, errorMessage *string) error
	RecordSourceObservation(ctx context.Context, observation *domain.SourceObservation) error
	MarkMissingSourceObservations(ctx context.Context, runID uuid.UUID, sourceSystem string, seenExternalIDs []string) error
}

// StopCandidateStore persists private intake candidates and explicit review
// decisions. Implementations must not expose candidates through publication
// reads unless an authored memento references one.
type StopCandidateStore interface {
	GetStopCandidate(ctx context.Context, id uuid.UUID) (*domain.StopCandidate, error)
	ListStopCandidatesByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.StopCandidate, error)
	UpsertStopCandidate(ctx context.Context, candidate *domain.StopCandidate) error
	ApplyStopReview(ctx context.Context, patch *domain.StopReviewPatch) error
}
