package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/runtime/importer"
)

func TestOpenConfiguresFileDatabaseForLocalRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "felicia.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if got := pragmaString(t, repo.db, "journal_mode"); got != "wal" {
		t.Fatalf("journal_mode = %q, want wal", got)
	}
	if got := pragmaInt(t, repo.db, "foreign_keys"); got != 1 {
		t.Fatalf("foreign_keys = %d, want 1", got)
	}
	if got := pragmaInt(t, repo.db, "busy_timeout"); got != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", got)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer reopened.Close()

	journal := &domain.Journal{ID: mustRuntimeUUID(t), CreatedAt: time.Now().UTC()}
	if err := reopened.CreateJournal(context.Background(), journal); err != nil {
		t.Fatalf("create journal after reopen: %v", err)
	}
	if _, err := reopened.GetJournal(context.Background(), journal.ID); err != nil {
		t.Fatalf("get journal after reopen: %v", err)
	}
}

func TestOpenConfiguresInMemoryDatabaseWithoutWAL(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	if got := pragmaString(t, repo.db, "journal_mode"); got != "memory" {
		t.Fatalf("journal_mode = %q, want memory", got)
	}
	if got := pragmaInt(t, repo.db, "foreign_keys"); got != 1 {
		t.Fatalf("foreign_keys = %d, want 1", got)
	}
}

func TestApplyPackageIsIdempotent(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()

	journeyID := mustRuntimeUUID(t)
	journalID := mustRuntimeUUID(t)
	mementoID := mustRuntimeUUID(t)
	photoID := mustRuntimeUUID(t)
	source := domain.SourceIdentity{System: "package:sample", ExternalID: mementoID.String()}
	document := &importer.PackageDocument{
		Journey:  &domain.Journey{ID: journeyID, JournalID: journalID, Slug: "sample", Title: "Sample", Place: "Kyoto", DateStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), GPSRoute: orb.MultiLineString{{{135.7, 35.0}, {135.8, 35.1}}}},
		Mementos: []*domain.Memento{{ID: mementoID, JourneyID: journeyID, Kind: "transit", Seq: 1, Title: "Train", Place: "Kyoto", Vendor: runtimeStringPtr("JR East"), Essay: runtimeStringPtr("A quiet departure."), PriceAmount: runtimeInt64Ptr(1800), PriceCurrency: runtimeStringPtr("JPY"), AuthoredFields: []string{"title", "vendor", "essay", "price_amount", "price_currency"}, KindData: []byte(`{"operator":"JR"}`), SourceIdentity: &source, State: domain.MementoPublished}},
		Photos:   []*domain.MementoPhoto{{ID: photoID, MementoID: mementoID, ObjectKey: "media/train.jpg", ContentHash: "sha256:train", Seq: 1}},
	}
	for attempt := 0; attempt < 2; attempt++ {
		if _, err := importer.ApplyPackage(context.Background(), document, repo); err != nil {
			t.Fatalf("apply package attempt %d: %v", attempt+1, err)
		}
	}
	journeys, err := repo.ListJourneys(context.Background())
	if err != nil || len(journeys) != 1 {
		t.Fatalf("journeys after repeat import: %d, %v", len(journeys), err)
	}
	mementos, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil || len(mementos) != 1 {
		t.Fatalf("mementos after repeat import: %d, %v", len(mementos), err)
	}
	if mementos[0].Vendor == nil || *mementos[0].Vendor != "JR East" || mementos[0].Essay == nil || *mementos[0].Essay != "A quiet departure." || mementos[0].PriceAmount == nil || *mementos[0].PriceAmount != 1800 || mementos[0].State != domain.MementoPublished {
		t.Fatalf("authored memento fields after import: %#v", mementos[0])
	}
	photos, err := repo.ListPhotosByMemento(context.Background(), mementoID)
	if err != nil || len(photos) != 1 {
		t.Fatalf("photos after repeat import: %d, %v", len(photos), err)
	}
}

