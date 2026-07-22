package importer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

// lifecycleStore is a minimal PackageStore that enforces the memento
// transition guard the real providers apply, so this test can prove the
// importer's ingest-then-manual sequence surfaces an illegal downgrade
// (docs/contracts/memento-lifecycle.md §7) without depending on a concrete
// provider (which would invert the runtime→providers layering). The embedded
// nil domain.Repository satisfies the rest of PackageStore's method set at
// compile time; ApplyPackage never calls those methods.
type lifecycleStore struct {
	domain.Repository
	states map[uuid.UUID]domain.MementoState
}

func newLifecycleStore() *lifecycleStore {
	return &lifecycleStore{states: map[uuid.UUID]domain.MementoState{}}
}

func (s *lifecycleStore) EnsureJournal(context.Context, *domain.Journal) error    { return nil }
func (s *lifecycleStore) UpsertJourney(context.Context, *domain.Journey) error    { return nil }
func (s *lifecycleStore) UpsertPhoto(context.Context, *domain.MementoPhoto) error { return nil }

func (s *lifecycleStore) ApplyIngestMementoPatch(_ context.Context, patch *domain.IngestMementoPatch) error {
	if _, ok := s.states[patch.Memento.ID]; !ok {
		state := patch.Memento.State
		if state == "" {
			state = domain.MementoCandidateState
		}
		s.states[patch.Memento.ID] = state // ingest sets state only on creation
	}
	return nil
}

func (s *lifecycleStore) ApplyManualMementoPatch(_ context.Context, patch *domain.ManualMementoPatch) error {
	from, exists := s.states[patch.Memento.ID]
	if exists && patch.State != "" && !domain.CanTransitionMementoState(from, patch.State) {
		return &domain.InvalidTransitionError{From: from, To: patch.State}
	}
	if patch.State != "" {
		s.states[patch.Memento.ID] = patch.State
	}
	return nil
}

// A published memento re-imported at the same state is a no-op (same-state is
// legal); a package that would move it backward (published→draft) fails loudly
// rather than silently unpublishing it (docs/contracts/memento-lifecycle.md §6, §7).
func TestReimportSameStateSucceedsAndDowngradeFails(t *testing.T) {
	ctx := context.Background()
	store := newLifecycleStore()

	journalID := uuid.Must(uuid.NewV7())
	journeyID := uuid.Must(uuid.NewV7())
	mementoID := uuid.Must(uuid.NewV7())
	source := domain.SourceIdentity{System: "local", ExternalID: "m-1"}

	doc := func(state domain.MementoState) *PackageDocument {
		return &PackageDocument{
			Journey: &domain.Journey{
				ID: journeyID, JournalID: journalID, Slug: "reimport", Title: "Reimport",
				Place: "Tokyo", DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
				DateEnd: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			},
			Mementos: []*domain.Memento{{
				ID: mementoID, JourneyID: journeyID, Kind: "goods", Seq: 1,
				OccurredAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
				Geom: orb.Point{139.7, 35.6}, Title: "Souvenir", Place: "Tokyo",
				KindData: []byte(`{"name":"thing"}`), AuthoredFields: []string{"title"},
				SourceIdentity: &source, State: state,
			}},
		}
	}

	if _, err := ApplyPackage(ctx, doc(domain.MementoPublished), store); err != nil {
		t.Fatalf("first import: %v", err)
	}
	if _, err := ApplyPackage(ctx, doc(domain.MementoPublished), store); err != nil {
		t.Fatalf("re-import same state: %v", err)
	}
	var invalid *domain.InvalidTransitionError
	if _, err := ApplyPackage(ctx, doc(domain.MementoDraft), store); !errors.As(err, &invalid) {
		t.Fatalf("downgrade re-import error = %v, want InvalidTransitionError", err)
	}
	if store.states[mementoID] != domain.MementoPublished {
		t.Fatalf("state after rejected downgrade = %q, want published", store.states[mementoID])
	}
}
