package immich

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestParseSearchPage(t *testing.T) {
	body, err := os.ReadFile("testdata/search_page1.json")
	if err != nil {
		t.Fatal(err)
	}
	assets, next, err := parseSearchPage(body)
	if err != nil {
		t.Fatalf("parseSearchPage: %v", err)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	if next != "2" {
		t.Errorf("expected nextPage 2, got %q", next)
	}

	a := assets[0]
	if a.ID != "asset-1" || a.SourceRef != "immich:asset:asset-1" {
		t.Errorf("unexpected id/ref: %q / %q", a.ID, a.SourceRef)
	}
	if a.Checksum != "abc123" {
		t.Errorf("unexpected checksum %q", a.Checksum)
	}
	if a.Coord == nil {
		t.Fatal("expected a GPS coord")
	}
	if a.Coord.Lon() != 139.6993 || a.Coord.Lat() != 35.6764 {
		t.Errorf("unexpected coord %v", *a.Coord)
	}
	want, _ := time.Parse(time.RFC3339, "2026-03-20T09:05:00.000Z")
	if !a.At.Equal(want) {
		t.Errorf("expected At %v, got %v", want, a.At)
	}
}

func TestParseSearchPageNoGPS(t *testing.T) {
	body, err := os.ReadFile("testdata/search_page2.json")
	if err != nil {
		t.Fatal(err)
	}
	assets, next, err := parseSearchPage(body)
	if err != nil {
		t.Fatalf("parseSearchPage: %v", err)
	}
	if next != "" {
		t.Errorf("expected empty nextPage, got %q", next)
	}
	if len(assets) != 1 {
		t.Fatalf("expected 1 asset, got %d", len(assets))
	}
	a := assets[0]
	if a.Coord != nil {
		t.Errorf("expected nil coord for no-GPS photo, got %v", *a.Coord)
	}
	// Falls back to localDateTime when dateTimeOriginal is null.
	want, _ := time.Parse(time.RFC3339, "2026-03-20T21:00:00.000Z")
	if !a.At.Equal(want) {
		t.Errorf("expected fallback At %v, got %v", want, a.At)
	}
}

func TestFetchAssetsFollowsCursor(t *testing.T) {
	page1, _ := os.ReadFile("testdata/search_page1.json")
	page2, _ := os.ReadFile("testdata/search_page2.json")

	var pages []int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/search/metadata" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "secret" {
			t.Errorf("missing x-api-key, got %q", r.Header.Get("x-api-key"))
		}
		var reqBody struct {
			Page        int    `json:"page"`
			Type        string `json:"type"`
			TakenAfter  string `json:"takenAfter"`
			TakenBefore string `json:"takenBefore"`
		}
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &reqBody)
		if reqBody.Type != "IMAGE" || reqBody.TakenAfter == "" || reqBody.TakenBefore == "" {
			t.Errorf("unexpected search body: %s", raw)
		}
		pages = append(pages, reqBody.Page)
		switch reqBody.Page {
		case 1:
			_, _ = w.Write(page1)
		case 2:
			_, _ = w.Write(page2)
		default:
			t.Errorf("unexpected page %d", reqBody.Page)
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "secret", srv.Client())
	assets, err := c.FetchAssets(context.Background(),
		time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets across 2 pages, got %d", len(assets))
	}
	if len(pages) != 2 || pages[0] != 1 || pages[1] != 2 {
		t.Errorf("expected page sequence [1 2], got %v", pages)
	}
	if assets[0].ID != "asset-1" || assets[1].ID != "asset-2" {
		t.Errorf("unexpected assets: %q, %q", assets[0].ID, assets[1].ID)
	}
}
