package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// seedGPSJourney stores a journey carrying a passive GPS route, a source
// reference and an extra authored claim, as an ingest write would leave it.
// The browser API cannot seed a GPS route, so the handler-level retention of
// route/source/mask across a metadata-only edit is verified here.
func seedGPSJourney(t *testing.T, repo interface {
	UpsertJourney(context.Context, *domain.Journey) error
	GetJourney(context.Context, uuid.UUID) (*domain.Journey, error)
}) *domain.Journey {
	t.Helper()
	journey := &domain.Journey{
		ID:             uuid.Must(uuid.NewV7()),
		JournalID:      uuid.MustParse("0190cbde-f300-7000-8000-111111111111"),
		Slug:           "seeded-trip",
		SourceRef:      strPtr("dawarich:trip-42"),
		Title:          "Seeded trip",
		Place:          "Kyoto",
		DateStart:      time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC),
		GPSRoute:       orb.MultiLineString{{{135.7, 35.0}, {135.8, 35.1}}},
		AuthoredFields: []string{"source_ref"},
	}
	if err := repo.UpsertJourney(context.Background(), journey); err != nil {
		t.Fatalf("seed journey: %v", err)
	}
	stored, err := repo.GetJourney(context.Background(), journey.ID)
	if err != nil {
		t.Fatalf("reload seeded journey: %v", err)
	}
	return stored
}

func strPtr(s string) *string { return &s }

// snapshotOfSlug reloads a journey by slug so an independent SQLite read can
// back an unchanged-row assertion.
func snapshotOfSlug(t *testing.T, repo interface {
	GetJourneyBySlug(context.Context, string) (*domain.Journey, error)
}, slug string) *domain.Journey {
	t.Helper()
	stored, err := repo.GetJourneyBySlug(context.Background(), slug)
	if err != nil {
		t.Fatalf("snapshot %q: %v", slug, err)
	}
	return stored
}

func metadataEditBody(journey *domain.Journey, slug string, start, end string, revision int64) string {
	return fmt.Sprintf(
		`{"id":%q,"journal_id":%q,"slug":%q,"source_ref":%q,"title":"Edited trip","place":"Osaka","date_start":%q,"date_end":%q,"expected_revision":%d}`,
		journey.ID, journey.JournalID, slug, *journey.SourceRef, start, end, revision,
	)
}

func TestJourneyMetadataEditRetainsRouteSourceAndMaskAndAdvancesRevisionExactly(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	seeded := seedGPSJourney(t, repo)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		metadataEditBody(seeded, seeded.Slug, "2026-06-10", "2026-06-15", seeded.Revision),
	)))
	if rec.Code != http.StatusOK {
		t.Fatalf("metadata edit: status %d: %s", rec.Code, rec.Body.String())
	}
	var edited domain.Journey
	if err := json.Unmarshal(rec.Body.Bytes(), &edited); err != nil {
		t.Fatal(err)
	}

	// The persisted row is reloaded independently from SQLite and compared
	// against the handler response, so a stale/echoed response cannot pass.
	persisted, err := repo.GetJourney(context.Background(), seeded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(persisted, &edited) {
		t.Fatalf("persisted row differs from response: row=%+v response=%+v", persisted, edited)
	}
	if edited.ID != seeded.ID || edited.JournalID != seeded.JournalID || edited.Slug != seeded.Slug {
		t.Fatalf("identity changed: id=%v journal=%v slug=%q", edited.ID, edited.JournalID, edited.Slug)
	}
	if edited.SourceRef == nil || *edited.SourceRef != *seeded.SourceRef {
		t.Fatalf("source_ref not retained: %v", edited.SourceRef)
	}
	if !reflect.DeepEqual(edited.GPSRoute, seeded.GPSRoute) {
		t.Fatalf("gps route not retained: got %v want %v", edited.GPSRoute, seeded.GPSRoute)
	}
	if edited.Title != "Edited trip" || edited.Place != "Osaka" {
		t.Fatalf("metadata edit not applied: %q %q", edited.Title, edited.Place)
	}
	if got, want := edited.DateStart.Format("2006-01-02"), "2026-06-10"; got != want {
		t.Fatalf("date_start %q, want %q", got, want)
	}
	if got, want := edited.DateEnd.Format("2006-01-02"), "2026-06-15"; got != want {
		t.Fatalf("date_end %q, want %q", got, want)
	}
	if edited.Revision != seeded.Revision+1 {
		t.Fatalf("revision %d, want exactly %d", edited.Revision, seeded.Revision+1)
	}
	wantMask := make([]string, 0, len(domain.JourneyAuthoredFields)+1)
	wantMask = append(wantMask, domain.JourneyAuthoredFields...)
	wantMask = append(wantMask, "source_ref")
	for _, field := range wantMask {
		if !slices.Contains(edited.AuthoredFields, field) {
			t.Fatalf("authored mask lost %q: %v", field, edited.AuthoredFields)
		}
	}
}

