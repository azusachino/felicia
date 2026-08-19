package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-server/api"
)

// handleGetPublicSite must reflect an admin PUT immediately: the admin write
// invalidates the public cache, so the next live GET recomputes from the
// repository rather than serving a stale cached projection.
func TestHandleGetPublicSiteReflectsAdminPut(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	// Defaults before any admin write — never a 404.
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/site", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get public site (defaults) = %d, want 200 (%s)", w.Code, w.Body)
	}
	// Capture the raw body before decodeSiteSettings drains w.Body (it reads
	// via json.Decoder, which consumes the recorder's buffer).
	firstBody := w.Body.String()
	defaults := decodeSiteSettings(t, w)
	if defaults.Design != "cartography" || defaults.DefaultLanguage != "ja" || defaults.DefaultTheme != "dark" {
		t.Fatalf("unexpected defaults: %+v", defaults)
	}

	// The .json alias shares the same handler and must match the same JSON
	// content. Trim trailing whitespace since a cache-hit response (raw
	// json.Marshal output, no caching in this test though) and a cache-miss
	// response (respondJSON's encoder, which appends "\n") can differ only
	// by that incidental trailing byte — see handleGetPublicJourneys for the
	// same existing pattern.
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, httptest.NewRequest(http.MethodGet, "/api/v1/site.json", nil))
	if w2.Code != http.StatusOK || strings.TrimRight(w2.Body.String(), "\n") != strings.TrimRight(firstBody, "\n") {
		t.Fatalf("site.json alias = %d %q, want 200 matching /api/v1/site body %q", w2.Code, w2.Body.String(), firstBody)
	}

	body, _ := json.Marshal(map[string]any{
		"title":  "Aaron's Waypoints",
		"design": "atlas",
		"accent": "#123abc",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("put site settings = %d, want 200 (%s)", w.Code, w.Body)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/site", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get public site (after put) = %d, want 200 (%s)", w.Code, w.Body)
	}
	got := decodeSiteSettings(t, w)
	if got.Title != "Aaron's Waypoints" || got.Design != "atlas" || got.Accent != "#123abc" {
		t.Errorf("public site after put = %+v, want the saved values reflected", got)
	}
}
