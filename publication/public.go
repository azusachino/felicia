package publication

import (
	"sort"
	"time"

	"github.com/azusachino/felicia/core/domain"
)

// SortJourneys orders journeys into the stable public index order
// (date_start, slug, id) shared by the live API and the static compiler.
func SortJourneys(journeys []*domain.Journey) {
	sort.Slice(journeys, func(i, j int) bool {
		if journeys[i].DateStart.Equal(journeys[j].DateStart) {
			if journeys[i].Slug == journeys[j].Slug {
				return journeys[i].ID.String() < journeys[j].ID.String()
			}
			return journeys[i].Slug < journeys[j].Slug
		}
		return journeys[i].DateStart.Before(journeys[j].DateStart)
	})
}

// PublishedMementos returns only mementos in the published state, sorted into
// the stable public display order (seq, occurred_at, id). It is the single
// publish gate shared by the live public API and the static compiler: a
// journey whose result is empty has no public projection at all.
func PublishedMementos(mementos []*domain.Memento) []*domain.Memento {
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
	return published
}

// NewStaticJourney projects a journey into the public detail shape.
func NewStaticJourney(journey *domain.Journey) StaticJourney {
	return StaticJourney{
		ID:             journey.ID.String(),
		JournalID:      journey.JournalID.String(),
		Slug:           journey.Slug,
		SourceRef:      journey.SourceRef,
		Title:          journey.Title,
		Place:          journey.Place,
		Country:        journey.Country,
		Region:         journey.Region,
		DateStart:      journey.DateStart.Format("2006-01-02"),
		DateEnd:        journey.DateEnd.Format("2006-01-02"),
		GPSRoute:       geometry(journey.GPSRoute),
		AuthoredFields: journey.AuthoredFields,
	}
}

// NewStaticMemento projects a memento and its already projected photos into
// the public shape. Callers are responsible for passing published mementos
// only (see PublishedMementos).
func NewStaticMemento(memento *domain.Memento, photos []StaticPhoto) StaticMemento {
	return StaticMemento{
		ID:            memento.ID.String(),
		JourneyID:     memento.JourneyID.String(),
		Kind:          memento.Kind,
		Seq:           memento.Seq,
		OccurredAt:    memento.OccurredAt.Format(time.RFC3339),
		OccurredTZ:    memento.OccurredTZ,
		Geom:          geometry(memento.Geom),
		Title:         memento.Title,
		Place:         memento.Place,
		Vendor:        memento.Vendor,
		Essay:         memento.Essay,
		PriceAmount:   memento.PriceAmount,
		PriceCurrency: memento.PriceCurrency,
		KindData:      memento.KindData,
		SourceRef:     memento.SourceRef,
		Photos:        photos,
	}
}

// NewStaticPhoto projects photo metadata into the public shape.
func NewStaticPhoto(photo *domain.MementoPhoto) StaticPhoto {
	var takenAt *string
	if photo.TakenAt != nil {
		formatted := photo.TakenAt.Format(time.RFC3339)
		takenAt = &formatted
	}
	return StaticPhoto{
		ID:          photo.ID.String(),
		MementoID:   photo.MementoID.String(),
		ObjectKey:   photo.ObjectKey,
		ContentHash: photo.ContentHash,
		Caption:     photo.Caption,
		Seq:         photo.Seq,
		TakenAt:     takenAt,
		SourceRef:   photo.SourceRef,
	}
}

// SortPhotos orders photo metadata into the stable public order (seq, id).
func SortPhotos(photos []*domain.MementoPhoto) {
	sort.SliceStable(photos, func(i, j int) bool {
		if photos[i].Seq != photos[j].Seq {
			return photos[i].Seq < photos[j].Seq
		}
		return photos[i].ID.String() < photos[j].ID.String()
	})
}
