package publication

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

func TestExtractImmichAssetID(t *testing.T) {
	cases := []struct {
		input    string
		expected string
		ok       bool
	}{
		{"immich:asset:00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000001", true},
		{"immich:asset-42", "asset-42", true},
		{"immich/photo-1", "photo-1", true},
		{"local/photo.jpg", "", false},
		{"", "", false},
	}

	for _, c := range cases {
		got, ok := ExtractImmichAssetID(c.input)
		if ok != c.ok || got != c.expected {
			t.Errorf("ExtractImmichAssetID(%q) = (%q, %v); want (%q, %v)", c.input, got, ok, c.expected, c.ok)
		}
	}
}

func TestExportJourneyGeoJSON_ReadModel(t *testing.T) {
	journeyID := uuid.New()
	slug := "test-journey"
	country := "Japan"
	region := "Shizuoka"

	journey := &domain.Journey{
		ID:        journeyID,
		Slug:      slug,
		Title:     "Test Journey",
		Place:     "Izu",
		Country:   &country,
		Region:    &region,
		DateStart: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC),
		GPSRoute: orb.MultiLineString{
			orb.LineString{
				orb.Point{139.1234567, 34.9876543},
				orb.Point{139.1250000, 34.9850000},
				orb.Point{139.1300000, 34.9800000},
				orb.Point{139.1350000, 34.9750000},
				orb.Point{139.1400000, 34.9700000},
				orb.Point{139.1450000, 34.9650000},
				orb.Point{139.1567891, 34.9543219},
			},
		},
	}

	stopID := uuid.New()
	stopCandidate := &domain.StopCandidate{
		ID:        stopID,
		JourneyID: journeyID,
		Identity: domain.CandidateIdentity{
			DerivationVersion: "test-v1",
			Key:               "stop-1",
		},
		Label:  "Scenic Viewpoint",
		Coord:  orb.Point{139.1234567, 34.9876543},
		Arrive: time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC),
		Depart: time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC),
		State:  domain.CandidateKept,
	}

	mementoPubID := uuid.New()
	essayText := "A fine day on the coast."
	photoRef := "immich:asset:11111111-2222-3333-4444-555555555555"
	captionText := "Coast View"

	publishedMemento := &domain.Memento{
		ID:         mementoPubID,
		JourneyID:  journeyID,
		Kind:       "souvenir",
		Seq:        1,
		OccurredAt: time.Date(2026, 8, 1, 10, 15, 0, 0, time.UTC),
		OccurredTZ: "Asia/Tokyo",
		Geom:       orb.Point{139.1234567, 34.9876543},
		Title:      "Coastal Souvenir",
		Place:      "Scenic Viewpoint",
		Essay:      &essayText,
		State:      domain.MementoPublished,
		SourceRef:  stringPtr("stop:stop-1"),
	}

	draftMemento := &domain.Memento{
		ID:         uuid.New(),
		JourneyID:  journeyID,
		Kind:       "ticket",
		Seq:        2,
		OccurredAt: time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC),
		OccurredTZ: "Asia/Tokyo",
		Geom:       orb.Point{139.1300, 34.9800},
		Title:      "Draft Ticket",
		State:      domain.MementoDraft, // must be excluded
	}

	read := &fakeReadModel{
		journeys: []*domain.Journey{journey},
		mementos: map[uuid.UUID][]*domain.Memento{
			journeyID: {publishedMemento, draftMemento},
		},
		photos: map[uuid.UUID][]*domain.MementoPhoto{
			mementoPubID: {
				{
					ID:        uuid.New(),
					MementoID: mementoPubID,
					SourceRef: &photoRef,
					Caption:   &captionText,
					Seq:       1,
				},
			},
		},
		stopCandidates: map[uuid.UUID][]*domain.StopCandidate{
			journeyID: {stopCandidate},
		},
	}

	// 1. Normal export with asset resolution
	col, err := ExportJourneyGeoJSON(context.Background(), slug, read, ExportOptions{
		Generator: "felicia-cli export test",
		Days: []GeoJSONDay{
			{Day: 1, Date: "2026-08-01", Title: "Coastal Arrival"},
		},
	})
	if err != nil {
		t.Fatalf("ExportJourneyGeoJSON failed: %v", err)
	}

	if col.Schema != GeoJSONSchemaV1 {
		t.Errorf("schema = %q, want %q", col.Schema, GeoJSONSchemaV1)
	}
	if col.Properties.Slug != slug {
		t.Errorf("slug = %q, want %q", col.Properties.Slug, slug)
	}
	if len(col.Properties.Days) != 1 || col.Properties.Days[0].Title != "Coastal Arrival" {
		t.Errorf("unexpected days: %+v", col.Properties.Days)
	}

	// Verify features
	var routeFeat, stopFeat, mementoFeat *GeoJSONExportFeature
	for i := range col.Features {
		f := &col.Features[i]
		switch f.Properties["role"] {
		case "route":
			routeFeat = f
		case "stop":
			stopFeat = f
		case "memento":
			mementoFeat = f
		}
	}

	if routeFeat == nil {
		t.Fatal("missing route feature")
	}
	if stopFeat == nil {
		t.Fatal("missing stop feature")
	}
	if mementoFeat == nil {
		t.Fatal("missing memento feature")
	}

	// Verify coordinate rounding to <= 4 decimals
	routeGeom, ok := routeFeat.Geometry.(map[string]any)
	if !ok || routeGeom["type"] != "LineString" {
		t.Fatalf("unexpected route geometry: %+v", routeFeat.Geometry)
	}
	coords := routeGeom["coordinates"].([][]float64)
	for _, pt := range coords {
		assertCoordinateDecimals(t, pt[0], pt[1])
	}

	// Verify stop coordinates and matching key
	stopGeom := stopFeat.Geometry.(map[string]any)
	stopCoord := stopGeom["coordinates"].([]float64)
	assertCoordinateDecimals(t, stopCoord[0], stopCoord[1])
	if stopFeat.ID != "stop-1" {
		t.Errorf("stop id = %q, want %q", stopFeat.ID, "stop-1")
	}

	// Verify memento stop_key referential integrity
	if mementoFeat.Properties["stop_key"] != "stop-1" {
		t.Errorf("memento stop_key = %q, want %q", mementoFeat.Properties["stop_key"], "stop-1")
	}
	mediaList := mementoFeat.Properties["media"].([]GeoJSONMedia)
	if len(mediaList) != 1 || mediaList[0].ImmichAssetID != "11111111-2222-3333-4444-555555555555" {
		t.Errorf("unexpected media list: %+v", mediaList)
	}

	// 2. Decision D12: Photo without Immich asset ID must fail export unless DeferAssets is true
	badRef := "local:unresolved.jpg"
	read.photos[mementoPubID] = []*domain.MementoPhoto{
		{
			ID:        uuid.New(),
			MementoID: mementoPubID,
			SourceRef: &badRef,
			Seq:       1,
		},
	}
	_, err = ExportJourneyGeoJSON(context.Background(), slug, read, ExportOptions{})
	if err == nil {
		t.Fatal("expected D12 failure for unresolved photo, got nil error")
	}

	// 3. Asset-deferred export succeeds even with unresolved photos
	deferredCol, err := ExportJourneyGeoJSON(context.Background(), slug, read, ExportOptions{
		DeferAssets: true,
	})
	if err != nil {
		t.Fatalf("asset deferred export failed: %v", err)
	}
	for _, f := range deferredCol.Features {
		if f.Properties["role"] == "memento" {
			deferredMedia := f.Properties["media"].([]GeoJSONMedia)
			if len(deferredMedia) != 0 {
				t.Errorf("expected empty media array in asset deferred mode, got %d items", len(deferredMedia))
			}
		}
	}
}

