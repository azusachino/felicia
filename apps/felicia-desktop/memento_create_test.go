package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

// These tests drive POST /api/admin/mementos/create through the desktop
// transport with the real SQLite repository: automatic positions sit above
// the journey's existing maximum, a same-ID retry returns the stored row
// unchanged, and incompatible retries or invalid bodies leave no row and
// never touch stored state (docs/contracts/automatic-sequence-allocation.md).

var createSeqJournalID = uuid.MustParse("0190cbde-f300-7000-8000-111111111111")

func seedCreateSeqJourney(t *testing.T, repo *sqlite.Repository, slug string) uuid.UUID {
	t.Helper()
	journey := &domain.Journey{
		ID:        uuid.Must(uuid.NewV7()),
		JournalID: createSeqJournalID,
		Slug:      slug,
		Title:     "Create sequence journey",
		Place:     "Kyoto",
		DateStart: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 4, 3, 0, 0, 0, 0, time.UTC),
	}
	if err := repo.UpsertJourney(context.Background(), journey); err != nil {
		t.Fatalf("seed journey: %v", err)
	}
	return journey.ID
}

func seedCreateSeqAnchor(t *testing.T, repo *sqlite.Repository, journeyID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if err := repo.ApplyManualMementoPatch(context.Background(), &domain.ManualMementoPatch{
		Memento: &domain.Memento{ID: id, JourneyID: journeyID, Kind: "ticket", Seq: 4, State: domain.MementoDraft, Title: "Anchor seq four", KindData: []byte(`{}`)},
		Fields:  []string{"kind", "seq", "title", "kind_data"},
		State:   domain.MementoDraft,
	}); err != nil {
		t.Fatalf("seed anchor memento: %v", err)
	}
	return id
}

func createSeqBody(id, journeyID uuid.UUID, title string) map[string]any {
	return map[string]any{
		"id": id, "journey_id": journeyID, "kind": "ticket",
		"title": title, "place": "Kyoto",
		"occurred_at": "2026-04-02T10:00:00Z", "occurred_tz": "Asia/Tokyo",
		"kind_data": map[string]any{}, "state": "draft",
	}
}

func postCreateSeq(t *testing.T, h http.Handler, body any) (*httptest.ResponseRecorder, *domain.Memento) {
	t.Helper()
	raw, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/mementos/create", strings.NewReader(string(raw))))
	row := &domain.Memento{}
	if rec.Code == http.StatusOK {
		if err := json.Unmarshal(rec.Body.Bytes(), row); err != nil {
			t.Fatalf("decode created row: %v (%s)", err, rec.Body.String())
		}
	}
	return rec, row
}

func listCreateSeqRows(t *testing.T, repo *sqlite.Repository, journeyID uuid.UUID) []*domain.Memento {
	t.Helper()
	rows, err := repo.ListMementosByJourney(context.Background(), journeyID)
	if err != nil {
		t.Fatalf("list mementos: %v", err)
	}
	return rows
}

func createSeqRowByTitle(rows []*domain.Memento, title string) *domain.Memento {
	for _, row := range rows {
		if row.Title == title {
			return row
		}
	}
	return nil
}

func int64Ptr(value int64) *int64 { return &value }

func TestDesktopCreateMementoAllocatesAboveExistingMax(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	journeyID := seedCreateSeqJourney(t, repo, "create-alpha")
	seedCreateSeqAnchor(t, repo, journeyID)

	recA, rowA := postCreateSeq(t, h, createSeqBody(uuid.Must(uuid.NewV7()), journeyID, "Alpha"))
	if recA.Code != http.StatusOK {
		t.Fatalf("first create: status %d body %s", recA.Code, recA.Body.String())
	}
	if rowA.Seq != 5 || rowA.Revision != 1 || rowA.State != domain.MementoDraft {
		t.Fatalf("first row seq=%d revision=%d state=%s, want 5/1/draft", rowA.Seq, rowA.Revision, rowA.State)
	}
	recB, rowB := postCreateSeq(t, h, createSeqBody(uuid.Must(uuid.NewV7()), journeyID, "Beta"))
	if recB.Code != http.StatusOK || rowB.Seq != 6 || rowB.Revision != 1 {
		t.Fatalf("second create: status %d seq=%d revision=%d", recB.Code, rowB.Seq, rowB.Revision)
	}
	rows := listCreateSeqRows(t, repo, journeyID)
	if len(rows) != 3 {
		t.Fatalf("persisted rows=%d, want 3", len(rows))
	}
	if anchor := createSeqRowByTitle(rows, "Anchor seq four"); anchor == nil || anchor.Seq != 4 {
		t.Fatalf("anchor row moved or missing: %+v", anchor)
	}
}

