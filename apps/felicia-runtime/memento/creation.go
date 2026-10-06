package memento

import (
	"context"
	"errors"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
)

// manualCreationAuthoredFields is the server-derived authorship mask for a
// manually created draft row. The transport layer never accepts a client
// mask on creation: ownership is a server decision, matching the manual
// upsert path's derived fields. seq is included even though its value is
// server-allocated here — as on the old form's manual row, the new row's
// ordering position is authored state that later imports must not overwrite.
var manualCreationAuthoredFields = []string{
	"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "title", "place", "kind_data",
}

// CreateManualMemento persists a prepared manual creation — the transport
// layer has already validated and authored every field, including the
// caller-chosen identity. The authorship mask is derived here, never taken
// from the caller: the prepared payload is cloned so a caller's struct is
// not mutated and every retry normalizes to the same canonical mask.
// Allocation, insertion and stable-retry handling are one atomic operation
// on the store; this seam never reads a maximum, falls back to upsert, or
// reuses the edit patch path.
func (s *Service) CreateManualMemento(ctx context.Context, memento *domain.Memento) (*domain.Memento, error) {
	if memento == nil {
		return nil, errors.New("manual memento creation payload is required")
	}
	creator, ok := s.store.(ports.MementoCreator)
	if !ok {
		return nil, errors.New("memento creation is not supported by this store")
	}
	prepared := *memento
	prepared.AuthoredFields = append([]string(nil), manualCreationAuthoredFields...)
	return creator.CreateMementoWithNextSequence(ctx, &prepared)
}