func TestExportJourneyFolderGeoJSON_IzuFixture(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "publication", "journeys", "izu-trip-2026-08-01")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("Izu fixture not found at %s", fixtureDir)
	}

	col, err := ExportJourneyFolderGeoJSON(fixtureDir, ExportOptions{
		Generator:   "felicia-cli export --journey izu-trip-2026-08-01",
		DeferAssets: true,
	})
	if err != nil {
		t.Fatalf("ExportJourneyFolderGeoJSON on Izu fixture failed: %v", err)
	}

	if col.Schema != GeoJSONSchemaV1 {
		t.Errorf("schema = %q, want %q", col.Schema, GeoJSONSchemaV1)
	}
	if col.Properties.Slug != "izu-trip-2026-08-01" {
		t.Errorf("slug = %q, want %q", col.Properties.Slug, "izu-trip-2026-08-01")
	}

	stopCount := 0
	mementoCount := 0
	hasRoute := false

	stopIDs := make(map[string]bool)

	for _, f := range col.Features {
		role := f.Properties["role"]
		switch role {
		case "route":
			hasRoute = true
			geom := f.Geometry.(map[string]any)
			coords := geom["coordinates"].([][]float64)
			for _, pt := range coords {
				assertCoordinateDecimals(t, pt[0], pt[1])
			}
		case "stop":
			stopCount++
			stopIDs[f.ID] = true
			geom := f.Geometry.(map[string]any)
			coords := geom["coordinates"].([]float64)
			assertCoordinateDecimals(t, coords[0], coords[1])
		case "memento":
			mementoCount++
			stopKey, ok := f.Properties["stop_key"].(string)
			if !ok || stopKey == "" {
				t.Errorf("memento %s missing stop_key", f.ID)
			}
			if !stopIDs[stopKey] {
				t.Errorf("memento %s references non-existent stop %q", f.ID, stopKey)
			}
			geom := f.Geometry.(map[string]any)
			coords := geom["coordinates"].([]float64)
			assertCoordinateDecimals(t, coords[0], coords[1])
		}
	}

	if !hasRoute {
		t.Error("expected route feature in exported Izu fixture")
	}
	if stopCount != 4 {
		t.Errorf("stopCount = %d, want 4", stopCount)
	}
	if mementoCount != 3 {
		t.Errorf("mementoCount = %d, want 3", mementoCount)
	}

	// Verify JSON serialization
	outBytes, err := MarshalGeoJSON(col)
	if err != nil {
		t.Fatalf("MarshalGeoJSON failed: %v", err)
	}
	if len(outBytes) == 0 {
		t.Fatal("empty serialized output")
	}
}

func assertCoordinateDecimals(t *testing.T, lon, lat float64) {
	t.Helper()
	// Capped at 4 decimals: math.Round(v*10000) / 10000
	scaledLon := lon * 10000.0
	if math.Abs(scaledLon-math.Round(scaledLon)) > 1e-6 {
		t.Errorf("lon %f exceeds 4 decimal places", lon)
	}
	scaledLat := lat * 10000.0
	if math.Abs(scaledLat-math.Round(scaledLat)) > 1e-6 {
		t.Errorf("lat %f exceeds 4 decimal places", lat)
	}
}
