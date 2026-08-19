// Package contract contains backend-neutral repository conformance tests.
package contract

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
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
		Slug:      fmt.Sprintf("contract-%s", mustUUID(t)),
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

	// Advance one legal step at a time (docs/contracts/memento-lifecycle.md §3):
	// draft→authored, then authored→published. A direct draft→published jump is
	// now illegal and is asserted separately below.
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
			State:      domain.MementoAuthored,
		},
		Fields:           []string{"occurred_at", "occurred_tz", "geom", "title", "place"},
		State:            domain.MementoAuthored,
		ExpectedRevision: int64Ptr(1),
	}); err != nil {
		t.Fatalf("author memento: %v", err)
	}
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento:          &domain.Memento{ID: memento.ID, JourneyID: journey.ID},
		State:            domain.MementoPublished,
		ExpectedRevision: int64Ptr(2),
	}); err != nil {
		t.Fatalf("publish memento: %v", err)
	}
	fetched, err = repo.GetMemento(ctx, memento.ID)
	if err != nil {
		t.Fatalf("get published memento: %v", err)
	}
	if fetched.State != domain.MementoPublished || fetched.Revision != 3 || fetched.Title != "Memory video" {
		t.Fatalf("unexpected published memento: %#v", fetched)
	}

	// An illegal transition on an existing row is rejected as InvalidTransitionError.
	var invalid *domain.InvalidTransitionError
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento:          &domain.Memento{ID: memento.ID, JourneyID: journey.ID},
		State:            domain.MementoDraft,
		ExpectedRevision: int64Ptr(3),
	}); !errors.As(err, &invalid) {
		t.Fatalf("published→draft error = %v, want InvalidTransitionError", err)
	}
	// Same-state save is always legal.
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento:          &domain.Memento{ID: memento.ID, JourneyID: journey.ID, Place: "Tokyo (same)"},
		Fields:           []string{"place"},
		State:            domain.MementoPublished,
		ExpectedRevision: int64Ptr(3),
	}); err != nil {
		t.Fatalf("same-state save: %v", err)
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

	assertSiteSettings(t, repo, journal.ID)
	assertAuthoredMementoFieldsSurviveIngest(t, repo, journey.ID)
	assertAuthoredJourneyFieldsSurviveIngest(t, repo, journal.ID)
}

// assertAuthoredMementoFieldsSurviveIngest pins the ADR-0022 guarantee that the
// importer "never overwrites authored fields — re-import is always safe" for
// mementos, in both providers (AGENTS.md development-flow constraint 2).
//
// It asserts, in order:
//   - an ingest write seeds every masked field on a fresh row;
//   - re-running the identical ingest is a no-op, not an error;
//   - after authoring title/place/kind_data, a third ingest carrying different
//     values leaves all three alone (invariant 1) and leaves the authored mask
//     intact (invariant 2), while still updating the unauthored occurred_tz
//     (invariant 4's "fields still update" half);
//   - an authoring write can still replace an already-authored field
//     (invariant 3).
func assertAuthoredMementoFieldsSurviveIngest(t *testing.T, repo domain.Repository, journeyID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	id := mustUUID(t)
	identity := domain.SourceIdentity{System: "contract-source", ExternalID: id.String()}
	ingest := func(title, place, tz, kindData string) *domain.IngestMementoPatch {
		return &domain.IngestMementoPatch{
			Memento: &domain.Memento{
				ID: id, JourneyID: journeyID, Kind: "ticket", Seq: 7,
				OccurredAt: time.Date(2026, 3, 21, 9, 0, 0, 0, time.UTC), OccurredTZ: tz,
				Geom: orb.Point{139.70, 35.60}, Title: title, Place: place,
				KindData: json.RawMessage(kindData), SourceIdentity: &identity,
				State: domain.MementoCandidateState,
			},
			Fields: []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "geom", "title", "place", "kind_data"},
		}
	}

	if err := repo.ApplyIngestMementoPatch(ctx, ingest("Machine title", "Machine place", "Asia/Tokyo", `{"operator":"machine"}`)); err != nil {
		t.Fatalf("first memento ingest: %v", err)
	}
	// Re-running the same import must not error (invariant 4).
	if err := repo.ApplyIngestMementoPatch(ctx, ingest("Machine title", "Machine place", "Asia/Tokyo", `{"operator":"machine"}`)); err != nil {
		t.Fatalf("repeat memento ingest: %v", err)
	}
	seeded, err := repo.GetMemento(ctx, id)
	if err != nil {
		t.Fatalf("get ingested memento: %v", err)
	}
	if seeded.Title != "Machine title" || seeded.Place != "Machine place" {
		t.Fatalf("ingest did not seed unauthored fields: %#v", seeded)
	}
	if len(seeded.AuthoredFields) != 0 {
		t.Fatalf("ingest claimed authorship: %#v", seeded.AuthoredFields)
	}

	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{
			ID: id, JourneyID: journeyID, Title: "Human title", Place: "Human place",
			KindData: json.RawMessage(`{"operator":"human"}`),
		},
		Fields: []string{"title", "place", "kind_data"},
		State:  domain.MementoDraft,
	}); err != nil {
		t.Fatalf("author memento fields: %v", err)
	}

	// Re-import with different source values for the authored fields and a new
	// value for an unauthored one.
	if err := repo.ApplyIngestMementoPatch(ctx, ingest("Machine title 2", "Machine place 2", "Asia/Osaka", `{"operator":"machine2"}`)); err != nil {
		t.Fatalf("re-import after authoring: %v", err)
	}
	after, err := repo.GetMemento(ctx, id)
	if err != nil {
		t.Fatalf("get memento after re-import: %v", err)
	}
	if after.Title != "Human title" {
		t.Fatalf("re-import overwrote authored title: %q", after.Title)
	}
	if after.Place != "Human place" {
		t.Fatalf("re-import overwrote authored place: %q", after.Place)
	}
	// Compared semantically: PostgreSQL stores kind_data as jsonb and returns a
	// re-serialized form, so a byte comparison would fail on whitespace alone.
	if operator := kindDataOperator(t, after.KindData); operator != "human" {
		t.Fatalf("re-import overwrote authored kind_data: %s", after.KindData)
	}
	for _, field := range []string{"title", "place", "kind_data"} {
		if !slices.Contains(after.AuthoredFields, field) {
			t.Fatalf("re-import shrank the authored mask: %#v", after.AuthoredFields)
		}
	}
	if after.OccurredTZ != "Asia/Osaka" {
		t.Fatalf("re-import failed to update unauthored occurred_tz: %q", after.OccurredTZ)
	}

	// Authoring is not read-only: an authored field can be re-authored.
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{ID: id, JourneyID: journeyID, Title: "Human title 2"},
		Fields:  []string{"title"},
	}); err != nil {
		t.Fatalf("re-author memento title: %v", err)
	}
	reauthored, err := repo.GetMemento(ctx, id)
	if err != nil {
		t.Fatalf("get re-authored memento: %v", err)
	}
	if reauthored.Title != "Human title 2" {
		t.Fatalf("authoring could not rewrite an authored field: %q", reauthored.Title)
	}
}

