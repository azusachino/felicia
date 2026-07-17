package memento_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/runtime/memento"
)

type fakeStore struct{ patch *domain.ManualMementoPatch }

func (s *fakeStore) GetMemento(context.Context, uuid.UUID) (*domain.Memento, error) {
	return nil, domain.ErrNotFound
}
func (s *fakeStore) GetMementoBySourceIdentity(context.Context, domain.SourceIdentity) (*domain.Memento, error) {
	return nil, domain.ErrNotFound
}
func (s *fakeStore) ListMementosByJourney(context.Context, uuid.UUID) ([]*domain.Memento, error) {
	return nil, nil
}
func (s *fakeStore) UpsertMemento(context.Context, *domain.Memento) error { return nil }
func (s *fakeStore) ApplyManualMementoPatch(_ context.Context, patch *domain.ManualMementoPatch) error {
	s.patch = patch
	return nil
}
func (s *fakeStore) ApplyIngestMementoPatch(context.Context, *domain.IngestMementoPatch) error {
	return nil
}

func TestApplyManualPatchUsesStorePort(t *testing.T) {
	store := &fakeStore{}
	service := memento.New(store)
	patch := &domain.ManualMementoPatch{Memento: &domain.Memento{ID: uuid.Must(uuid.NewV7())}}
	if err := service.ApplyManualPatch(context.Background(), patch); err != nil {
		t.Fatalf("apply patch: %v", err)
	}
	if store.patch != patch {
		t.Fatal("manual patch did not reach store port")
	}
	if err := service.ApplyManualPatch(context.Background(), nil); err == nil {
		t.Fatal("expected nil patch to fail")
	}
}
