package dawarich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestParseTracks(t *testing.T) {
	body, err := os.ReadFile("testdata/tracks.json")
	if err != nil {
		t.Fatal(err)
	}
	routes, err := parseTracks(body)
	if err != nil {
		t.Fatalf("parseTracks: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(routes))
	}

	r := routes[0]
	if len(r.Line) != 2 {
		t.Errorf("expected 2 points, got %d", len(r.Line))
	}
	if r.Line[0].Lon() != 139.7671 || r.Line[0].Lat() != 35.6812 {
		t.Errorf("unexpected first coord %v", r.Line[0])
	}
	if r.DistanceM != 6300 {
		t.Errorf("expected distance 6300, got %d", r.DistanceM)
	}
	if r.Mode != "walking" {
		t.Errorf("expected mode walking, got %q", r.Mode)
	}
	if r.SourceRef != "dawarich:track:42" {
		t.Errorf("unexpected source ref %q", r.SourceRef)
	}
	want, _ := time.Parse(time.RFC3339, "2026-03-20T09:00:00Z")
	if !r.From.Equal(want) {
		t.Errorf("expected From %v, got %v", want, r.From)
	}
}

func TestParseVisits(t *testing.T) {
	body, err := os.ReadFile("testdata/visits.json")
	if err != nil {
		t.Fatal(err)
	}
	visits, err := parseVisits(body)
	if err != nil {
		t.Fatalf("parseVisits: %v", err)
	}
	// The declined visit is dropped.
	if len(visits) != 2 {
		t.Fatalf("expected 2 visits (declined filtered), got %d", len(visits))
	}

	v := visits[0]
	if v.Coord.Lon() != 139.6993 || v.Coord.Lat() != 35.6764 {
		t.Errorf("unexpected coord %v", v.Coord)
	}
	if v.Label != "明治神宮" {
		t.Errorf("expected label 明治神宮, got %q", v.Label)
	}
	if v.SourceRef != "dawarich:visit:7" {
		t.Errorf("unexpected source ref %q", v.SourceRef)
	}
	if v.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %v", v.Confidence)
	}
	for _, v := range visits {
		if v.Label == "Declined stop" {
			t.Error("declined visit should have been dropped")
		}
	}
}

func TestFetchRoutesPaginates(t *testing.T) {
	const page1 = `{"type":"FeatureCollection","features":[
		{"type":"Feature","geometry":{"type":"LineString","coordinates":[[139.0,35.0],[139.1,35.1]]},
		 "properties":{"id":1,"start_at":"2026-03-20T09:00:00Z","end_at":"2026-03-20T10:00:00Z","distance":100,"dominant_mode":"walking"}}]}`
	const page2 = `{"type":"FeatureCollection","features":[
		{"type":"Feature","geometry":{"type":"LineString","coordinates":[[135.0,34.0],[135.1,34.1]]},
		 "properties":{"id":2,"start_at":"2026-03-20T11:00:00Z","end_at":"2026-03-20T12:00:00Z","distance":200,"dominant_mode":"train"}}]}`

	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("api_key") != "secret" && r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("missing auth: query=%q header=%q", q.Get("api_key"), r.Header.Get("Authorization"))
		}
		if q.Get("start_at") == "" || q.Get("end_at") == "" {
			t.Errorf("missing time range: %v", q)
		}
		gotPaths = append(gotPaths, r.URL.Path+"?page="+q.Get("page"))
		w.Header().Set("X-Total-Pages", "2")
		switch q.Get("page") {
		case "1":
			_, _ = w.Write([]byte(page1))
		case "2":
			_, _ = w.Write([]byte(page2))
		default:
			t.Errorf("unexpected page %q", q.Get("page"))
		}
	}))
	defer srv.Close()

	c := New(srv.URL, "secret", srv.Client())
	routes, err := c.FetchRoutes(context.Background(),
		time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchRoutes: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes across 2 pages, got %d", len(routes))
	}
	if len(gotPaths) != 2 {
		t.Fatalf("expected 2 page requests, got %v", gotPaths)
	}
	if routes[0].SourceRef != "dawarich:track:1" || routes[1].SourceRef != "dawarich:track:2" {
		t.Errorf("unexpected routes across pages: %q, %q", routes[0].SourceRef, routes[1].SourceRef)
	}
}
