package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
)

// SourceIdentity is the stable identity of an object in an external system.
// Local UUIDs identify felicia rows; source identities identify observations
// across repeated imports and must therefore survive a re-run.
type SourceIdentity struct {
	System     string
	ExternalID string
}

// Valid reports whether an identity is usable as an idempotency key.
func (i SourceIdentity) Valid() bool {
	return i.System != "" && i.ExternalID != ""
}

// Ref returns the namespaced form used at current adapter seams.
func (i SourceIdentity) Ref() string {
	if !i.Valid() {
		return ""
	}
	return i.System + ":" + i.ExternalID
}

// Validate returns a descriptive error for an identity that cannot support a
// durable source observation.
func (i SourceIdentity) Validate() error {
	if i.System == "" {
		return fmt.Errorf("source identity system is required")
	}
	if i.ExternalID == "" {
		return fmt.Errorf("source identity external ID is required")
	}
	return nil
}

// Provenance records where a canonical value came from and how trustworthy
// the source observation is. Authorship is deliberately separate: provenance
// describes origin, while the write model will decide whether an authored
// field may be changed.
type Provenance struct {
	Source     SourceIdentity
	ObservedAt time.Time
	Confidence float64
}

// ObservationKind identifies the canonical shape produced by an adapter.
type ObservationKind string

const (
	// ObservationRoute identifies a normalized route observation.
	ObservationRoute ObservationKind = "route"
	// ObservationVisit identifies a normalized place observation.
	ObservationVisit ObservationKind = "visit"
	// ObservationPhoto identifies a normalized media observation.
	ObservationPhoto ObservationKind = "photo"
	// ObservationMedia identifies a normalized media observation. It covers
	// assets that are not still images, such as video, links, and embeds.
	ObservationMedia ObservationKind = "media"
	// ObservationMemento identifies a normalized memento candidate.
	ObservationMemento ObservationKind = "memento"
)

// MemoryLink connects an observed or authored object to a canonical memory.
// EntityType is intentionally open for future journal, journey, and memento
// targets; adapters must not invent provider-specific foreign keys here.
type MemoryLink struct {
	EntityType string    `json:"entity_type"`
	EntityID   uuid.UUID `json:"entity_id"`
	Relation   string    `json:"relation"`
}

// Observation is the envelope shared by source-specific adapters. Payload is
// one of the canonical shapes below; provider DTOs stay outside domain.
type Observation struct {
	Kind       ObservationKind
	Source     SourceIdentity
	ObservedAt time.Time
	Confidence float64
	Payload    any
}

// ImportRunStatus describes the lifecycle of one source synchronization run.
type ImportRunStatus string

const (
	// ImportRunRunning means source collection is in progress.
	ImportRunRunning ImportRunStatus = "running"
	// ImportRunSucceeded means source collection and observation persistence completed.
	ImportRunSucceeded ImportRunStatus = "succeeded"
	// ImportRunFailed means source collection stopped with an error.
	ImportRunFailed ImportRunStatus = "failed"
)

// ImportRun records the durable boundary around one source synchronization.
type ImportRun struct {
	ID           uuid.UUID
	SourceSystem string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Status       ImportRunStatus
	ErrorMessage *string
}

// SourceObservation is a persisted canonical snapshot, never a provider DTO.
// Rows are associated with an import run so changes and missing observations
// can be investigated without modifying authored mementos.
type SourceObservation struct {
	ID         uuid.UUID
	RunID      uuid.UUID
	Source     SourceIdentity
	Kind       ObservationKind
	ObservedAt time.Time
	Confidence float64
	Payload    json.RawMessage
	Changed    bool
	OrphanedAt *time.Time
	CreatedAt  time.Time
}

// ObservationStore is the persistence seam for import history. Implementations
// store canonical payloads only; provider-specific DTOs stay in adapters.
type ObservationStore interface {
	CreateImportRun(ctx context.Context, run *ImportRun) error
	FinishImportRun(ctx context.Context, id uuid.UUID, status ImportRunStatus, finishedAt time.Time, errorMessage *string) error
	RecordSourceObservation(ctx context.Context, observation *SourceObservation) error
	MarkMissingSourceObservations(ctx context.Context, runID uuid.UUID, sourceSystem string, seenExternalIDs []string) error
}

