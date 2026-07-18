package publication

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
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
				if err := output.WriteMedia(photo.ObjectKey, reader); err != nil {
					_ = reader.Close()
					return report, fmt.Errorf("write media %s: %w", photo.ObjectKey, err)
				}
				if err := reader.Close(); err != nil {
					return report, fmt.Errorf("close media %s: %w", photo.ObjectKey, err)
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

func geometry(value orb.Geometry) *GeoJSONGeometry {
	switch value := value.(type) {
	case orb.Point:
		return &GeoJSONGeometry{Type: "Point", Coordinates: []float64{value.X(), value.Y()}}
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
				points = append(points, []float64{point.X(), point.Y()})
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
			points = append(points, []float64{point.X(), point.Y()})
		}
		return &GeoJSONGeometry{Type: "LineString", Coordinates: points}
	default:
		return nil
	}
}
