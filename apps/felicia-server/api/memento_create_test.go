package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	"github.com/azusachino/felicia/apps/felicia-server/api"
)

// These tests drive POST /api/admin/mementos/create through the server
// transport with the real SQLite repository, mirroring the desktop
// transport: the same admin bundle can target either host via VITE_API_BASE,
// so both must expose identical creation semantics
// (docs/contracts/automatic-sequence-allocation.md).

func newCreateSeqServer(t *testing.T) (http.Handler, *sqlite.Repository) {
	t.Helper()
	repo, err := sqlite.Open(filepath.Join(t.TempDir(), "test.sqlite"))
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	if err := repo.CreateJournal(context.Background(), &domain.Journal{ID: uuid.Must(uuid.NewV7()), CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	srv := api.NewServer(repo, loadKinds(t), api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{})
	return srv.Handler(), repo
}

func seedServerCreateJourney(t *testing.T, repo *sqlite.Repository) uuid.UUID {
	t.Helper()
	journey := &domain.Journey{
		ID:        uuid.Must(uuid.NewV7()),
		JournalID: mustJournalID(t, repo),
		Slug:      "server-create-seq",
		Title:     "Server create sequence journey",
		Place:     "Kyoto",
		DateStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(context.Background(), journey); err != nil {
		t.Fatalf("seed journey: %v", err)
	}
	return journey.ID
}

func mustJournalID(t *testing.T, repo *sqlite.Repository) uuid.UUID {
	t.Helper()
	journal, err := repo.GetSoleJournal(context.Background())
	if err != nil {
		t.Fatalf("sole journal: %v", err)
	}
	return journal.ID
}

func postServerCreate(t *testing.T, h http.Handler, body any) (*httptest.ResponseRecorder, *domain.Memento) {
	t.Helper()
	raw, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/admin/mementos/create", strings.NewReader(string(raw)))
	request.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, request)
	row := &domain.Memento{}
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), row); err != nil {
			t.Fatalf("decode created row: %v (%s)", err, rec.Body.String())
		}
	}
	return rec, row
}

func serverCreateBody(id, journeyID uuid.UUID, title string) map[string]any {
	return map[string]any{
		"id": id, "journey_id": journeyID, "kind": "ticket",
		"title": title, "place": "Kyoto",
		"occurred_at": "2026-04-02T10:00:00Z", "occurred_tz": "Asia/Tokyo",
		"kind_data": map[string]any{}, "state": "draft",
	}
}

func TestServerCreateMementoAllocatesAndRetriesStably(t *testing.T) {
	h, repo := newCreateSeqServer(t)
	journeyID := seedServerCreateJourney(t, repo)

	recA, rowA := postServerCreate(t, h, serverCreateBody(uuid.Must(uuid.NewV7()), journeyID, "Alpha"))
	if recA.Code != http.StatusOK {
		t.Fatalf("first create: status %d body %s", recA.Code, recA.Body.String())
	}
	if rowA.Seq != 0 || rowA.Revision != 1 || rowA.State != domain.MementoDraft {
		t.Fatalf("first row seq=%d revision=%d state=%s, want 0/1/draft", rowA.Seq, rowA.Revision, rowA.State)
	}
	recB, rowB := postServerCreate(t, h, serverCreateBody(uuid.Must(uuid.NewV7()), journeyID, "Beta"))
	if recB.Code != http.StatusOK || rowB.Seq != 1 || rowB.Revision != 1 {
		t.Fatalf("second create: status %d seq=%d revision=%d", recB.Code, rowB.Seq, rowB.Revision)
	}
	// Same-ID retry with matching values returns the stored row unchanged.
	body := serverCreateBody(rowA.ID, journeyID, "Alpha")
	body["occurred_at"] = "2026-04-02T10:00:00Z"
	recRetry, rowRetry := postServerCreate(t, h, body)
	if recRetry.Code != http.StatusOK {
		t.Fatalf("retry: %d %s", recRetry.Code, recRetry.Body.String())
	}
	if rowRetry.ID != rowA.ID || rowRetry.Seq != rowA.Seq || rowRetry.Revision != rowA.Revision {
		t.Fatalf("retry changed the row: %d/%d vs %d/%d", rowRetry.Seq, rowRetry.Revision, rowA.Seq, rowA.Revision)
	}
	rows, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil || len(rows) != 2 {
		t.Fatalf("persisted rows=%d err=%v", len(rows), err)
	}
}

