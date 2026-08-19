package publication

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// StaticJourney is the public journey detail projection.
type StaticJourney struct {
	ID             string           `json:"id"`
	JournalID      string           `json:"journal_id"`
	Slug           string           `json:"slug"`
	SourceRef      *string          `json:"source_ref,omitempty"`
	Title          string           `json:"title"`
	Place          string           `json:"place"`
	Country        *string          `json:"country,omitempty"`
	Region         *string          `json:"region,omitempty"`
	DateStart      string           `json:"date_start"`
	DateEnd        string           `json:"date_end"`
	GPSRoute       *GeoJSONGeometry `json:"gps_route,omitempty"`
	AuthoredFields []string         `json:"authored_fields"`
}

// StaticMemento is the public memento projection.
type StaticMemento struct {
	ID            string           `json:"id"`
	JourneyID     string           `json:"journey_id"`
	Kind          string           `json:"kind"`
	Seq           int              `json:"seq"`
	OccurredAt    string           `json:"occurred_at"`
	OccurredTZ    string           `json:"occurred_tz"`
	Geom          *GeoJSONGeometry `json:"geom,omitempty"`
	Title         string           `json:"title"`
	Place         string           `json:"place"`
	Vendor        *string          `json:"vendor,omitempty"`
	Essay         *string          `json:"essay,omitempty"`
	PriceAmount   *int64           `json:"price_amount,omitempty"`
	PriceCurrency *string          `json:"price_currency,omitempty"`
	KindData      json.RawMessage  `json:"kind_data,omitempty"`
	SourceRef     *string          `json:"source_ref,omitempty"`
	Photos        []StaticPhoto    `json:"photos,omitempty"`
}

// StaticPhoto is the public media metadata projection.
type StaticPhoto struct {
	ID          string  `json:"id"`
	MementoID   string  `json:"memento_id"`
	ObjectKey   string  `json:"object_key"`
	ContentHash string  `json:"content_hash"`
	Caption     *string `json:"caption,omitempty"`
	Seq         int     `json:"seq"`
	TakenAt     *string `json:"taken_at,omitempty"`
	SourceRef   *string `json:"source_ref,omitempty"`
}

// GeoJSONGeometry is the stable geometry shape used by the public API.
type GeoJSONGeometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

// StaticSiteSettings is the public site identity/style projection served
// identically at /api/v1/site.json (static) and /api/v1/site (live).
type StaticSiteSettings struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Design          string `json:"design"`
	DefaultLanguage string `json:"default_language"`
	DefaultTheme    string `json:"default_theme"`
	Accent          string `json:"accent"`
}

// NewStaticSiteSettings projects domain site settings into the public shape.
func NewStaticSiteSettings(settings domain.SiteSettings) StaticSiteSettings {
	return StaticSiteSettings{
		Title:           settings.Title,
		Description:     settings.Description,
		Design:          settings.Design,
		DefaultLanguage: settings.DefaultLanguage,
		DefaultTheme:    settings.DefaultTheme,
		Accent:          settings.Accent,
	}
}

// ResolveSiteSettings resolves the sole journal's site settings, falling
// back to domain.DefaultSiteSettings when none have been saved yet. Both
// StaticCompiler.Compile and the live handleGetPublicSite call this one
// function so the two surfaces can never diverge on how "absent settings"
// is interpreted.
func ResolveSiteSettings(ctx context.Context, read ReadModel) (domain.SiteSettings, error) {
	journal, err := read.GetSoleJournal(ctx)
	if err != nil {
		return domain.SiteSettings{}, err
	}
	settings, err := read.GetSiteSettings(ctx, journal.ID)
	if errors.Is(err, domain.ErrNotFound) {
		return domain.DefaultSiteSettings(journal.ID), nil
	}
	if err != nil {
		return domain.SiteSettings{}, err
	}
	return *settings, nil
}

// StaticCompiler writes the public JSON tree and referenced media files.
type StaticCompiler struct{}

