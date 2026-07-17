package publication

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
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
	ID         string           `json:"id"`
	JourneyID  string           `json:"journey_id"`
	Kind       string           `json:"kind"`
	Seq        int              `json:"seq"`
	OccurredAt string           `json:"occurred_at"`
	OccurredTZ string           `json:"occurred_tz"`
	Geom       *GeoJSONGeometry `json:"geom,omitempty"`
	Title      string           `json:"title"`
	Place      string           `json:"place"`
	KindData   json.RawMessage  `json:"kind_data,omitempty"`
	SourceRef  *string          `json:"source_ref,omitempty"`
	Photos     []StaticPhoto    `json:"photos,omitempty"`
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
	sort.Slice(journeys, func(i, j int) bool {
		if journeys[i].DateStart.Equal(journeys[j].DateStart) {
			if journeys[i].Slug == journeys[j].Slug {
				return journeys[i].ID.String() < journeys[j].ID.String()
			}
			return journeys[i].Slug < journeys[j].Slug
		}
		return journeys[i].DateStart.Before(journeys[j].DateStart)
	})

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
		published := make([]*domain.Memento, 0, len(mementos))
		for _, memento := range mementos {
			if memento.State == domain.MementoPublished {
				published = append(published, memento)
			}
		}
		sort.SliceStable(published, func(i, j int) bool {
			if published[i].Seq != published[j].Seq {
				return published[i].Seq < published[j].Seq
			}
			if published[i].OccurredAt.Equal(published[j].OccurredAt) {
				return published[i].ID.String() < published[j].ID.String()
			}
			return published[i].OccurredAt.Before(published[j].OccurredAt)
		})
		index = append(index, NewJourneyListItem(journey, published))
		report.Journeys++
		report.Mementos += len(published)

		journeyProjection := StaticJourney{
			ID: journey.ID.String(), JournalID: journey.JournalID.String(), Slug: journey.Slug,
			SourceRef: journey.SourceRef, Title: journey.Title, Place: journey.Place,
			Country: journey.Country, Region: journey.Region,
			DateStart: journey.DateStart.Format("2006-01-02"), DateEnd: journey.DateEnd.Format("2006-01-02"),
			GPSRoute: geometry(journey.GPSRoute), AuthoredFields: journey.AuthoredFields,
		}
		if err := output.WriteJSON("api/v1/journeys/"+journey.ID.String()+".json", journeyProjection); err != nil {
			return report, fmt.Errorf("write journey %s: %w", journey.ID, err)
		}

		mementoProjection := make([]StaticMemento, 0, len(published))
		for _, memento := range published {
			photos, err := read.ListPhotosByMemento(ctx, memento.ID)
			if err != nil {
				return report, fmt.Errorf("list photos for %s: %w", memento.ID, err)
			}
			sort.SliceStable(photos, func(i, j int) bool {
				if photos[i].Seq != photos[j].Seq {
					return photos[i].Seq < photos[j].Seq
				}
				return photos[i].ID.String() < photos[j].ID.String()
			})
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
				photoProjection = append(photoProjection, StaticPhoto{ID: photo.ID.String(), MementoID: photo.MementoID.String(), ObjectKey: photo.ObjectKey, ContentHash: photo.ContentHash, Caption: photo.Caption, Seq: photo.Seq, SourceRef: photo.SourceRef})
			}
			mementoProjection = append(mementoProjection, StaticMemento{ID: memento.ID.String(), JourneyID: memento.JourneyID.String(), Kind: memento.Kind, Seq: memento.Seq, OccurredAt: memento.OccurredAt.Format(time.RFC3339), OccurredTZ: memento.OccurredTZ, Geom: geometry(memento.Geom), Title: memento.Title, Place: memento.Place, KindData: memento.KindData, SourceRef: memento.SourceRef, Photos: photoProjection})
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
		points := make([][]float64, 0, len(value))
		for _, point := range value {
			points = append(points, []float64{point.X(), point.Y()})
		}
		return &GeoJSONGeometry{Type: "LineString", Coordinates: points}
	default:
		return nil
	}
}
