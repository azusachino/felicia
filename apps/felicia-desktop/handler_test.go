package main

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
)

func setupTestHandler(t *testing.T, mode string, token string) (*DesktopHandler, *sqlite.Repository, string) {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.sqlite")
	mediaDir := filepath.Join(dir, "media")
	publicDir := filepath.Join(dir, "public")

	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = repo.Close() })

	journalID := uuid.MustParse("0190cbde-f300-7000-8000-111111111111")
	if err := repo.CreateJournal(context.Background(), &domain.Journal{ID: journalID, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("create journal: %v", err)
	}

	h, err := NewHandler(HandlerConfig{
		Repo:      repo,
		MediaRoot: mediaDir,
		PublicDir: publicDir,
		Mode:      mode,
		Token:     token,
		OnPickFolder: func(_ string) (string, error) {
			return "/test/path/to/folder", nil
		},
	})
	if err != nil {
		t.Fatalf("new handler: %v", err)
	}
	return h, repo, dir
}

func TestHandlerBootPrefs(t *testing.T) {
	h, _, _ := setupTestHandler(t, "admin", "")
	req := httptest.NewRequest(http.MethodGet, "/boot.js", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "window.bootPrefs =") {
		t.Fatalf("expected window.bootPrefs, got %s", rec.Body.String())
	}
}

func TestHandlerJourneysCRUD(t *testing.T) {
	h, _, _ := setupTestHandler(t, "admin", "secret-token")

	// 1. List initially empty
	req := httptest.NewRequest(http.MethodGet, "/api/admin/journeys", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list journeys: expected 200, got %d", rec.Code)
	}

	// 2. Token required for creation
	body := `{"slug":"trip-1","title":"Trip One","place":"Kyoto","date_start":"2026-05-01","date_end":"2026-05-03"}`
	req = httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(body))
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when token omitted, got %d", rec.Code)
	}

	// 3. Create with token
	req = httptest.NewRequest(http.MethodPost, "/api/admin/journeys", strings.NewReader(body))
	req.Header.Set("X-Desktop-Token", "secret-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("create journey: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	var created domain.Journey
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created journey: %v", err)
	}
	if created.Title != "Trip One" || created.Slug != "trip-1" {
		t.Fatalf("unexpected journey data: %+v", created)
	}

	// 4. Get journey by ID
	req = httptest.NewRequest(http.MethodGet, "/api/admin/journeys/"+created.ID.String(), nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get journey: expected 200, got %d", rec.Code)
	}

	// 5. Delete journey with token
	req = httptest.NewRequest(http.MethodDelete, "/api/admin/journeys/"+created.ID.String(), nil)
	req.Header.Set("X-Desktop-Token", "secret-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete journey: expected 200, got %d", rec.Code)
	}
}

func TestHandlerReaderModeBlocksAdmin(t *testing.T) {
	h, _, _ := setupTestHandler(t, "reader", "")

	req := httptest.NewRequest(http.MethodGet, "/api/admin/journeys", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("reader mode must return 403 for /api/admin/*, got %d", rec.Code)
	}
}

func TestHandlerPickFolder(t *testing.T) {
	h, _, _ := setupTestHandler(t, "admin", "secure-token")

	// 1. Missing token returns 403 Forbidden
	req := httptest.NewRequest(http.MethodPost, "/api/desktop/pick-folder", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 when token omitted, got %d", rec.Code)
	}

	// 2. Invalid token returns 403 Forbidden
	req = httptest.NewRequest(http.MethodPost, "/api/desktop/pick-folder", nil)
	req.Header.Set("X-Desktop-Token", "wrong-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 on invalid token, got %d", rec.Code)
	}

	// 3. Valid token returns 200 OK
	req = httptest.NewRequest(http.MethodPost, "/api/desktop/pick-folder", strings.NewReader(`{"title":"Pick workspace"}`))
	req.Header.Set("X-Desktop-Token", "secure-token")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("pick folder: expected 200, got %d", rec.Code)
	}

	var res struct {
		Selected bool   `json:"selected"`
		Path     string `json:"path"`
		Basename string `json:"basename"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal pick folder response: %v", err)
	}
	if !res.Selected || res.Basename != "folder" {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestHandlerSiteSettingsAndCompile(t *testing.T) {
	h, _, _ := setupTestHandler(t, "admin", "token-123")

	// Get settings
	req := httptest.NewRequest(http.MethodGet, "/api/admin/site-settings", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get settings: expected 200, got %d", rec.Code)
	}

	// Update settings with token
	updateBody := `{"title":"My Travel Studio","default_theme":"dark"}`
	req = httptest.NewRequest(http.MethodPut, "/api/admin/site-settings", strings.NewReader(updateBody))
	req.Header.Set("X-Desktop-Token", "token-123")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("put settings: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}

	// Compile with token
	req = httptest.NewRequest(http.MethodPost, "/api/admin/compile", strings.NewReader(`{}`))
	req.Header.Set("X-Desktop-Token", "token-123")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("compile: expected 200, got %d (%s)", rec.Code, rec.Body.String())
	}
}
