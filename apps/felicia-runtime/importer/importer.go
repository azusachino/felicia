// Package importer is the auto-ingest application service (the "A" of A+E): it
// seeds ingested fields from the sources and never overwrites authored ones.
package importer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
)

// DefaultEpsilon is the RDP simplification tolerance (~10m at these latitudes),
// per ADR 0009.
const DefaultEpsilon = 0.0001

// Sentinel errors for when a capability's source was not wired.
var (
	ErrNoTrackSource = errors.New("importer: no track source configured")
	ErrNoPhotoSource = errors.New("importer: no photo source configured")
)

// JourneyStore is the narrow slice of the repository the importer needs. The
// full domain.Repository satisfies it.
type JourneyStore = ports.JourneySyncStore

// Importer joins the track and photo sources to the journey store. Either source
// may be nil; the matching Sync* method then returns ErrNo{Track,Photo}Source.
type Importer struct {
	tracks  domain.TrackSource
	photos  domain.PhotoSource
	store   JourneyStore
	epsilon float64
	logger  *slog.Logger
}

// PersistObservations records one complete canonical source response. It keeps
// provider DTOs out of storage, compares observations through the repository,
// and marks identities absent from the response as orphaned.
func PersistObservations(ctx context.Context, store ports.ObservationStore, sourceSystem string, observations []domain.Observation) (*domain.ImportRun, error) {
	run := &domain.ImportRun{SourceSystem: sourceSystem, Status: domain.ImportRunRunning, StartedAt: time.Now().UTC()}
	if err := store.CreateImportRun(ctx, run); err != nil {
		return nil, fmt.Errorf("start %s import: %w", sourceSystem, err)
	}
	fail := func(err error) (*domain.ImportRun, error) {
		message := err.Error()
		_ = store.FinishImportRun(ctx, run.ID, domain.ImportRunFailed, time.Now().UTC(), &message)
		return run, err
	}
	seen := make([]string, 0, len(observations))
	for _, observation := range observations {
		if observation.Source.System != sourceSystem {
			return fail(fmt.Errorf("observation source %q does not match run source %q", observation.Source.System, sourceSystem))
		}
		payload, err := json.Marshal(observation.Payload)
		if err != nil {
			return fail(fmt.Errorf("marshal %s observation %s: %w", observation.Kind, observation.Source.Ref(), err))
		}
		if err := store.RecordSourceObservation(ctx, &domain.SourceObservation{
			RunID: run.ID, Source: observation.Source, Kind: observation.Kind,
			ObservedAt: observation.ObservedAt, Confidence: observation.Confidence, Payload: payload,
		}); err != nil {
			return fail(fmt.Errorf("record %s observation %s: %w", observation.Kind, observation.Source.Ref(), err))
		}
		if !slices.Contains(seen, observation.Source.ExternalID) {
			seen = append(seen, observation.Source.ExternalID)
		}
	}
	if err := store.MarkMissingSourceObservations(ctx, run.ID, sourceSystem, seen); err != nil {
		return fail(fmt.Errorf("mark missing %s observations: %w", sourceSystem, err))
	}
	if err := store.FinishImportRun(ctx, run.ID, domain.ImportRunSucceeded, time.Now().UTC(), nil); err != nil {
		return nil, fmt.Errorf("finish %s import: %w", sourceSystem, err)
	}
	run.Status = domain.ImportRunSucceeded
	return run, nil
}

// New builds an Importer. A non-positive epsilon falls back to DefaultEpsilon.
func New(tracks domain.TrackSource, photos domain.PhotoSource, store JourneyStore, epsilon float64) *Importer {
	return NewWithLogger(tracks, photos, store, epsilon, slog.Default())
}

