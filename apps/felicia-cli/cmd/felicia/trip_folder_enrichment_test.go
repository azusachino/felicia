package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
)

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewRandom()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func writeTripMetadata(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "trip.json"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func enrichmentPlan(segments int) intake.DraftPlan {
	plan := intake.DraftPlan{DateStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), DateEnd: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)}
	for i := 0; i < segments; i++ {
		plan.Routes = append(plan.Routes, domain.Route{Line: orb.LineString{{1, 1}, {2, 2}}})
	}
	return plan
}

func TestBuildPackageDocumentEnrichmentContract(t *testing.T) {
	journeyID := mustUUID(t)
	journalID := mustUUID(t)

	t.Run("valid metadata is carried through typed", func(t *testing.T) {
		dir := t.TempDir()
		writeTripMetadata(t, dir, `{"leg_modes":["car","ferry"],"weather":{"2026-07-01":{"icon":"sun","label":"8–19°C","min":8,"max":19}}}`)
		doc, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(2), nil)
		if err != nil {
			t.Fatalf("buildPackageDocument: %v", err)
		}
		if len(doc.Journey.LegModes) != 2 || doc.Journey.LegModes[0] != domain.LegModeCar || doc.Journey.LegModes[1] != domain.LegModeFerry {
			t.Errorf("unexpected leg modes: %v", doc.Journey.LegModes)
		}
		if doc.Journey.Weather == nil || doc.Journey.Weather["2026-07-01"].Icon != "sun" || doc.Journey.Weather["2026-07-01"].Min == nil {
			t.Errorf("unexpected weather: %#v", doc.Journey.Weather)
		}
	})

	t.Run("unknown leg mode fails loudly", func(t *testing.T) {
		dir := t.TempDir()
		writeTripMetadata(t, dir, `{"leg_modes":["car","hovercraft"]}`)
		if _, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(2), nil); err == nil {
			t.Fatal("expected unknown leg mode to be rejected")
		}
	})

	t.Run("malformed weather fails loudly", func(t *testing.T) {
		dir := t.TempDir()
		writeTripMetadata(t, dir, `{"weather":{"2026-07-01":"sunny"}}`)
		if _, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(1), nil); err == nil {
			t.Fatal("expected malformed weather to be rejected")
		}
		dir = t.TempDir()
		writeTripMetadata(t, dir, `{"weather":{"2026-07-01":{"label":"no icon"}}}`)
		if _, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(1), nil); err == nil {
			t.Fatal("expected weather without icon to be rejected")
		}
	})

	t.Run("leg mode count must match route segment count", func(t *testing.T) {
		dir := t.TempDir()
		writeTripMetadata(t, dir, `{"leg_modes":["car"]}`)
		_, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(2), nil)
		if err == nil {
			t.Fatal("expected leg mode/segment count mismatch to be rejected")
		}
	})

	t.Run("missing metadata stays absent, never invented", func(t *testing.T) {
		dir := t.TempDir()
		restore := fetchWeather
		fetchWeather = func(float64, float64, time.Time, time.Time) domain.Weather { return nil }
		defer func() { fetchWeather = restore }()
		doc, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(2), nil)
		if err != nil {
			t.Fatalf("buildPackageDocument: %v", err)
		}
		if len(doc.Journey.LegModes) != 0 {
			t.Errorf("expected no invented leg modes, got %v", doc.Journey.LegModes)
		}
		if doc.Journey.Weather != nil {
			t.Errorf("expected no invented weather, got %#v", doc.Journey.Weather)
		}
	})

	t.Run("weather.json sidecar validates typed", func(t *testing.T) {
		dir := t.TempDir()
		writeTripMetadata(t, dir, `{}`)
		if err := os.WriteFile(filepath.Join(dir, "weather.json"), []byte(`{"2026-07-01":{"icon":"sun"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		doc, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(1), nil)
		if err != nil {
			t.Fatalf("buildPackageDocument: %v", err)
		}
		if doc.Journey.Weather["2026-07-01"].Icon != "sun" {
			t.Errorf("unexpected weather: %#v", doc.Journey.Weather)
		}
		if err := os.WriteFile(filepath.Join(dir, "weather.json"), []byte(`{"2026-07-01":{}}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := buildPackageDocument(journeyID, journalID, "trip", "Trip", "Place", TripFolderConfig{Dir: dir}, enrichmentPlan(1), nil); err == nil {
			t.Fatal("expected weather.json without icon to be rejected")
		}
	})
}
