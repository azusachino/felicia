package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJourneyCreationRejectsSlugCollisionWithoutReplacingOriginal(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	for index, title := range []string{"Original", "Replacement"} {
		body := fmt.Sprintf(`{"slug":"reserved-slug","title":%q,"place":"Kyoto","date_start":"2026-05-01","date_end":"2026-05-03"}`, title)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(body)))
		want := http.StatusOK
		if index == 1 {
			want = http.StatusConflict
		}
		if rec.Code != want {
			t.Fatalf("creation %d: status %d, want %d: %s", index, rec.Code, want, rec.Body.String())
		}
	}
	rows, err := repo.ListJourneys(context.Background())
	if err != nil || len(rows) != 1 || rows[0].Title != "Original" {
		t.Fatalf("collision changed original: rows=%v err=%v", rows, err)
	}
}

func TestJourneyCreationRejectsReversedDatesWithoutSaving(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	body := `{"slug":"invalid-dates","title":"Invalid","place":"Kyoto","date_start":"2026-05-03","date_end":"2026-05-01"}`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	rows, err := repo.ListJourneys(context.Background())
	if err != nil || len(rows) != 0 {
		t.Fatalf("invalid request saved a row: rows=%v err=%v", rows, err)
	}
}