// Compile emits only published mementos and their referenced public media.
func (StaticCompiler) Compile(ctx context.Context, input Input, read ReadModel, media MediaSource, output ArtifactWriter) (BuildReport, error) {
	journeys, err := read.ListJourneys(ctx)
	if err != nil {
		return BuildReport{}, fmt.Errorf("list journeys: %w", err)
	}
	allowed := make(map[uuid.UUID]bool, len(input.JourneyIDs))
	for _, id := range input.JourneyIDs {
		allowed[id] = true
	}
	SortJourneys(journeys)

	index := make([]JourneyListItem, 0, len(journeys))
	report := BuildReport{}

	// Site identity/style is independent of journey/memento content and is
	// always written, even on a fresh DB where it reflects
	// domain.DefaultSiteSettings (ADMIN-02 M2 §4, "absent settings = current
	// demo behavior").
	settings, err := ResolveSiteSettings(ctx, read)
	if err != nil {
		return report, fmt.Errorf("resolve site settings: %w", err)
	}
	if err := output.WriteJSON("api/v1/site.json", NewStaticSiteSettings(settings)); err != nil {
		return report, fmt.Errorf("write site settings: %w", err)
	}

	for _, journey := range journeys {
		if len(allowed) > 0 && !allowed[journey.ID] {
			continue
		}
		mementos, err := read.ListMementosByJourney(ctx, journey.ID)
		if err != nil {
			return report, fmt.Errorf("list mementos for %s: %w", journey.ID, err)
		}
		published := PublishedMementos(mementos)
		// A journey without published content has no public projection at
		// all — matching the live public API, which hides such journeys.
		if len(published) == 0 {
			continue
		}
		index = append(index, NewJourneyListItem(journey, published))
		report.Journeys++
		report.Mementos += len(published)

		if err := output.WriteJSON("api/v1/journeys/"+journey.ID.String()+".json", NewStaticJourney(journey)); err != nil {
			return report, fmt.Errorf("write journey %s: %w", journey.ID, err)
		}

		mementoProjection := make([]StaticMemento, 0, len(published))
		for _, memento := range published {
			photos, err := read.ListPhotosByMemento(ctx, memento.ID)
			if err != nil {
				return report, fmt.Errorf("list photos for %s: %w", memento.ID, err)
			}
			SortPhotos(photos)
			photoProjection := make([]StaticPhoto, 0, len(photos))
			for _, photo := range photos {
				if media == nil {
					return report, fmt.Errorf("media source is required for %s", photo.ObjectKey)
				}
				reader, err := media.Open(ctx, photo.ObjectKey)
				if err != nil {
					return report, fmt.Errorf("open media %s: %w", photo.ObjectKey, err)
				}
				// Originals are never published verbatim: the derivative is
				// resized and stripped of EXIF (including GPS) here, the sole
				// media egress point in the system. Sanitizing before the
				// writer is reached means there is no code path that can emit
				// an unprocessed original.
				derivative, err := SanitizePublicImage(photo.ObjectKey, reader)
				closeErr := reader.Close()
				if err != nil {
					return report, fmt.Errorf("sanitize media %s: %w", photo.ObjectKey, err)
				}
				if closeErr != nil {
					return report, fmt.Errorf("close media %s: %w", photo.ObjectKey, closeErr)
				}
				if err := output.WriteMedia(photo.ObjectKey, bytes.NewReader(derivative)); err != nil {
					return report, fmt.Errorf("write media %s: %w", photo.ObjectKey, err)
				}
				report.Media++
				photoProjection = append(photoProjection, NewStaticPhoto(photo))
			}
			mementoProjection = append(mementoProjection, NewStaticMemento(memento, photoProjection))
		}
		if err := output.WriteJSON("api/v1/journeys/"+journey.ID.String()+"/mementos.json", mementoProjection); err != nil {
			return report, fmt.Errorf("write mementos %s: %w", journey.ID, err)
		}
	}
	if err := output.WriteJSON("api/v1/journeys.json", index); err != nil {
		return report, fmt.Errorf("write journey index: %w", err)
	}
	return report, nil
}

// publicCoordDecimals is the decimal precision every coordinate is rounded
// to before it crosses the publication boundary, in either direction: the
// static compiler's JSON tree and the live /api/v1 handlers (which project
// through the same NewStaticJourney/NewStaticMemento — see
// apps/felicia-server/api/server.go). 4 decimal places is ~11m of ground distance at the
// equator. This is not a value chosen here: it is the precision already
// documented for this exact purpose in docs/archive/spec-gaps.md ("D2.
// Public coordinate rounding") and cross-referenced in
// docs/research/backend-stack.md and docs/research/liuaaron-teardown.md, so
// it is reused rather than picked independently.
//
// ADR-0025 requires the static artifact to never contain "unrounded private
// geometry". Rounding here — the sole place both the route (a journey's
// passive GPS trace) and every memento Geom (frequently derived from that
// same trace at a stop's timestamp, see apps/felicia-runtime/importer) are projected to
// GeoJSON — makes the guarantee hold for every importer that could have
// populated the stored geometry, not just the one path a defect happened to
// skip.
const publicCoordDecimals = 4

// roundCoord rounds a single coordinate ordinate (longitude or latitude) to
// publicCoordDecimals, so no full-precision value survives the round trip
// through this package regardless of how many decimals the stored value
// carried.
func roundCoord(v float64) float64 {
	const scale = 1e4 // 10^publicCoordDecimals
	return math.Round(v*scale) / scale
}

func geometry(value orb.Geometry) *GeoJSONGeometry {
	switch value := value.(type) {
	case orb.Point:
		return &GeoJSONGeometry{Type: "Point", Coordinates: []float64{roundCoord(value.X()), roundCoord(value.Y())}}
	case orb.MultiLineString:
		// An empty route is omitted from the projection entirely rather
		// than encoded as a degenerate geometry.
		if len(value) == 0 {
			return nil
		}
		coordinates := make([][][]float64, 0, len(value))
		for _, line := range value {
			points := make([][]float64, 0, len(line))
			for _, point := range line {
				points = append(points, []float64{roundCoord(point.X()), roundCoord(point.Y())})
			}
			coordinates = append(coordinates, points)
		}
		return &GeoJSONGeometry{Type: "MultiLineString", Coordinates: coordinates}
	case orb.LineString:
		if len(value) == 0 {
			return nil
		}
		points := make([][]float64, 0, len(value))
		for _, point := range value {
			points = append(points, []float64{roundCoord(point.X()), roundCoord(point.Y())})
		}
		return &GeoJSONGeometry{Type: "LineString", Coordinates: points}
	default:
		return nil
	}
}