// assertAuthoredJourneyFieldsSurviveIngest pins the same four invariants for
// journeys, where the write path is split between the authoring upsert
// (UpsertJourney) and the ingest seam (ApplyIngestJourneyPatch) — see ADR-0033.
func assertAuthoredJourneyFieldsSurviveIngest(t *testing.T, repo domain.Repository, journalID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	id := mustUUID(t)
	slug := fmt.Sprintf("ingest-%s", id)
	fields := []string{"slug", "source_ref", "title", "place", "country", "region", "date_start", "date_end", "gps_route"}
	ingest := func(title, place string, start time.Time) *domain.IngestJourneyPatch {
		return &domain.IngestJourneyPatch{
			Journey: &domain.Journey{
				ID: id, JournalID: journalID, Slug: slug, Title: title, Place: place,
				DateStart: start, DateEnd: time.Date(2026, 5, 4, 0, 0, 0, 0, time.UTC),
				GPSRoute: orb.MultiLineString{{{135.5, 34.7}, {135.6, 34.8}}},
			},
			Fields: fields,
		}
	}

	// An ingest patch creates the row when it does not exist yet.
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	if err := repo.ApplyIngestJourneyPatch(ctx, ingest("Machine journey", "Kyoto", start)); err != nil {
		t.Fatalf("first journey ingest: %v", err)
	}
	// Re-running the same import must not error (invariant 4).
	if err := repo.ApplyIngestJourneyPatch(ctx, ingest("Machine journey", "Kyoto", start)); err != nil {
		t.Fatalf("repeat journey ingest: %v", err)
	}
	seeded, err := repo.GetJourney(ctx, id)
	if err != nil {
		t.Fatalf("get ingested journey: %v", err)
	}
	if seeded.Title != "Machine journey" || seeded.Place != "Kyoto" || len(seeded.GPSRoute) != 1 {
		t.Fatalf("journey ingest did not seed unauthored fields: %#v", seeded)
	}
	if len(seeded.AuthoredFields) != 0 {
		t.Fatalf("journey ingest claimed authorship: %#v", seeded.AuthoredFields)
	}

	// Authoring claims title and gps_route (invariant 3: the authoring write
	// sets the value and the mask, even though ingest already seeded them).
	authored := *seeded
	authored.Title = "Human journey"
	authored.GPSRoute = orb.MultiLineString{{{1.1, 2.2}, {3.3, 4.4}, {5.5, 6.6}}}
	authored.AuthoredFields = []string{"title", "gps_route"}
	if err := repo.UpsertJourney(ctx, &authored); err != nil {
		t.Fatalf("author journey: %v", err)
	}

	// Re-import with different source values.
	newStart := time.Date(2026, 4, 28, 0, 0, 0, 0, time.UTC)
	if err := repo.ApplyIngestJourneyPatch(ctx, ingest("Machine journey 2", "Osaka", newStart)); err != nil {
		t.Fatalf("re-import journey after authoring: %v", err)
	}
	after, err := repo.GetJourney(ctx, id)
	if err != nil {
		t.Fatalf("get journey after re-import: %v", err)
	}
	if after.Title != "Human journey" {
		t.Fatalf("re-import overwrote authored journey title: %q", after.Title)
	}
	if len(after.GPSRoute) != 1 || len(after.GPSRoute[0]) != 3 {
		t.Fatalf("re-import overwrote authored gps_route: %#v", after.GPSRoute)
	}
	if len(after.AuthoredFields) != 2 || !slices.Contains(after.AuthoredFields, "title") || !slices.Contains(after.AuthoredFields, "gps_route") {
		t.Fatalf("re-import reset the journey authored mask: %#v", after.AuthoredFields)
	}
	if after.Place != "Osaka" {
		t.Fatalf("re-import failed to update unauthored place: %q", after.Place)
	}
	if !after.DateStart.Equal(newStart) {
		t.Fatalf("re-import failed to update unauthored date_start: %s", after.DateStart)
	}

	// Authoring is not read-only: an already-authored journey field can be
	// rewritten (invariant 3).
	reauthor := *after
	reauthor.Title = "Human journey 2"
	if err := repo.UpsertJourney(ctx, &reauthor); err != nil {
		t.Fatalf("re-author journey title: %v", err)
	}
	reauthored, err := repo.GetJourney(ctx, id)
	if err != nil {
		t.Fatalf("get re-authored journey: %v", err)
	}
	if reauthored.Title != "Human journey 2" {
		t.Fatalf("authoring could not rewrite an authored journey field: %q", reauthored.Title)
	}
}

