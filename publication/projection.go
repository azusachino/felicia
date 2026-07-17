// Package publication defines Felicia's presentation-agnostic public contract.
package publication

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

// JourneyListItem is the compact landing-page projection for one journey.
type JourneyListItem struct {
	ID                 uuid.UUID           `json:"id"`
	Slug               string              `json:"slug"`
	Title              string              `json:"title"`
	MementoCount       int                 `json:"memento_count"`
	RepresentativeDots []RepresentativeDot `json:"representative_dots"`
}

// RepresentativeDot is a spatial place anchor for a journey card or map.
type RepresentativeDot struct {
	Coord []float64 `json:"coord"`
	Label string    `json:"label"`
}

// NewJourneyListItem builds the shared compact projection from an already
// loaded journey and its ordered mementos.
func NewJourneyListItem(journey *domain.Journey, mementos []*domain.Memento) JourneyListItem {
	return JourneyListItem{
		ID:                 journey.ID,
		Slug:               journey.Slug,
		Title:              journey.Title,
		MementoCount:       len(mementos),
		RepresentativeDots: representativeDots(mementos),
	}
}

func representativeDots(mementos []*domain.Memento) []RepresentativeDot {
	var dots []RepresentativeDot
	seenPlaces := make(map[string]bool)
	for _, memento := range mementos {
		if len(dots) >= 3 {
			break
		}
		if seenPlaces[memento.Place] {
			continue
		}

		coord := representativeCoord(memento.Geom)
		if coord == nil {
			continue
		}
		seenPlaces[memento.Place] = true
		dots = append(dots, RepresentativeDot{Coord: coord, Label: memento.Place})
	}
	return dots
}

func representativeCoord(geom orb.Geometry) []float64 {
	switch g := geom.(type) {
	case orb.Point:
		return []float64{g.X(), g.Y()}
	case orb.LineString:
		if len(g) > 0 {
			return []float64{g[0].X(), g[0].Y()}
		}
	}
	return nil
}

// ReadModel is the provider-neutral read surface required by a compiler.
// Implementations may be backed by SQLite, PostgreSQL, or another store.
type ReadModel interface {
	ListJourneys(context.Context) ([]*domain.Journey, error)
	ListMementosByJourney(context.Context, uuid.UUID) ([]*domain.Memento, error)
	ListPhotosByMemento(context.Context, uuid.UUID) ([]*domain.MementoPhoto, error)
}

// MediaSource opens a source object by its canonical object key.
type MediaSource interface {
	Open(context.Context, string) (io.ReadCloser, error)
}

// ArtifactWriter receives deterministic public JSON and media files.
type ArtifactWriter interface {
	WriteJSON(path string, value any) error
	WriteMedia(path string, source io.Reader) error
}

// PublicationInput controls one static compilation.
type PublicationInput struct {
	JourneyIDs []uuid.UUID
}

// BuildReport describes the generated public artifact.
type BuildReport struct {
	Journeys int
	Mementos int
	Media    int
}

// Compiler is the shared publication boundary used by CLI and server modes.
type Compiler interface {
	Compile(context.Context, PublicationInput, ReadModel, MediaSource, ArtifactWriter) (BuildReport, error)
}