// NewWithLogger builds an Importer with structured operation logging.
func NewWithLogger(tracks domain.TrackSource, photos domain.PhotoSource, store JourneyStore, epsilon float64, logger *slog.Logger) *Importer {
	if epsilon <= 0 {
		epsilon = DefaultEpsilon
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Importer{tracks: tracks, photos: photos, store: store, epsilon: epsilon, logger: logger}
}

// Tracks returns the configured track source, or nil when none was wired.
// Callers that need read-only route/visit access outside the Sync* methods
// (the admin intake-plan endpoint, for one) use this instead of duplicating
// source composition.
func (im *Importer) Tracks() domain.TrackSource { return im.tracks }

// Photos returns the configured photo source, or nil when none was wired.
func (im *Importer) Photos() domain.PhotoSource { return im.photos }

// SyncRoute fetches the journey's Dawarich tracks, RDP-simplifies them, and
// writes the union into gps_route. It is a no-op when gps_route is authored, so
// re-import never clobbers a hand-edited route (design §5).
func (im *Importer) SyncRoute(ctx context.Context, journeyID uuid.UUID) error {
	started := time.Now()
	outcome := "success"
	defer func() {
		im.logger.Info("import operation", "operation", "sync_route", "journey_id", journeyID, "outcome", outcome, "duration_ms", time.Since(started).Milliseconds())
	}()
	if im.tracks == nil {
		outcome = "source_unavailable"
		return ErrNoTrackSource
	}
	j, err := im.store.GetJourney(ctx, journeyID)
	if err != nil {
		outcome = "error"
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}
	if slices.Contains(j.AuthoredFields, "gps_route") {
		return nil
	}

	routes, err := im.tracks.FetchRoutes(ctx, j.DateStart, endOfDay(j.DateEnd))
	if err != nil {
		outcome = "error"
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}

	raw := make(orb.MultiLineString, 0, len(routes))
	for _, r := range routes {
		if len(r.Line) >= 2 {
			raw = append(raw, r.Line)
		}
	}
	j.GPSRoute = simplify.DouglasPeucker(im.epsilon).MultiLineString(raw)

	// Written through the ingest seam so the authored mask survives the write
	// even though the early return above already skipped an authored route.
	if err := im.store.ApplyIngestJourneyPatch(ctx, &domain.IngestJourneyPatch{Journey: j, Fields: []string{"gps_route"}}); err != nil {
		outcome = "error"
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}
	return nil
}

// SyncVisits returns the journey's Dawarich visits as derived-place candidates.
// They are not persisted here — they become mementos through admin curation.
func (im *Importer) SyncVisits(ctx context.Context, journeyID uuid.UUID) ([]domain.Visit, error) {
	started := time.Now()
	outcome := "success"
	defer func() {
		im.logger.Info("import operation", "operation", "sync_visits", "journey_id", journeyID, "outcome", outcome, "duration_ms", time.Since(started).Milliseconds())
	}()
	if im.tracks == nil {
		outcome = "source_unavailable"
		return nil, ErrNoTrackSource
	}
	j, err := im.store.GetJourney(ctx, journeyID)
	if err != nil {
		outcome = "error"
		return nil, fmt.Errorf("sync visits %s: %w", journeyID, err)
	}
	visits, err := im.tracks.FetchVisits(ctx, j.DateStart, endOfDay(j.DateEnd))
	if err != nil {
		outcome = "error"
		return nil, fmt.Errorf("sync visits %s: %w", journeyID, err)
	}
	return visits, nil
}

// SyncPhotoTray returns the journey's Immich photos for the date range as tray
// candidates for drag-to-snap curation. They are not persisted here.
func (im *Importer) SyncPhotoTray(ctx context.Context, journeyID uuid.UUID) ([]domain.PhotoAsset, error) {
	started := time.Now()
	outcome := "success"
	defer func() {
		im.logger.Info("import operation", "operation", "sync_photo_tray", "journey_id", journeyID, "outcome", outcome, "duration_ms", time.Since(started).Milliseconds())
	}()
	if im.photos == nil {
		outcome = "source_unavailable"
		return nil, ErrNoPhotoSource
	}
	j, err := im.store.GetJourney(ctx, journeyID)
	if err != nil {
		outcome = "error"
		return nil, fmt.Errorf("sync photo tray %s: %w", journeyID, err)
	}
	assets, err := im.photos.FetchAssets(ctx, j.DateStart, endOfDay(j.DateEnd))
	if err != nil {
		outcome = "error"
		return nil, fmt.Errorf("sync photo tray %s: %w", journeyID, err)
	}
	return assets, nil
}

// endOfDay returns the inclusive end of the journey's last date, so a track or
// visit anywhere on that day is captured by the source's overlap filter.
func endOfDay(d time.Time) time.Time {
	return d.AddDate(0, 0, 1).Add(-time.Second)
}
