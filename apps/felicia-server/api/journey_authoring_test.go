package api_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

// The authoring API used to take the client's authored mask at face value and
// assign every column from the request. A save that omitted either therefore
// destroyed data twice over (ADR-0039): the mask was cleared, so the next
// import legally overwrote the author's work, and gps_route was written as
// null, so the stored trace was blanked by the save itself.
//
// These pin the invariants ADR-0039 states, which is what makes them testable
// before #82 builds the form that would otherwise reach them first.

func saveJourney(t *testing.T, handler http.Handler, body string) {
	t.Helper()
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/journeys", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(w, request)
	if w.Code != http.StatusOK {
		t.Fatalf("upsert journey: expected 200, got %d (%s)", w.Code, w.Body)
	}
}

func authoringHandler(t *testing.T) (http.Handler, *mockRepository) {
	t.Helper()
	repo := newMockRepository()
	reg := loadKinds(t)
	return api.NewServer(repo, reg, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler(), repo
}

func TestUpsertJourneyClaimsItsOwnAuthoredFields(t *testing.T) {
	handler, repo := authoringHandler(t)
	id := uuid.New()

	// No authored_fields in the body at all -- the shape that used to clear it.
	saveJourney(t, handler, `{"id":"`+id.String()+`","journal_id":"`+uuid.NewString()+`","slug":"izu","title":"Izu","place":"Izu Peninsula","date_start":"2026-08-01","date_end":"2026-08-02"}`)

	stored := repo.journeys[id]
	if stored == nil {
		t.Fatal("journey was not stored")
	}
	for _, field := range domain.JourneyAuthoredFields {
		if !slices.Contains(stored.AuthoredFields, field) {
			t.Errorf("authoring write did not claim %q; mask is %v", field, stored.AuthoredFields)
		}
	}
	if slices.Contains(stored.AuthoredFields, "gps_route") {
		t.Error("authoring write claimed gps_route; ingest must keep owning the trace")
	}
}

func TestUpsertJourneyLeavesTheStoredRouteAlone(t *testing.T) {
	handler, repo := authoringHandler(t)
	id := uuid.New()
	route := orb.MultiLineString{{{139.0, 35.0}, {139.7, 35.6}}}
	repo.journeys[id] = &domain.Journey{
		ID:             id,
		Slug:           "izu",
		Title:          "Imported name",
		Place:          "Izu Peninsula",
		DateStart:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		GPSRoute:       route,
		AuthoredFields: []string{},
	}

	// A save that carries no route. This used to write gps_route = null.
	saveJourney(t, handler, `{"id":"`+id.String()+`","journal_id":"`+uuid.NewString()+`","slug":"izu","title":"The name I chose","place":"Izu Peninsula","date_start":"2026-08-01","date_end":"2026-08-02"}`)

	stored := repo.journeys[id]
	if len(stored.GPSRoute) != len(route) {
		t.Fatalf("stored route was blanked by an authoring save: got %v, want %v", stored.GPSRoute, route)
	}
	if stored.Title != "The name I chose" {
		t.Errorf("authoring write did not apply the title: %q", stored.Title)
	}
}

func TestUpsertJourneyNeverShrinksTheStoredMask(t *testing.T) {
	handler, repo := authoringHandler(t)
	id := uuid.New()
	// A field the authoring surface does not write, claimed by some earlier
	// path. Invariant 4: no sequence of authoring writes may drop it.
	repo.journeys[id] = &domain.Journey{
		ID:             id,
		Slug:           "izu",
		Title:          "Izu",
		Place:          "Izu Peninsula",
		DateStart:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		AuthoredFields: []string{"gps_route"},
	}

	saveJourney(t, handler, `{"id":"`+id.String()+`","journal_id":"`+uuid.NewString()+`","slug":"izu","title":"Izu","place":"Izu Peninsula","date_start":"2026-08-01","date_end":"2026-08-02"}`)

	if !slices.Contains(repo.journeys[id].AuthoredFields, "gps_route") {
		t.Errorf("authoring write dropped a stored claim; mask is %v", repo.journeys[id].AuthoredFields)
	}
}

func TestIngestCannotOverwriteWhatAuthoringJustClaimed(t *testing.T) {
	handler, repo := authoringHandler(t)
	id := uuid.New()
	repo.journeys[id] = &domain.Journey{
		ID:             id,
		Slug:           "izu",
		Title:          "Imported name",
		Place:          "Izu Peninsula",
		DateStart:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:        time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		AuthoredFields: []string{},
	}

	saveJourney(t, handler, `{"id":"`+id.String()+`","journal_id":"`+uuid.NewString()+`","slug":"izu","title":"The name I chose","place":"Izu Peninsula","date_start":"2026-08-01","date_end":"2026-08-02"}`)

	// The whole point of the mask: an import arriving afterwards must not win.
	if err := repo.ApplyIngestJourneyPatch(context.Background(), &domain.IngestJourneyPatch{
		Journey: &domain.Journey{ID: id, Title: "Imported name", Place: "Somewhere else"},
		Fields:  []string{"title", "place"},
	}); err != nil {
		t.Fatalf("ingest patch: %v", err)
	}

	if got := repo.journeys[id].Title; got != "The name I chose" {
		t.Errorf("import overwrote an authored title: got %q", got)
	}
}
