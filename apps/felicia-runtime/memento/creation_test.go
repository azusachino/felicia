package memento_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-runtime/memento"
)

// fakeCreatorStore records the prepared creation payload and replays a
// canned result. It embeds the plain patch-only fake so the unsupported
// variant is exactly the same store without the creator capability.
type fakeCreatorStore struct {
	fakeStore
	in  *domain.Memento
	out *domain.Memento
	err error
}

func (s *fakeCreatorStore) CreateMementoWithNextSequence(_ context.Context, memento *domain.Memento) (*domain.Memento, error) {
	s.in = memento
	if s.err != nil {
		return nil, s.err
	}
	return s.out, nil
}

func TestCreateManualMementoDelegatesPreparedPayload(t *testing.T) {
	stored := &domain.Memento{ID: uuid.Must(uuid.NewV7()), Seq: 5, Revision: 1}
	store := &fakeCreatorStore{out: stored}
	service := memento.New(store)
	prepared := &domain.Memento{ID: stored.ID, JourneyID: uuid.Must(uuid.NewV7()), Kind: "ticket", Title: "Fushimi"}
	got, err := service.CreateManualMemento(context.Background(), prepared)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if got != stored {
		t.Fatal("creation did not return the stored row")
	}
	if store.in.ID != prepared.ID {
		t.Fatal("prepared payload did not reach the store unchanged")
	}
}

func TestCreateManualMementoDerivesAuthorshipServerSide(t *testing.T) {
	store := &fakeCreatorStore{out: &domain.Memento{ID: uuid.Must(uuid.NewV7()), Seq: 5}}
	service := memento.New(store)
	prepared := &domain.Memento{ID: uuid.Must(uuid.NewV7()), Kind: "ticket", Title: "Fushimi", AuthoredFields: []string{"kind"}}
	if _, err := service.CreateManualMemento(context.Background(), prepared); err != nil {
		t.Fatalf("create: %v", err)
	}
	// The stored mask is server-derived and identical for every retry; the
	// caller's own struct (and its mask) is never mutated.
	canonical := []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "title", "place", "kind_data"}
	if !slices.Equal(store.in.AuthoredFields, canonical) {
		t.Fatalf("stored mask = %v, want %v", store.in.AuthoredFields, canonical)
	}
	if !slices.Equal(prepared.AuthoredFields, []string{"kind"}) {
		t.Fatalf("caller mask mutated: %v", prepared.AuthoredFields)
	}
	if _, err := service.CreateManualMemento(context.Background(), prepared); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if !slices.Equal(store.in.AuthoredFields, canonical) {
		t.Fatalf("retry mask = %v, want the same canonical mask", store.in.AuthoredFields)
	}
}

func TestCreateManualMementoPropagatesStoreErrors(t *testing.T) {
	store := &fakeCreatorStore{err: domain.ErrWriteConflict}
	service := memento.New(store)
	if _, err := service.CreateManualMemento(context.Background(), &domain.Memento{ID: uuid.Must(uuid.NewV7())}); !errors.Is(err, domain.ErrWriteConflict) {
		t.Fatalf("conflict = %v, want ErrWriteConflict", err)
	}
	if _, err := service.CreateManualMemento(context.Background(), nil); err == nil {
		t.Fatal("expected nil payload to fail")
	}
}

func TestCreateManualMementoFailsWithoutCreatorCapability(t *testing.T) {
	service := memento.New(&fakeStore{})
	if _, err := service.CreateManualMemento(context.Background(), &domain.Memento{ID: uuid.Must(uuid.NewV7())}); err == nil {
		t.Fatal("expected store without creation capability to fail")
	}
}
