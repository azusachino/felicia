// Package contract contains backend-neutral repository conformance tests.
package contract

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
)

// Run exercises the behavior that every persistence provider must expose.
func Run(t *testing.T, repo domain.Repository) {
	t.Helper()
	ctx := context.Background()

	journal := &domain.Journal{ID: mustUUID(t), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("create journal: %v", err)
	}

	journey := &domain.Journey{
		ID:        mustUUID(t),
		JournalID: journal.ID,
		Slug:      "contract-journey",
		Title:     "Contract journey",
		Place:     "Tokyo",
		DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		GPSRoute:  orb.MultiLineString{{{139.7, 35.6}, {139.8, 35.7}}},
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("upsert journey: %v", err)
	}
	assertJourney(t, repo, journey)

	memento := &domain.Memento{
		ID:        mustUUID(t),
		JourneyID: journey.ID,
		Kind:      "video",
		Seq:       1,
		KindData:  json.RawMessage(`{"url":"https://example.com/memory"}`),
		State:     domain.MementoDraft,
	}
	if err := repo.UpsertMemento(ctx, memento); err != nil {
		t.Fatalf("upsert draft memento: %v", err)
	}
	fetched, err := repo.GetMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("get draft memento: %v", err)
	}
	if fetched.State != domain.MementoDraft || fetched.Revision != 1 || fetched.Kind != "video" {
		t.Fatalf("unexpected draft memento: %#v", fetched)
	}

	listed, err := repo.ListMementosByJourney(ctx, journey.ID)
	if err != nil {
		t.Fatalf("list mementos: %v", err)
	}
	if len(listed) != 1 || listed[0].ID != memento.ID {
		t.Fatalf("unexpected memento list: %#v", listed)
	}

	occurred := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{
			ID:         memento.ID,
			JourneyID:  journey.ID,
			OccurredAt: occurred,
			OccurredTZ: "Asia/Tokyo",
			Geom:       orb.Point{139.75, 35.69},
			Title:      "Memory video",
			Place:      "Tokyo",
			State:      domain.MementoPublished,
		},
		Fields:           []string{"occurred_at", "occurred_tz", "geom", "title", "place"},
		State:            domain.MementoPublished,
		ExpectedRevision: int64Ptr(1),
	}); err != nil {
		t.Fatalf("publish memento: %v", err)
	}
	fetched, err = repo.GetMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("get published memento: %v", err)
	}
	if fetched.State != domain.MementoPublished || fetched.Revision != 2 || fetched.Title != "Memory video" {
		t.Fatalf("unexpected published memento: %#v", fetched)
	}

	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento:          &domain.Memento{ID: memento.ID, JourneyID: journey.ID, Title: "stale"},
		Fields:           []string{"title"},
		ExpectedRevision: int64Ptr(1),
	}); err == nil {
		t.Fatal("expected stale memento revision to fail")
	}

	photo := &domain.MementoPhoto{
		ID:          mustUUID(t),
		MementoID:   memento.ID,
		ObjectKey:   "memory-video-poster.jpg",
		ContentHash: "hash",
		Seq:         1,
	}
	if err := repo.UpsertPhoto(ctx, photo); err != nil {
		t.Fatalf("upsert photo: %v", err)
	}
	photos, err := repo.ListPhotosByMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("list photos: %v", err)
	}
	if len(photos) != 1 || photos[0].ID != photo.ID {
		t.Fatalf("unexpected photos: %#v", photos)
	}
}

func assertJourney(t *testing.T, repo domain.Repository, expected *domain.Journey) {
	t.Helper()
	ctx := context.Background()
	fetched, err := repo.GetJourney(ctx, expected.ID)
	if err != nil {
		t.Fatalf("get journey: %v", err)
	}
	if fetched.Title != expected.Title || len(fetched.GPSRoute) != 1 || len(fetched.GPSRoute[0]) != 2 {
		t.Fatalf("unexpected journey: %#v", fetched)
	}
	bySlug, err := repo.GetJourneyBySlug(ctx, expected.Slug)
	if err != nil || bySlug.ID != expected.ID {
		t.Fatalf("get journey by slug: %v %#v", err, bySlug)
	}
	journeys, err := repo.ListJourneys(ctx)
	if err != nil || len(journeys) != 1 || journeys[0].ID != expected.ID {
		t.Fatalf("list journeys: %v %#v", err, journeys)
	}
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func int64Ptr(value int64) *int64 { return &value }