// assertSiteSettings exercises the site settings round trip (ADMIN-02 M2):
// not-found before any upsert, upsert-then-read equality, and upsert-again
// replacing (not duplicating) the single row scoped to journalID.
func assertSiteSettings(t *testing.T, repo domain.Repository, journalID uuid.UUID) {
	t.Helper()
	ctx := context.Background()

	soleJournal, err := repo.GetSoleJournal(ctx)
	if err != nil {
		t.Fatalf("get sole journal: %v", err)
	}
	if soleJournal.ID != journalID {
		t.Fatalf("get sole journal: got %s, want %s", soleJournal.ID, journalID)
	}

	if _, err := repo.GetSiteSettings(ctx, journalID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("get site settings before upsert: got %v, want domain.ErrNotFound", err)
	}

	settings := &domain.SiteSettings{
		JournalID:       journalID,
		Title:           "Contract Journal",
		Description:     "A contract test journal",
		Design:          "cabinet",
		DefaultLanguage: "en",
		DefaultTheme:    "light",
		Accent:          "#ea580c",
	}
	if err := repo.UpsertSiteSettings(ctx, settings); err != nil {
		t.Fatalf("upsert site settings: %v", err)
	}
	fetched, err := repo.GetSiteSettings(ctx, journalID)
	if err != nil {
		t.Fatalf("get site settings: %v", err)
	}
	if fetched.Title != settings.Title || fetched.Description != settings.Description ||
		fetched.Design != settings.Design || fetched.DefaultLanguage != settings.DefaultLanguage ||
		fetched.DefaultTheme != settings.DefaultTheme || fetched.Accent != settings.Accent {
		t.Fatalf("unexpected site settings after upsert: %#v", fetched)
	}

	replacement := &domain.SiteSettings{
		JournalID:       journalID,
		Title:           "Replaced Title",
		Description:     "Replaced description",
		Design:          "techo",
		DefaultLanguage: "zh",
		DefaultTheme:    "dark",
		Accent:          "#123456",
	}
	if err := repo.UpsertSiteSettings(ctx, replacement); err != nil {
		t.Fatalf("upsert site settings again: %v", err)
	}
	replaced, err := repo.GetSiteSettings(ctx, journalID)
	if err != nil {
		t.Fatalf("get site settings after replace: %v", err)
	}
	if replaced.Title != replacement.Title || replaced.Design != replacement.Design ||
		replaced.DefaultLanguage != replacement.DefaultLanguage || replaced.DefaultTheme != replacement.DefaultTheme ||
		replaced.Accent != replacement.Accent {
		t.Fatalf("site settings did not replace, got %#v", replaced)
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

// kindDataOperator decodes the "operator" key the authored-field assertions use
// as a kind_data marker, so providers may re-serialize the JSON freely.
func kindDataOperator(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var decoded struct {
		Operator string `json:"operator"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decode kind_data %s: %v", raw, err)
	}
	return decoded.Operator
}
