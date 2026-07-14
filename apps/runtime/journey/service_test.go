package journey_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/apps/runtime/journey"
)

type fakeStore struct{ saved *domain.Journey }

func (s *fakeStore) GetJourney(context.Context, uuid.UUID) (*domain.Journey, error) {
	return nil, domain.ErrNotFound
}
func (s *fakeStore) GetJourneyBySlug(context.Context, string) (*domain.Journey, error) {
	return nil, domain.ErrNotFound
}
func (s *fakeStore) ListJourneys(context.Context) ([]*domain.Journey, error) { return nil, nil }
func (s *fakeStore) UpsertJourney(_ context.Context, value *domain.Journey) error {
	s.saved = value
	return nil
}

func TestSaveGeneratesIDAndRejectsIncompleteJourney(t *testing.T) {
	store := &fakeStore{}
	service := journey.New(store)
	value := &domain.Journey{JournalID: uuid.Must(uuid.NewV7()), Slug: "tokyo", Title: "Tokyo", Place: "Japan"}
	if err := service.Save(context.Background(), value); err != nil {
		t.Fatalf("save: %v", err)
	}
	if value.ID == uuid.Nil || store.saved != value {
		t.Fatalf("journey was not saved with generated ID: %#v", value)
	}
	if err := service.Save(context.Background(), &domain.Journey{}); err == nil {
		t.Fatal("expected incomplete journey to fail")
	}
}
