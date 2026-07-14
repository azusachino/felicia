// Package journey contains journey authoring use cases.
package journey

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/apps/core/ports"
)

// Service owns journey writes and hides the persistence implementation from
// the transport layer.
type Service struct {
	store ports.JourneyStore
}

// New creates a journey application service from a narrow store port.
func New(store ports.JourneyStore) *Service {
	return &Service{store: store}
}

// Save persists a journey authoring command. IDs are generated here when the
// API intentionally leaves them empty, keeping identity policy out of HTTP.
func (s *Service) Save(ctx context.Context, journey *domain.Journey) error {
	if journey == nil {
		return errors.New("journey is required")
	}
	if journey.ID == uuid.Nil {
		journey.ID = uuid.Must(uuid.NewV7())
	}
	if journey.JournalID == uuid.Nil {
		return errors.New("journey journal ID is required")
	}
	if journey.Slug == "" {
		return errors.New("journey slug is required")
	}
	if journey.Title == "" || journey.Place == "" {
		return errors.New("journey title and place are required")
	}
	return s.store.UpsertJourney(ctx, journey)
}
