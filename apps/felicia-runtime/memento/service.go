// Package memento contains memento authoring and ingestion use cases.
package memento

import (
	"context"
	"errors"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
)

// Service owns memento write boundaries and hides the persistence
// implementation from the HTTP transport.
type Service struct {
	store ports.MementoStore
}

// New creates a memento application service from a narrow store port.
func New(store ports.MementoStore) *Service {
	return &Service{store: store}
}

// ApplyManualPatch persists an authoring operation and preserves the store's
// optimistic-concurrency error for transport mapping.
func (s *Service) ApplyManualPatch(ctx context.Context, patch *domain.ManualMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("manual memento patch is required")
	}
	return s.store.ApplyManualMementoPatch(ctx, patch)
}

// ApplyIngestPatch persists source-owned fields through the same runtime seam.
func (s *Service) ApplyIngestPatch(ctx context.Context, patch *domain.IngestMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("ingest memento patch is required")
	}
	return s.store.ApplyIngestMementoPatch(ctx, patch)
}
