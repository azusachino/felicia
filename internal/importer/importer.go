// Package importer is the auto-ingest application service (the "A" of A+E): it
// seeds ingested fields from the sources and never overwrites authored ones.
package importer

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"

	"github.com/azusachino/felicia/internal/domain"
)

// DefaultEpsilon is the RDP simplification tolerance (~10m at these latitudes),
// per ADR 0009.
const DefaultEpsilon = 0.0001

// JourneyStore is the narrow slice of the repository the importer needs. The
// full domain.Repository satisfies it.
type JourneyStore interface {
	GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error)
	UpsertJourney(ctx context.Context, journey *domain.Journey) error
}

// Importer joins a TrackSource to the journey store.
type Importer struct {
	tracks  domain.TrackSource
	store   JourneyStore
	epsilon float64
}

// New builds an Importer. A non-positive epsilon falls back to DefaultEpsilon.
func New(tracks domain.TrackSource, store JourneyStore, epsilon float64) *Importer {
	if epsilon <= 0 {
		epsilon = DefaultEpsilon
	}
	return &Importer{tracks: tracks, store: store, epsilon: epsilon}
}

// SyncRoute fetches the journey's Dawarich tracks, RDP-simplifies them, and
// writes the union into gps_route. It is a no-op when gps_route is authored, so
// re-import never clobbers a hand-edited route (design §5).
func (im *Importer) SyncRoute(ctx context.Context, journeyID uuid.UUID) error {
	j, err := im.store.GetJourney(ctx, journeyID)
	if err != nil {
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}
	if slices.Contains(j.AuthoredFields, "gps_route") {
		return nil
	}

	routes, err := im.tracks.FetchRoutes(ctx, j.DateStart, endOfDay(j.DateEnd))
	if err != nil {
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}

	raw := make(orb.MultiLineString, 0, len(routes))
	for _, r := range routes {
		if len(r.Line) >= 2 {
			raw = append(raw, r.Line)
		}
	}
	j.GPSRoute = simplify.DouglasPeucker(im.epsilon).MultiLineString(raw)

	if err := im.store.UpsertJourney(ctx, j); err != nil {
		return fmt.Errorf("sync route %s: %w", journeyID, err)
	}
	return nil
}

// SyncVisits returns the journey's Dawarich visits as derived-place candidates.
// They are not persisted here — they become mementos through admin curation.
func (im *Importer) SyncVisits(ctx context.Context, journeyID uuid.UUID) ([]domain.Visit, error) {
	j, err := im.store.GetJourney(ctx, journeyID)
	if err != nil {
		return nil, fmt.Errorf("sync visits %s: %w", journeyID, err)
	}
	visits, err := im.tracks.FetchVisits(ctx, j.DateStart, endOfDay(j.DateEnd))
	if err != nil {
		return nil, fmt.Errorf("sync visits %s: %w", journeyID, err)
	}
	return visits, nil
}

// endOfDay returns the inclusive end of the journey's last date, so a track or
// visit anywhere on that day is captured by the source's overlap filter.
func endOfDay(d time.Time) time.Time {
	return d.AddDate(0, 0, 1).Add(-time.Second)
}
