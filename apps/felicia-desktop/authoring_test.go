package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestDesktopMementoEssayIsPersistedAndCanBeCleared(t *testing.T) {
	h, repo, _ := setupTestHandler(t, "admin", "")
	created := httptest.NewRecorder()
	h.ServeHTTP(created, httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(
		`{"slug":"essay-save","title":"Synthetic journey","place":"Kyoto","date_start":"2026-03-20","date_end":"2026-03-21"}`,
	)))
	if created.Code != http.StatusOK {
		t.Fatalf("create journey: %d %s", created.Code, created.Body.String())
	}
	var journey struct{ ID uuid.UUID }
	if err := json.Unmarshal(created.Body.Bytes(), &journey); err != nil {
		t.Fatal(err)
	}
	id := uuid.Must(uuid.NewV7())
	for revision, essay := range []string{"Original story", "Edited story", ""} {
		revisionField := ""
		if revision > 0 {
			revisionField = fmt.Sprintf(`,"expected_revision":%d`, revision)
		}
		body := fmt.Sprintf(`{"id":%q,"journey_id":%q,"kind":"goods","seq":1,"title":"Postcard","place":"Kyoto","state":"draft","kind_data":{"name":"Synthetic postcard"},"essay":%q%s}`, id, journey.ID, essay, revisionField)
		response := httptest.NewRecorder()
		h.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/admin/mementos", strings.NewReader(body)))
		if response.Code != http.StatusOK {
			t.Fatalf("save %d: %d %s", revision, response.Code, response.Body.String())
		}
		stored, err := repo.GetMemento(context.Background(), id)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Essay == nil || *stored.Essay != essay || !slices.Contains(stored.AuthoredFields, "essay") {
			t.Fatalf("save %d did not persist authored essay: %+v", revision, stored)
		}
	}
}
