// Package publication builds the presentation-agnostic public projections
// shared by the live API and static compiler.
package publication

import (
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/internal/domain"
)

// JourneyListItem is the compact landing-page projection for one journey.
type JourneyListItem struct {
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
