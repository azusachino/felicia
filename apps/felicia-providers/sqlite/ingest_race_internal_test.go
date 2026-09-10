package sqlite

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// TestIngestAndAuthoringWriteAreSerialised reproduces the interleaving that
// made journey ingest unsafe. Ingest read a journey, decided ownership from
// that copy, then wrote the whole row back; an authoring write landing in
// between was overwritten by state that was already stale, and it took the
// authored mask with it. The mask is what ADR-0033 relies on to protect every
// later import, so losing it silently downgrades all future ingests.
//
// The window is too narrow for a stress loop to hit, so the test widens it
// through afterIngestRead rather than racing and hoping. With the read and the
// write in one transaction the authoring write cannot land inside the window at
// all: it waits for the connection this provider pools singly, then applies
// afterwards.
func TestIngestAndAuthoringWriteAreSerialised(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()
	ctx := context.Background()

	journal := &domain.Journal{ID: uuid.New(), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	id := uuid.New()
	seed := &domain.Journey{
		ID: id, JournalID: journal.ID, Slug: "serialised", Title: "Imported name", Place: "Kyoto",
		DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 3, 22, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(ctx, seed); err != nil {
		t.Fatalf("seed journey: %v", err)
	}

	readHappened := make(chan struct{})
	original := afterIngestRead
	afterIngestRead = func() {
		close(readHappened)
		// Long enough that an unsynchronised authoring write would certainly
		// land before the ingest writes its stale copy back.
		time.Sleep(100 * time.Millisecond)
	}
	t.Cleanup(func() { afterIngestRead = original })

	ingestDone := make(chan error, 1)
	go func() {
		ingestDone <- repo.ApplyIngestJourneyPatch(ctx, &domain.IngestJourneyPatch{
			Journey: &domain.Journey{
				ID: id, JournalID: journal.ID, Slug: seed.Slug, Title: "Imported name", Place: "Osaka",
				DateStart: seed.DateStart, DateEnd: seed.DateEnd,
			},
			Fields: []string{"title", "place"},
		})
	}()

	<-readHappened
	authored := &domain.Journey{
		ID: id, JournalID: journal.ID, Slug: seed.Slug, Title: "The name I chose", Place: "Kyoto",
		DateStart: seed.DateStart, DateEnd: seed.DateEnd,
		AuthoredFields: []string{"title"},
	}
	if err := repo.UpsertJourney(ctx, authored); err != nil {
		t.Fatalf("authoring write: %v", err)
	}
	if err := <-ingestDone; err != nil {
		t.Fatalf("ingest: %v", err)
	}

	stored, err := repo.GetJourney(ctx, id)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if stored.Title != authored.Title {
		t.Errorf("authored title was overwritten by ingest: got %q, want %q", stored.Title, authored.Title)
	}
	if !slices.Contains(stored.AuthoredFields, "title") {
		t.Errorf("authored mask was dropped, leaving future imports unprotected: %v", stored.AuthoredFields)
	}
}