func TestJourneyMetadataEditFailuresPreserveRowThenObservedRevisionRetrySucceeds(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	seeded := seedGPSJourney(t, repo)
	snapshot := func(id uuid.UUID) *domain.Journey {
		t.Helper()
		stored, err := repo.GetJourney(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		return stored
	}
	unchanged := func(stage string, before *domain.Journey) {
		t.Helper()
		after := snapshot(before.ID)
		if !reflect.DeepEqual(after, before) {
			t.Fatalf("%s: row changed: before=%+v after=%+v", stage, before, after)
		}
	}

	// invalid date range is rejected without touching the row
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		metadataEditBody(seeded, seeded.Slug, "2026-06-15", "2026-06-10", seeded.Revision),
	)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reversed dates: status %d: %s", rec.Code, rec.Body.String())
	}
	unchanged("reversed dates", seeded)

	// slug collision with a separate journey is rejected without touching either row
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		`{"slug":"other-trip","title":"Other trip","place":"Nara","date_start":"2026-05-01","date_end":"2026-05-02"}`,
	)))
	if rec.Code != http.StatusOK {
		t.Fatalf("create collision target: status %d: %s", rec.Code, rec.Body.String())
	}
	neighbourBefore := snapshotOfSlug(t, repo, "other-trip")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		metadataEditBody(seeded, "other-trip", "2026-06-10", "2026-06-15", seeded.Revision),
	)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("slug collision: status %d: %s", rec.Code, rec.Body.String())
	}
	unchanged("slug collision", seeded)
	unchanged("slug collision (neighbour)", neighbourBefore)

	// stale expected revision is rejected without touching the row
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		metadataEditBody(seeded, seeded.Slug, "2026-06-10", "2026-06-15", seeded.Revision-1),
	)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("stale save: status %d: %s", rec.Code, rec.Body.String())
	}
	unchanged("stale save", seeded)

	// retry with the newly observed revision succeeds and advances by exactly one
	fresh := snapshot(seeded.ID)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		metadataEditBody(fresh, fresh.Slug, "2026-06-10", "2026-06-15", fresh.Revision),
	)))
	if rec.Code != http.StatusOK {
		t.Fatalf("retry with observed revision: status %d: %s", rec.Code, rec.Body.String())
	}
	var edited domain.Journey
	if err := json.Unmarshal(rec.Body.Bytes(), &edited); err != nil {
		t.Fatal(err)
	}
	if edited.Revision != fresh.Revision+1 {
		t.Fatalf("retry revision %d, want exactly %d", edited.Revision, fresh.Revision+1)
	}
	if edited.Title != "Edited trip" || !reflect.DeepEqual(edited.GPSRoute, seeded.GPSRoute) {
		t.Fatalf("retry lost edit or route: %+v", edited)
	}
	persisted, err := repo.GetJourney(context.Background(), seeded.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(persisted, &edited) {
		t.Fatalf("retry: persisted row differs from response: row=%+v response=%+v", persisted, edited)
	}
	if !reflect.DeepEqual(persisted.GPSRoute, seeded.GPSRoute) || persisted.SourceRef == nil || *persisted.SourceRef != *seeded.SourceRef {
		t.Fatalf("retry lost route or source_ref: %+v", persisted)
	}
	unchanged("after successful retry (neighbour)", neighbourBefore)
}