func TestServerCreateMementoIncompatibleRetryConflicts(t *testing.T) {
	h, repo := newCreateSeqServer(t)
	journeyID := seedServerCreateJourney(t, repo)
	other := &domain.Journey{
		ID: uuid.Must(uuid.NewV7()), JournalID: mustJournalID(t, repo), Slug: "server-create-seq-other",
		Title: "Other", Place: "Kyoto",
		DateStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(context.Background(), other); err != nil {
		t.Fatalf("seed other journey: %v", err)
	}

	id := uuid.Must(uuid.NewV7())
	if rec, _ := postServerCreate(t, h, serverCreateBody(id, journeyID, "Original title")); rec.Code != http.StatusOK {
		t.Fatalf("seed create: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ := postServerCreate(t, h, serverCreateBody(id, journeyID, "Edited title")); rec.Code != http.StatusConflict {
		t.Fatalf("changed-title retry: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ := postServerCreate(t, h, serverCreateBody(id, other.ID, "Original title")); rec.Code != http.StatusConflict {
		t.Fatalf("different-journey retry: %d %s", rec.Code, rec.Body.String())
	}
	rows, _ := repo.ListMementosByJourney(context.Background(), journeyID)
	if len(rows) != 1 || rows[0].Title != "Original title" || rows[0].Revision != 1 {
		t.Fatalf("stored row changed: %+v", rows)
	}
}

func TestServerCreateMementoRejectsInvalidBodies(t *testing.T) {
	h, repo := newCreateSeqServer(t)
	journeyID := seedServerCreateJourney(t, repo)

	cases := map[string]func(map[string]any){
		"missing id":           func(b map[string]any) { b["id"] = uuid.Nil },
		"missing journey":      func(b map[string]any) { b["journey_id"] = uuid.Nil },
		"missing title":        func(b map[string]any) { b["title"] = "" },
		"client seq":           func(b map[string]any) { b["seq"] = 3 },
		"expected revision":    func(b map[string]any) { b["expected_revision"] = 1 },
		"invalid kind":         func(b map[string]any) { b["kind"] = "nonexistent" },
		"invalid date":         func(b map[string]any) { b["occurred_at"] = "not-a-date" },
		"invalid timezone":     func(b map[string]any) { b["occurred_tz"] = "Mars/Olympus" },
		"non-draft state":      func(b map[string]any) { b["state"] = "published" },
		"non-object kind_data": func(b map[string]any) { b["kind_data"] = "nope" },
		"null kind_data":       func(b map[string]any) { b["kind_data"] = nil },
		"array kind_data":      func(b map[string]any) { b["kind_data"] = []any{} },
		"whitespace title":     func(b map[string]any) { b["title"] = "   " },
		"forged authored_fields": func(b map[string]any) {
			b["authored_fields"] = []string{"journey_id", "kind"}
		},
		"empty authored_fields":  func(b map[string]any) { b["authored_fields"] = []string{} },
		"null authored_fields":   func(b map[string]any) { b["authored_fields"] = nil },
		"null seq":               func(b map[string]any) { b["seq"] = nil },
		"null expected_revision": func(b map[string]any) { b["expected_revision"] = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			body := serverCreateBody(uuid.Must(uuid.NewV7()), journeyID, "Invalid "+name)
			mutate(body)
			rec, _ := postServerCreate(t, h, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: status %d, want 400: %s", name, rec.Code, rec.Body.String())
			}
			rows, err := repo.ListMementosByJourney(context.Background(), journeyID)
			if err != nil || len(rows) != 0 {
				t.Fatalf("%s: persisted %d rows err=%v", name, len(rows), err)
			}
		})
	}
}

func TestServerCreateMementoUnknownFieldRejected(t *testing.T) {
	h, repo := newCreateSeqServer(t)
	journeyID := seedServerCreateJourney(t, repo)
	body := serverCreateBody(uuid.Must(uuid.NewV7()), journeyID, "Geom")
	body["geom"] = nil
	rec, _ := postServerCreate(t, h, body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown field: status %d, want 400: %s", rec.Code, rec.Body.String())
	}
	rows, _ := repo.ListMementosByJourney(context.Background(), journeyID)
	if len(rows) != 0 {
		t.Fatalf("unknown-field request persisted %d rows", len(rows))
	}
}

// canonicalManualMask is the authorship a manual HTTP creation must carry:
// the same derived ownership the old manual upsert path gave a new row.
func canonicalManualMask(t *testing.T, mask []string, where string) {
	t.Helper()
	have := map[string]bool{}
	for _, field := range mask {
		have[field] = true
	}
	for _, want := range []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "title", "place", "kind_data"} {
		if !have[want] {
			t.Errorf("%s: authored mask missing %q; mask=%v", where, want, mask)
		}
	}
}

// TestServerCreateMementoManualAuthorshipSurvivesIngest creates through the
// HTTP endpoint with NO client-authored_fields and proves the stored row
// keeps the manual authorship (and its values) when a later ingest targets
// the same identity: protected authored fields are never overwritten.
func TestServerCreateMementoManualAuthorshipSurvivesIngest(t *testing.T) {
	h, repo := newCreateSeqServer(t)
	journeyID := seedServerCreateJourney(t, repo)

	body := serverCreateBody(uuid.Must(uuid.NewV7()), journeyID, "Manual draft")
	rec, created := postServerCreate(t, h, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: status %d body %s", rec.Code, rec.Body.String())
	}
	canonicalManualMask(t, created.AuthoredFields, "created row response")

	rows, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("persisted rows=%d err=%v", len(rows), err)
	}
	canonicalManualMask(t, rows[0].AuthoredFields, "persisted row")
	if rows[0].Seq != 0 {
		t.Fatalf("pre-ingest seq=%d, want the allocated 0", rows[0].Seq)
	}

	ingestAt := time.Date(2026, 5, 9, 8, 0, 0, 0, time.UTC)
	if err := repo.ApplyIngestMementoPatch(context.Background(), &domain.IngestMementoPatch{
		Memento: &domain.Memento{
			ID: created.ID, JourneyID: journeyID, Kind: "ticket", Seq: 99,
			OccurredAt: ingestAt, OccurredTZ: "Asia/Tokyo",
			Title: "Imported title", Place: "Imported place", KindData: []byte(`{"imported":true}`),
		},
		Fields: []string{"title", "place", "seq", "kind_data", "occurred_at", "occurred_tz"},
	}); err != nil {
		t.Fatalf("ingest patch: %v", err)
	}

	after, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil || len(after) != 1 {
		t.Fatalf("rows after ingest=%d err=%v", len(after), err)
	}
	stored := after[0]
	canonicalManualMask(t, stored.AuthoredFields, "row after ingest")
	if stored.Title != "Manual draft" {
		t.Errorf("authored title overwritten by ingest: %q", stored.Title)
	}
	if stored.Place != "Kyoto" {
		t.Errorf("authored place overwritten by ingest: %q", stored.Place)
	}
	if stored.Seq != 0 {
		t.Errorf("authored seq overwritten by ingest: %d", stored.Seq)
	}
	if !stored.OccurredAt.Equal(time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("authored occurred_at overwritten by ingest: %v", stored.OccurredAt)
	}
	if stored.OccurredTZ != "Asia/Tokyo" {
		t.Errorf("authored occurred_tz overwritten by ingest: %q", stored.OccurredTZ)
	}
	if string(stored.KindData) != "{}" {
		t.Errorf("authored kind_data overwritten by ingest: %s", stored.KindData)
	}
}
