package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/azusachino/felicia/server/api"
)

type siteSettingsPayload struct {
	Title           string `json:"title"`
	Description     string `json:"description"`
	Design          string `json:"design"`
	DefaultLanguage string `json:"default_language"`
	DefaultTheme    string `json:"default_theme"`
	Accent          string `json:"accent"`
}

func decodeSiteSettings(t *testing.T, w *httptest.ResponseRecorder) siteSettingsPayload {
	t.Helper()
	var got siteSettingsPayload
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode site settings: %v (%s)", err, w.Body)
	}
	return got
}

// A fresh DB (no site settings saved yet, only the always-present sole
// journal) must resolve to defaults with a 200 — never a 404 — matching the
// static compiler's "absent settings = current demo behavior" (ADMIN-02 M2 §4).
func TestGetSiteSettingsDefaultsOnEmptyDB(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/site-settings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get site settings on empty DB = %d, want 200 (%s)", w.Code, w.Body)
	}

	got := decodeSiteSettings(t, w)
	want := siteSettingsPayload{Design: "v1", DefaultLanguage: "ja", DefaultTheme: "dark"}
	if got != want {
		t.Errorf("defaults = %+v, want %+v", got, want)
	}
}

func TestPutSiteSettingsValidRoundTrip(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	body, _ := json.Marshal(map[string]any{
		"title":            "Aaron's Waypoints",
		"description":      "A travel journal",
		"design":           "v3",
		"default_language": "en",
		"default_theme":    "light",
		"accent":           "#ea580c",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("put site settings = %d, want 200 (%s)", w.Code, w.Body)
	}
	want := siteSettingsPayload{
		Title: "Aaron's Waypoints", Description: "A travel journal", Design: "v3",
		DefaultLanguage: "en", DefaultTheme: "light", Accent: "#ea580c",
	}
	if got := decodeSiteSettings(t, w); got != want {
		t.Errorf("put response = %+v, want %+v", got, want)
	}

	// The save round-trips through a subsequent GET.
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/admin/site-settings", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get after put = %d, want 200 (%s)", w.Code, w.Body)
	}
	if got := decodeSiteSettings(t, w); got != want {
		t.Errorf("get after put = %+v, want %+v", got, want)
	}
}

func TestPutSiteSettingsRejectsInvalidFields(t *testing.T) {
	cases := []struct {
		name  string
		field string
		body  map[string]any
	}{
		{"invalid design", "design", map[string]any{"design": "v9"}},
		{"invalid default_language", "default_language", map[string]any{"default_language": "fr"}},
		{"invalid default_theme", "default_theme", map[string]any{"default_theme": "sepia"}},
		{"invalid accent", "accent", map[string]any{"accent": "orange"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPut, "/api/admin/site-settings", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != http.StatusUnprocessableEntity {
				t.Fatalf("%s: status = %d, want 422 (%s)", tc.name, w.Code, w.Body)
			}
			var res map[string]any
			if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
				t.Fatalf("decode error response: %v", err)
			}
			issues, ok := res["issues"].([]any)
			if !ok || len(issues) == 0 {
				t.Fatalf("%s: expected validation issues, got %v", tc.name, res)
			}
			found := false
			for _, raw := range issues {
				issue, ok := raw.(map[string]any)
				if ok && issue["Field"] == tc.field {
					found = true
				}
			}
			if !found {
				t.Errorf("%s: issues %v do not mention field %q", tc.name, issues, tc.field)
			}
		})
	}
}

// An accent of "" (unsetting it) is legal — only a non-empty, malformed value
// is rejected.
func TestPutSiteSettingsAllowsClearingAccent(t *testing.T) {
	handler := api.NewServer(newMockRepository(), nil, api.NewCacheManager("", testLogger), testLogger, nil, api.RouteConfig{}).Handler()

	body, _ := json.Marshal(map[string]any{"accent": ""})
	req := httptest.NewRequest(http.MethodPut, "/api/admin/site-settings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("clear accent status = %d, want 200 (%s)", w.Code, w.Body)
	}
}