// Ingest value objects — the normalized shapes every external source is mapped
// into at the edge (source-connectors.md). The core stays generic over these,
// never over a specific provider.

// Route is a normalized track segment: a polyline with the window it covers.
// Dawarich tracks map to this; the journey's gps_route is the union of routes.
type Route struct {
	Line       orb.LineString
	Points     []TrackPoint
	From       time.Time
	To         time.Time
	DistanceM  int
	Mode       string // dominant transport mode, e.g. "walking", "car"
	SourceRef  string // legacy adapter ref; use Provenance for new writes
	Provenance Provenance
}

// TrackPoint preserves timestamped samples when a source provides them. The
// public route does not require these samples; local planning uses them to
// derive visits when no VisitSource is available.
type TrackPoint struct {
	Coord orb.Point
	At    time.Time
}

// Visit is a normalized stay — a derived place (place-as-derived-visit, ADR
// 0005). Dawarich visits map to this; a memento anchors to the nearest visit.
type Visit struct {
	Coord      orb.Point // [lng, lat]
	Label      string
	Arrive     time.Time
	Depart     time.Time
	Confidence float64
	SourceRef  string // legacy adapter ref; use Provenance for new writes
	Provenance Provenance
}

// MediaKind identifies a canonical asset attached to a memento.
type MediaKind string

const (
	// MediaImage is a raster image or image derivative.
	MediaImage MediaKind = "image"
	// MediaVideo is a playable video asset.
	MediaVideo MediaKind = "video"
	// MediaAudio is a playable audio asset.
	MediaAudio MediaKind = "audio"
	// MediaDocument is a downloadable document such as a scanned ticket PDF.
	MediaDocument MediaKind = "document"
	// MediaLink is an external URL shown as a link card.
	MediaLink MediaKind = "link"
	// MediaEmbed is a provider-approved external embed URL.
	MediaEmbed MediaKind = "embed"
)

// MediaAsset is a normalized media item. Immich assets map to this; Coord is
// nil when the asset carries no location (filled from the track by timestamp,
// or on drop). Raw arbitrary HTML is never part of the canonical value.
type MediaAsset struct {
	ID          string
	Kind        MediaKind
	At          time.Time
	Coord       *orb.Point
	Checksum    string
	SourceRef   string // legacy adapter ref; use Provenance for new writes
	Provenance  Provenance
	URI         string
	MIME        string
	Title       string
	Provider    string
	EmbedURL    string
	MemoryLinks []MemoryLink
}

// PhotoAsset remains as a source-compatibility alias while adapters migrate
// to the broader canonical media vocabulary.
type PhotoAsset = MediaAsset

// MementoCandidate is the canonical candidate shape shared by source
// adapters and the future ingest patch writer. It is intentionally a
// candidate, not a persisted Memento: authored fields and publication state
// belong to the write side.
type MementoCandidate struct {
	Source      SourceIdentity
	StopKey     string
	Kind        string
	OccurredAt  time.Time
	OccurredTZ  string
	Geom        orb.Geometry
	Title       string
	Place       string
	KindData    map[string]any
	Media       []MediaAsset
	MemoryLinks []MemoryLink
	Provenance  Provenance
}

// RouteSource yields normalized route segments for a time range. Dawarich and
// local GPX adapters implement this capability.
type RouteSource interface {
	FetchRoutes(ctx context.Context, from, to time.Time) ([]Route, error)
}

// VisitSource yields semantic stays for a time range. Dawarich implements it;
// a local GPX planner may derive the same Visit shape when it is unavailable.
type VisitSource interface {
	FetchVisits(ctx context.Context, from, to time.Time) ([]Visit, error)
}

// TrackSource is the legacy combined capability retained while runtime callers
// migrate to RouteSource and VisitSource independently.
type TrackSource interface {
	RouteSource
	VisitSource
}

// PhotoSource yields photo assets for a time range. Immich implements it.
type PhotoSource interface {
	FetchAssets(ctx context.Context, from, to time.Time) ([]PhotoAsset, error)
}