// The admin intake inbox lists exactly what ListStopCandidatesByJourney
// returns, so a package import that writes no candidates leaves a CLI-imported
// trip permanently uncuratable (issue #79). This exercises the real store, end
// to end: import, curate, re-import.
func TestApplyPackagePersistsStopCandidatesForReview(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()

	journeyID := mustRuntimeUUID(t)
	journalID := mustRuntimeUUID(t)
	arrive := time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)
	stop := func(key, label string, offset time.Duration) *domain.StopCandidate {
		return &domain.StopCandidate{
			JourneyID:  journeyID,
			Identity:   domain.CandidateIdentity{DerivationVersion: "gpx-stops-v1", Key: key},
			Label:      label,
			Coord:      orb.Point{135.7681, 35.0116},
			Arrive:     arrive.Add(offset),
			Depart:     arrive.Add(offset + time.Hour),
			Confidence: 0.5,
			Evidence:   []domain.EvidenceRef{{Kind: domain.EvidenceRoute, Source: domain.SourceIdentity{System: "local-track", ExternalID: key}, Locator: key}},
			Provenance: []domain.Provenance{{Source: domain.SourceIdentity{System: "package:local-kyoto", ExternalID: key}, ObservedAt: arrive.Add(offset), Confidence: 0.5}},
		}
	}
	document := &importer.PackageDocument{
		Journey: &domain.Journey{ID: journeyID, JournalID: journalID, Slug: "stops", Title: "Stops", Place: "Kyoto", DateStart: arrive, DateEnd: arrive, GPSRoute: orb.MultiLineString{{{135.7, 35.0}, {135.8, 35.1}}}},
		Stops:   []*domain.StopCandidate{stop("cluster-001", "", 0), stop("cluster-002", "Nishiki", 4*time.Hour)},
	}

	report, err := importer.ApplyPackage(ctx, document, repo)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if report.Stops != 2 {
		t.Fatalf("report.Stops = %d, want 2", report.Stops)
	}
	candidates, err := repo.ListStopCandidatesByJourney(ctx, journeyID)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates after import = %d, want 2", len(candidates))
	}
	if candidates[0].State != domain.CandidateProposed || len(candidates[0].Provenance) != 1 || candidates[0].Provenance[0].Source.System != "package:local-kyoto" {
		t.Fatalf("imported candidate is not reviewable with honest provenance: %#v", candidates[0])
	}
	if len(candidates[0].Evidence) != 1 {
		t.Fatalf("imported candidate evidence = %#v", candidates[0].Evidence)
	}

	// The author curates in the GUI: discard the first, name the second.
	if err := repo.ApplyStopReview(ctx, &domain.StopReviewPatch{CandidateID: candidates[0].ID, State: domain.CandidateIgnored, ExpectedRevision: runtimeInt64Ptr(candidates[0].Revision)}); err != nil {
		t.Fatalf("discard candidate: %v", err)
	}
	if err := repo.ApplyStopReview(ctx, &domain.StopReviewPatch{CandidateID: candidates[1].ID, State: domain.CandidateKept, Label: runtimeStringPtr("Nishiki Market"), ExpectedRevision: runtimeInt64Ptr(candidates[1].Revision)}); err != nil {
		t.Fatalf("name candidate: %v", err)
	}

	if _, err := importer.ApplyPackage(ctx, document, repo); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	after, err := repo.ListStopCandidatesByJourney(ctx, journeyID)
	if err != nil {
		t.Fatalf("list candidates after re-import: %v", err)
	}
	if len(after) != 2 {
		t.Fatalf("re-import duplicated candidates: %d, want 2", len(after))
	}
	if after[0].State != domain.CandidateIgnored {
		t.Fatalf("re-import resurrected a discarded candidate: %q", after[0].State)
	}
	if after[1].State != domain.CandidateKept || after[1].Label != "Nishiki Market" {
		t.Fatalf("re-import overwrote the author's naming: %q / %q", after[1].State, after[1].Label)
	}
}

func runtimeStringPtr(value string) *string { return &value }

func runtimeInt64Ptr(value int64) *int64 { return &value }

func pragmaString(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	var value string
	if err := db.QueryRow("PRAGMA " + name).Scan(&value); err != nil {
		t.Fatalf("read pragma %s: %v", name, err)
	}
	return value
}

func pragmaInt(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var value int
	if err := db.QueryRow("PRAGMA " + name).Scan(&value); err != nil {
		t.Fatalf("read pragma %s: %v", name, err)
	}
	return value
}

func mustRuntimeUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
