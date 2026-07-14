package sqlite_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/apps/providers/sqlite"
)

func TestRepositoryJourneyWorkflow(t *testing.T) {
	repo, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()

	journal := &domain.Journal{ID: mustUUID(t), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	journey := &domain.Journey{
		ID: mustUUID(t), JournalID: journal.ID, Slug: "sqlite-journey",
		Title: "SQLite journey", Place: "Tokyo",
		DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
		GPSRoute:  orb.MultiLineString{{{139.7, 35.6}, {139.8, 35.7}}},
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("upsert journey: %v", err)
	}

	memento := &domain.Memento{ID: mustUUID(t), JourneyID: journey.ID, Kind: "live", Seq: 1, State: domain.MementoDraft, KindData: json.RawMessage(`{"artist":"羊文学"}`)}
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: memento, Fields: []string{"kind", "seq", "kind_data"}, State: domain.MementoDraft}); err != nil {
		t.Fatalf("save incomplete draft: %v", err)
	}
	fetched, err := repo.GetMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("get draft: %v", err)
	}
	if fetched.State != domain.MementoDraft || fetched.Revision != 1 || fetched.OccurredAt != (time.Time{}) || fetched.Geom != nil {
		t.Fatalf("unexpected draft: %#v", fetched)
	}

	occurred := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	complete := &domain.Memento{ID: memento.ID, JourneyID: journey.ID, Kind: "live", Seq: 1, OccurredAt: occurred, OccurredTZ: "Asia/Tokyo", Geom: orb.Point{139.75, 35.69}, Title: "Live show", Place: "Tokyo", KindData: json.RawMessage(`{"artist":"羊文学"}`), State: domain.MementoPublished}
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: complete, Fields: []string{"occurred_at", "occurred_tz", "geom", "title", "place", "kind_data"}, State: domain.MementoPublished, ExpectedRevision: int64Ptr(1)}); err != nil {
		t.Fatalf("publish memento: %v", err)
	}
	fetched, err = repo.GetMemento(ctx, memento.ID)
	if err != nil || fetched.State != domain.MementoPublished || fetched.Revision != 2 {
		t.Fatalf("unexpected published memento: %v %#v", err, fetched)
	}

	photo := &domain.MementoPhoto{ID: mustUUID(t), MementoID: memento.ID, ObjectKey: "photo.jpg", ContentHash: "hash"}
	if err := repo.UpsertPhoto(ctx, photo); err != nil {
		t.Fatalf("upsert photo: %v", err)
	}
	translation := &domain.Translation{ID: mustUUID(t), OwnerType: "memento", OwnerID: memento.ID, Lang: "en", Field: "title", Value: "Live show", Provenance: "authored"}
	if err := repo.UpsertTranslation(ctx, translation); err != nil {
		t.Fatalf("upsert translation: %v", err)
	}
	if photos, err := repo.ListPhotosByMemento(ctx, memento.ID); err != nil || len(photos) != 1 {
		t.Fatalf("list photos: %v (%d)", err, len(photos))
	}
	if translations, err := repo.ListTranslations(ctx, "memento", memento.ID); err != nil || len(translations) != 1 {
		t.Fatalf("list translations: %v (%d)", err, len(translations))
	}

	run := &domain.ImportRun{SourceSystem: "sqlite-test", StartedAt: time.Now().UTC(), Status: domain.ImportRunRunning}
	if err := repo.CreateImportRun(ctx, run); err != nil {
		t.Fatalf("create import run: %v", err)
	}
	observation := &domain.SourceObservation{RunID: run.ID, Source: domain.SourceIdentity{System: "sqlite-test", ExternalID: "source-1"}, Kind: domain.ObservationMemento, ObservedAt: time.Now().UTC(), Confidence: 1, Payload: json.RawMessage(`{"memento_id":"test"}`)}
	if err := repo.RecordSourceObservation(ctx, observation); err != nil {
		t.Fatalf("record observation: %v", err)
	}
	if err := repo.MarkMissingSourceObservations(ctx, mustUUID(t), "sqlite-test", nil); err != nil {
		t.Fatalf("mark missing observations: %v", err)
	}
	if err := repo.FinishImportRun(ctx, run.ID, domain.ImportRunSucceeded, time.Now().UTC(), nil); err != nil {
		t.Fatalf("finish import run: %v", err)
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
