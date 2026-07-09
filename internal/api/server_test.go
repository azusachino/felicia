package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/internal/api"
	"github.com/azusachino/felicia/internal/domain"
)

type mockRepository struct {
	journeys     map[uuid.UUID]*domain.Journey
	mementos     map[uuid.UUID]*domain.Memento
	photos       map[uuid.UUID]*domain.MementoPhoto
	translations []*domain.Translation
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		journeys:     make(map[uuid.UUID]*domain.Journey),
		mementos:     make(map[uuid.UUID]*domain.Memento),
		photos:       make(map[uuid.UUID]*domain.MementoPhoto),
		translations: []*domain.Translation{},
	}
}

func (m *mockRepository) GetJournal(_ context.Context, id uuid.UUID) (*domain.Journal, error) {
	return &domain.Journal{ID: id, CreatedAt: time.Now()}, nil
}

func (m *mockRepository) CreateJournal(_ context.Context, _ *domain.Journal) error {
	return nil
}

func (m *mockRepository) GetJourney(_ context.Context, id uuid.UUID) (*domain.Journey, error) {
	j, ok := m.journeys[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return j, nil
}

func (m *mockRepository) GetJourneyBySlug(_ context.Context, slug string) (*domain.Journey, error) {
	for _, j := range m.journeys {
		if j.Slug == slug {
			return j, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (m *mockRepository) ListJourneys(_ context.Context) ([]*domain.Journey, error) {
	var list []*domain.Journey
	for _, j := range m.journeys {
		list = append(list, j)
	}
	return list, nil
}

func (m *mockRepository) UpsertJourney(_ context.Context, journey *domain.Journey) error {
	m.journeys[journey.ID] = journey
	return nil
}

func (m *mockRepository) GetMemento(_ context.Context, id uuid.UUID) (*domain.Memento, error) {
	mem, ok := m.mementos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return mem, nil
}

func (m *mockRepository) ListMementosByJourney(_ context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	var list []*domain.Memento
	for _, mem := range m.mementos {
		if mem.JourneyID == journeyID {
			list = append(list, mem)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertMemento(_ context.Context, memento *domain.Memento) error {
	m.mementos[memento.ID] = memento
	return nil
}

func (m *mockRepository) GetPhoto(_ context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	ph, ok := m.photos[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return ph, nil
}

func (m *mockRepository) ListPhotosByMemento(_ context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	var list []*domain.MementoPhoto
	for _, ph := range m.photos {
		if ph.MementoID == mementoID {
			list = append(list, ph)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertPhoto(_ context.Context, photo *domain.MementoPhoto) error {
	m.photos[photo.ID] = photo
	return nil
}

func (m *mockRepository) ListTranslations(_ context.Context, ownerType string, ownerID uuid.UUID) ([]*domain.Translation, error) {
	var list []*domain.Translation
	for _, t := range m.translations {
		if t.OwnerType == ownerType && t.OwnerID == ownerID {
			list = append(list, t)
		}
	}
	return list, nil
}

func (m *mockRepository) UpsertTranslation(_ context.Context, translation *domain.Translation) error {
	for i, t := range m.translations {
		if t.OwnerType == translation.OwnerType && t.OwnerID == translation.OwnerID && t.Lang == translation.Lang && t.Field == translation.Field {
			m.translations[i] = translation
			return nil
		}
	}
	m.translations = append(m.translations, translation)
	return nil
}

func TestServerGetTemplates(t *testing.T) {
	reg, err := domain.LoadRegistry(os.DirFS("../../kinds"))
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}

	repo := newMockRepository()
	srv := api.NewServer(repo, reg)
	handler := srv.Handler()

	req := httptest.NewRequest("GET", "/api/admin/templates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var res map[string]domain.Template
	if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if _, ok := res["transit"]; !ok {
		t.Error("expected transit template to be returned")
	}
}

func TestServerUpsertMementoValidation(t *testing.T) {
	reg, err := domain.LoadRegistry(os.DirFS("../../kinds"))
	if err != nil {
		t.Fatalf("failed to load kinds templates: %v", err)
	}

	repo := newMockRepository()
	srv := api.NewServer(repo, reg)
	handler := srv.Handler()

	// 1. Submit invalid memento (missing required transit fields 'operator', 'from', 'to')
	payload := map[string]any{
		"id":          uuid.New(),
		"journey_id":  uuid.New(),
		"kind":        "transit",
		"seq":         1,
		"occurred_at": time.Now().Format(time.RFC3339),
		"occurred_tz": "Asia/Tokyo",
		"title":       "Invalid Transit",
		"place":       "Somewhere",
		"kind_data":   map[string]any{"line": "Yamanote Line"}, // missing 'operator', 'from', 'to'
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/api/admin/mementos", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request, got %d", w.Code)
	}

	var res map[string]any
	_ = json.NewDecoder(w.Body).Decode(&res)
	if res["error"] != "validation failed" {
		t.Errorf("expected validation failed error, got: %v", res["error"])
	}
	issues, ok := res["issues"].([]any)
	if !ok || len(issues) == 0 {
		t.Error("expected validation issues in response")
	}
}