func TestDesktopCreateMementoRetrySamePayloadReturnsStoredRow(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	journeyID := seedCreateSeqJourney(t, repo, "create-retry")
	body := createSeqBody(uuid.Must(uuid.NewV7()), journeyID, "Retryable")
	recA, rowA := postCreateSeq(t, h, body)
	if recA.Code != http.StatusOK {
		t.Fatalf("first create: %d %s", recA.Code, recA.Body.String())
	}
	recB, rowB := postCreateSeq(t, h, body)
	if recB.Code != http.StatusOK {
		t.Fatalf("retry: %d %s", recB.Code, recB.Body.String())
	}
	if rowB.ID != rowA.ID || rowB.Seq != rowA.Seq || rowB.Revision != rowA.Revision {
		t.Fatalf("retry changed the row: %d/%d vs %d/%d", rowB.Seq, rowB.Revision, rowA.Seq, rowA.Revision)
	}
	if rows := listCreateSeqRows(t, repo, journeyID); len(rows) != 1 {
		t.Fatalf("retry created rows: %d", len(rows))
	}
}

func TestDesktopCreateMementoIncompatibleRetryConflicts(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	journeyID := seedCreateSeqJourney(t, repo, "create-conflict")
	otherJourney := seedCreateSeqJourney(t, repo, "create-conflict-other")

	id := uuid.Must(uuid.NewV7())
	if rec, _ := postCreateSeq(t, h, createSeqBody(id, journeyID, "Original title")); rec.Code != http.StatusOK {
		t.Fatalf("seed create: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ := postCreateSeq(t, h, createSeqBody(id, journeyID, "Edited title")); rec.Code != http.StatusConflict {
		t.Fatalf("changed-title retry: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ := postCreateSeq(t, h, createSeqBody(id, otherJourney, "Original title")); rec.Code != http.StatusConflict {
		t.Fatalf("different-journey retry: %d %s", rec.Code, rec.Body.String())
	}
	// An intervening edit must never be overwritten by a create retry.
	if err := repo.ApplyManualMementoPatch(context.Background(), &domain.ManualMementoPatch{
		Memento:          &domain.Memento{ID: id, JourneyID: journeyID, Kind: "ticket", Seq: 5, State: domain.MementoDraft, Title: "Manually edited", KindData: []byte(`{}`)},
		Fields:           []string{"title"},
		State:            domain.MementoDraft,
		ExpectedRevision: int64Ptr(1),
	}); err != nil {
		t.Fatalf("intervening edit: %v", err)
	}
	if rec, _ := postCreateSeq(t, h, createSeqBody(id, journeyID, "Original title")); rec.Code != http.StatusConflict {
		t.Fatalf("post-edit retry: %d %s", rec.Code, rec.Body.String())
	}
	rows := listCreateSeqRows(t, repo, journeyID)
	if len(rows) != 1 || rows[0].Title != "Manually edited" || rows[0].Revision != 2 {
		t.Fatalf("stored row changed: %+v", rows)
	}
}

func TestDesktopCreateMementoRejectsInvalidBodies(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	journeyID := seedCreateSeqJourney(t, repo, "create-invalid")

	cases := map[string]func(map[string]any){
		"missing id":           func(b map[string]any) { b["id"] = uuid.Nil },
		"missing journey":      func(b map[string]any) { b["journey_id"] = uuid.Nil },
		"missing kind":         func(b map[string]any) { b["kind"] = "" },
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
		"unknown field":          func(b map[string]any) { b["geom"] = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			body := createSeqBody(uuid.Must(uuid.NewV7()), journeyID, "Invalid "+name)
			mutate(body)
			rec, _ := postCreateSeq(t, h, body)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("%s: status %d, want 400: %s", name, rec.Code, rec.Body.String())
			}
			if rows := listCreateSeqRows(t, repo, journeyID); len(rows) != 0 {
				t.Fatalf("%s: persisted %d rows", name, len(rows))
			}
		})
	}
}

func TestDesktopCreateMementoMissingJourneyFailsWithoutRow(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	missing := uuid.Must(uuid.NewV7())
	rec, _ := postCreateSeq(t, h, createSeqBody(uuid.Must(uuid.NewV7()), missing, "Orphan"))
	if rec.Code == http.StatusOK {
		t.Fatalf("creation against a nonexistent journey must fail: %s", rec.Body.String())
	}
	if rows := listCreateSeqRows(t, repo, missing); len(rows) != 0 {
		t.Fatalf("partial row persisted: %d", len(rows))
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

// TestDesktopCreateMementoManualAuthorshipSurvivesIngest creates through the
// HTTP endpoint with NO client-authored_fields and proves the stored row
// keeps the manual authorship (and its values) when a later ingest targets
// the same identity: protected authored fields are never overwritten.
func TestDesktopCreateMementoManualAuthorshipSurvivesIngest(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	journeyID := seedCreateSeqJourney(t, repo, "create-authorship")

	body := createSeqBody(uuid.Must(uuid.NewV7()), journeyID, "Manual draft")
	rec, created := postCreateSeq(t, h, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("create: status %d body %s", rec.Code, rec.Body.String())
	}
	canonicalManualMask(t, created.AuthoredFields, "created row response")

	rows := listCreateSeqRows(t, repo, journeyID)
	if len(rows) != 1 {
		t.Fatalf("persisted rows=%d", len(rows))
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

	after := listCreateSeqRows(t, repo, journeyID)
	t.Logf("DEBUG pre-ingest row: %+v", rows[0])
	if len(after) != 1 {
		t.Fatalf("rows after ingest=%d", len(after))
	}
	stored := after[0]
	canonicalManualMask(t, stored.AuthoredFields, "row after ingest")
	// Values survive; a legitimate ingest revision bump is allowed.
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
