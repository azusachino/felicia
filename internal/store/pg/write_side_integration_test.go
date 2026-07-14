package pg_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/apps/runtime/importer"
	"github.com/azusachino/felicia/internal/store/pg"
)

func TestPgWriteSideIntegration(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN environment variable not set, skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
	_, err = pool.Exec(ctx, "TRUNCATE TABLE tb_journal, tb_import_runs CASCADE")
	if err != nil {
		t.Fatalf("failed to clean integration tables: %v", err)
	}

	repo := pg.NewRepository(pool)
	journal := &domain.Journal{ID: mustUUIDv7(t), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	journey := &domain.Journey{
		ID:             mustUUIDv7(t),
		JournalID:      journal.ID,
		Slug:           "write-side-integration",
		Title:          "Write-side integration",
		Place:          "Tokyo",
		DateStart:      time.Now().UTC().Truncate(24 * time.Hour),
		DateEnd:        time.Now().UTC().Add(24 * time.Hour).Truncate(24 * time.Hour),
		AuthoredFields: []string{},
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("upsert journey: %v", err)
	}

	authored := &domain.Memento{
		ID:             mustUUIDv7(t),
		JourneyID:      journey.ID,
		Kind:           "ticket",
		Seq:            1,
		OccurredAt:     time.Now().UTC().Truncate(time.Microsecond),
		OccurredTZ:     "Asia/Tokyo",
		Geom:           orb.Point{139.7671, 35.6812},
		Title:          "Human title",
		Place:          "Tokyo Station",
		KindData:       json.RawMessage(`{"operator":"JR East"}`),
		AuthoredFields: []string{"title"},
		State:          domain.MementoAuthored,
	}
	if err := repo.UpsertMemento(ctx, authored); err != nil {
		t.Fatalf("create authored memento: %v", err)
	}

	identity := domain.SourceIdentity{System: "integration", ExternalID: "ticket-1"}
	ingest := *authored
	ingest.Title = "Machine title"
	ingest.Place = "Imported station"
	ingest.SourceIdentity = &identity
	ingest.State = domain.MementoCandidateState
	if err := repo.ApplyIngestMementoPatch(ctx, &domain.IngestMementoPatch{
		Memento: &ingest,
		Fields:  []string{"title", "place", "kind_data"},
	}); err != nil {
		t.Fatalf("first ingest patch: %v", err)
	}
	if err := repo.ApplyIngestMementoPatch(ctx, &domain.IngestMementoPatch{
		Memento: &ingest,
		Fields:  []string{"title", "place", "kind_data"},
	}); err != nil {
		t.Fatalf("second ingest patch: %v", err)
	}

	fetched, err := repo.GetMementoBySourceIdentity(ctx, identity)
	if err != nil {
		t.Fatalf("get memento by source identity: %v", err)
	}
	if fetched.Title != "Human title" {
		t.Fatalf("ingest clobbered authored title: %q", fetched.Title)
	}
	if fetched.SourceIdentity == nil || *fetched.SourceIdentity != identity {
		t.Fatalf("source identity was not persisted: %#v", fetched.SourceIdentity)
	}
	var mementoCount int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM tb_mementos WHERE source_system = $1 AND source_external_id = $2", identity.System, identity.ExternalID).Scan(&mementoCount); err != nil {
		t.Fatalf("count source mementos: %v", err)
	}
	if mementoCount != 1 {
		t.Fatalf("expected one source memento after repeated ingest, got %d", mementoCount)
	}

	store := repo.(domain.ObservationStore)
	observedAt := time.Now().UTC().Truncate(time.Microsecond)
	firstRun, err := importer.PersistObservations(ctx, store, "integration", []domain.Observation{{
		Kind: domain.ObservationMemento, Source: domain.SourceIdentity{System: "integration", ExternalID: "observation-1"},
		ObservedAt: observedAt, Confidence: 1, Payload: map[string]any{"memento_id": fetched.ID.String()},
	}})
	if err != nil {
		t.Fatalf("first observation run: %v", err)
	}
	secondRun, err := importer.PersistObservations(ctx, store, "integration", nil)
	if err != nil {
		t.Fatalf("second observation run: %v", err)
	}
	var orphanedAt *time.Time
	if err := pool.QueryRow(ctx, "SELECT orphaned_at FROM tb_source_observations WHERE run_id = $1 AND source_external_id = $2", firstRun.ID, "observation-1").Scan(&orphanedAt); err != nil {
		t.Fatalf("load orphaned observation: %v", err)
	}
	if orphanedAt == nil {
		t.Fatal("missing observation was not marked orphaned")
	}
	if secondRun.Status != domain.ImportRunSucceeded {
		t.Fatalf("expected empty follow-up run to succeed, got %s", secondRun.Status)
	}
	fetched, err = repo.GetMementoBySourceIdentity(ctx, identity)
	if err != nil || fetched.Title != "Human title" {
		t.Fatalf("observation reconciliation changed authored memento: %v", err)
	}

	_, err = importer.PersistObservations(ctx, store, "integration", []domain.Observation{{
		Kind: domain.ObservationMemento, Source: domain.SourceIdentity{System: "other", ExternalID: "wrong-source"}, Payload: map[string]any{},
	}})
	if err == nil {
		t.Fatal("expected mixed-source observation run to fail")
	}
	var status string
	if err := pool.QueryRow(ctx, "SELECT status FROM tb_import_runs WHERE source_system = $1 ORDER BY started_at DESC LIMIT 1", "integration").Scan(&status); err != nil {
		t.Fatalf("load failed import status: %v", err)
	}
	if status != string(domain.ImportRunFailed) {
		t.Fatalf("expected mixed-source run to be failed, got %s", status)
	}
}

func mustUUIDv7(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("generate UUIDv7: %v", err)
	}
	return id
}
