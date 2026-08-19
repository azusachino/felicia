package importer

import (
	"context"
	"errors"
	"slices"
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
func (s *lifecycleStore) UpsertPhoto(context.Context, *domain.MementoPhoto) error { return nil }

func (s *lifecycleStore) ApplyIngestJourneyPatch(context.Context, *domain.IngestJourneyPatch) error {
	return nil
}

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

// authorshipStore records what ApplyPackage asks the persistence layer to do,
// applying the same authored-field rules the providers apply. It exists so this
// test proves ApplyPackage routes journey writes through the ingest seam with a
// correct field mask, without depending on a concrete provider (which would
// invert the runtime→providers layering); the cross-provider assertions live in
// providers/contract.
type authorshipStore struct {
	domain.Repository
	journey  *domain.Journey
	mementos map[uuid.UUID]*domain.Memento
	// journeyUpserts counts authoring writes, which an import must never make.
	journeyUpserts int
}

func newAuthorshipStore(journey *domain.Journey) *authorshipStore {
	return &authorshipStore{journey: journey, mementos: map[uuid.UUID]*domain.Memento{}}
}

func (s *authorshipStore) EnsureJournal(context.Context, *domain.Journal) error    { return nil }
func (s *authorshipStore) UpsertPhoto(context.Context, *domain.MementoPhoto) error { return nil }

func (s *authorshipStore) UpsertJourney(_ context.Context, journey *domain.Journey) error {
	s.journeyUpserts++
	s.journey = journey
	return nil
}

func (s *authorshipStore) ApplyIngestJourneyPatch(_ context.Context, patch *domain.IngestJourneyPatch) error {
	if s.journey == nil || s.journey.ID != patch.Journey.ID {
		s.journey = &domain.Journey{ID: patch.Journey.ID, JournalID: patch.Journey.JournalID}
	}
	domain.MergeIngestJourney(s.journey, patch)
	return nil
}

func (s *authorshipStore) ApplyIngestMementoPatch(_ context.Context, patch *domain.IngestMementoPatch) error {
	current, ok := s.mementos[patch.Memento.ID]
	if !ok {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID, State: patch.Memento.State}
		s.mementos[current.ID] = current
	}
	for _, field := range domain.IngestableFields(patch.Fields, current.AuthoredFields) {
		switch field {
		case "title":
			current.Title = patch.Memento.Title
		case "place":
			current.Place = patch.Memento.Place
		case "kind_data":
			current.KindData = patch.Memento.KindData
		}
	}
	return nil
}

func (s *authorshipStore) ApplyManualMementoPatch(_ context.Context, patch *domain.ManualMementoPatch) error {
	current, ok := s.mementos[patch.Memento.ID]
	if !ok {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID}
		s.mementos[current.ID] = current
	}
	for _, field := range patch.Fields {
		switch field {
		case "title":
			current.Title = patch.Memento.Title
		case "place":
			current.Place = patch.Memento.Place
		case "kind_data":
			current.KindData = patch.Memento.KindData
		}
		if !slices.Contains(current.AuthoredFields, field) {
			current.AuthoredFields = append(current.AuthoredFields, field)
		}
	}
	return nil
}

// ApplyPackage must never take the authoring write path for a journey, and the
// ingest mask it supplies must leave an authored journey field and the journey's
// authored mask untouched (AGENTS.md / ADR-0022: "never overwrites authored
// fields — re-import is always safe").
func TestApplyPackageDoesNotClobberAuthoredJourneyFields(t *testing.T) {
	ctx := context.Background()
	journalID := uuid.Must(uuid.NewV7())
	journeyID := uuid.Must(uuid.NewV7())
	mementoID := uuid.Must(uuid.NewV7())
	source := domain.SourceIdentity{System: "local", ExternalID: "m-1"}

	// The stored journey: title authored by hand, place still source-owned.
	stored := &domain.Journey{
		ID: journeyID, JournalID: journalID, Slug: "authored-slug", Title: "Human title",
		Place: "Machine place", DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
		GPSRoute:       orb.MultiLineString{{{1, 2}, {3, 4}}},
		AuthoredFields: []string{"title", "gps_route"},
	}
	store := newAuthorshipStore(stored)

	doc := &PackageDocument{
		Journey: &domain.Journey{
			ID: journeyID, JournalID: journalID, Slug: "package-slug", Title: "Package title",
			Place: "Package place", DateStart: time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
			DateEnd:  time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
			GPSRoute: orb.MultiLineString{},
		},
		Mementos: []*domain.Memento{{
			ID: mementoID, JourneyID: journeyID, Kind: "goods", Seq: 1,
			OccurredAt: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
			Geom: orb.Point{139.7, 35.6}, Title: "Package memento", Place: "Package place",
			KindData: []byte(`{"name":"thing"}`), SourceIdentity: &source,
			State: domain.MementoCandidateState,
		}},
	}

	if _, err := ApplyPackage(ctx, doc, store); err != nil {
		t.Fatalf("first import: %v", err)
	}
	// Author the memento title, then re-import the same package.
	if err := store.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{ID: mementoID, JourneyID: journeyID, Title: "Human memento"},
		Fields:  []string{"title"},
	}); err != nil {
		t.Fatalf("author memento: %v", err)
	}
	if _, err := ApplyPackage(ctx, doc, store); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	if store.journeyUpserts != 0 {
		t.Fatalf("import took the authoring write path %d time(s); it must use the ingest seam", store.journeyUpserts)
	}
	if store.journey.Title != "Human title" {
		t.Fatalf("re-import overwrote the authored journey title: %q", store.journey.Title)
	}
	if len(store.journey.GPSRoute) != 1 {
		t.Fatalf("route-less package blanked the authored gps_route: %#v", store.journey.GPSRoute)
	}
	if len(store.journey.AuthoredFields) != 2 {
		t.Fatalf("re-import reset the journey authored mask: %#v", store.journey.AuthoredFields)
	}
	if store.journey.Place != "Package place" {
		t.Fatalf("re-import failed to update the unauthored journey place: %q", store.journey.Place)
	}
	if got := store.mementos[mementoID].Title; got != "Human memento" {
		t.Fatalf("re-import overwrote the authored memento title: %q", got)
	}
	if got := store.mementos[mementoID].Place; got != "Package place" {
		t.Fatalf("re-import failed to update the unauthored memento place: %q", got)
	}
}
