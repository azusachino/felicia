package publication

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

// fullPrecisionLng and fullPrecisionLat carry far more than four decimal
// digits, standing in for a stored coordinate at whatever precision an
// importer happened to persist (GPX devices routinely emit 7+ digits).
// Both are near Tokyo purely so the fixture reads like a plausible route.
const (
	fullPrecisionLng = 139.691706123456789
	fullPrecisionLat = 35.689487987654321
)

// assertOnPublicPrecisionGrid fails the test unless v sits exactly on the
// publicCoordDecimals rounding grid, i.e. no residual precision beyond what
// the publication boundary promises survived the round trip.
func assertOnPublicPrecisionGrid(t *testing.T, label string, v float64) {
	t.Helper()
	const scale = 1e4 // matches publicCoordDecimals / roundCoord
	scaled := v * scale
	if diff := math.Abs(scaled - math.Round(scaled)); diff > 1e-6 {
		t.Errorf("%s = %v is not rounded to %d decimals (off-grid by %v)", label, v, publicCoordDecimals, diff)
	}
}

func TestGeometryRoundsCoordinatesToPublicPrecision(t *testing.T) {
	t.Run("Point", func(t *testing.T) {
		got := geometry(orb.Point{fullPrecisionLng, fullPrecisionLat})
		if got == nil {
			t.Fatal("geometry(Point) = nil")
		}
		coords, ok := got.Coordinates.([]float64)
		if !ok || len(coords) != 2 {
			t.Fatalf("Coordinates = %#v, want a [lng, lat] pair", got.Coordinates)
		}
		assertOnPublicPrecisionGrid(t, "Point.lng", coords[0])
		assertOnPublicPrecisionGrid(t, "Point.lat", coords[1])
		if coords[0] == fullPrecisionLng || coords[1] == fullPrecisionLat {
			t.Error("Point coordinates were passed through unrounded")
		}
	})

	t.Run("LineString", func(t *testing.T) {
		line := orb.LineString{
			{fullPrecisionLng, fullPrecisionLat},
			{fullPrecisionLng + 0.00001234567, fullPrecisionLat - 0.00009876543},
		}
		got := geometry(line)
		if got == nil {
			t.Fatal("geometry(LineString) = nil")
		}
		coords, ok := got.Coordinates.([][]float64)
		if !ok || len(coords) != 2 {
			t.Fatalf("Coordinates = %#v, want two [lng, lat] pairs", got.Coordinates)
		}
		for i, pair := range coords {
			assertOnPublicPrecisionGrid(t, "LineString point lng", pair[0])
			assertOnPublicPrecisionGrid(t, "LineString point lat", pair[1])
			if pair[0] == line[i].X() || pair[1] == line[i].Y() {
				t.Errorf("LineString point %d was passed through unrounded", i)
			}
		}
	})

	t.Run("MultiLineString", func(t *testing.T) {
		multi := orb.MultiLineString{
			orb.LineString{{fullPrecisionLng, fullPrecisionLat}, {fullPrecisionLng + 0.0001, fullPrecisionLat + 0.0001}},
			orb.LineString{{fullPrecisionLng + 1, fullPrecisionLat + 1}},
		}
		got := geometry(multi)
		if got == nil {
			t.Fatal("geometry(MultiLineString) = nil")
		}
		coords, ok := got.Coordinates.([][][]float64)
		if !ok || len(coords) != 2 {
			t.Fatalf("Coordinates = %#v, want two lines", got.Coordinates)
		}
		for li, line := range coords {
			for pi, pair := range line {
				assertOnPublicPrecisionGrid(t, "MultiLineString point lng", pair[0])
				assertOnPublicPrecisionGrid(t, "MultiLineString point lat", pair[1])
				original := multi[li][pi]
				if pair[0] == original.X() || pair[1] == original.Y() {
					t.Errorf("MultiLineString line %d point %d was passed through unrounded", li, pi)
				}
			}
		}
	})
}

// TestNewStaticJourneyRoundsFullPrecisionStoredRoute asserts the guarantee
// holds against a full-precision *stored* route — i.e. domain.Journey as it
// would come back from the repository, not a fixture already massaged to
// four decimals. NewStaticJourney is the sole place a journey's route is
// projected for both the static compiler and the live /api/v1 handlers, so
// this covers both surfaces regardless of which importer wrote GPSRoute.
func TestNewStaticJourneyRoundsFullPrecisionStoredRoute(t *testing.T) {
	journey := &domain.Journey{
		ID: uuid.New(), JournalID: uuid.New(), Slug: "tokyo", Title: "Tokyo", Place: "Tokyo",
		DateStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		GPSRoute: orb.MultiLineString{
			orb.LineString{
				{fullPrecisionLng, fullPrecisionLat},
				{fullPrecisionLng + 0.00042, fullPrecisionLat - 0.00017},
			},
		},
	}

	projected := NewStaticJourney(journey)
	if projected.GPSRoute == nil {
		t.Fatal("GPSRoute = nil, want the rounded route")
	}
	coords, ok := projected.GPSRoute.Coordinates.([][][]float64)
	if !ok || len(coords) != 1 || len(coords[0]) != 2 {
		t.Fatalf("Coordinates = %#v, want one line of two points", projected.GPSRoute.Coordinates)
	}
	for _, pair := range coords[0] {
		assertOnPublicPrecisionGrid(t, "stored route point lng", pair[0])
		assertOnPublicPrecisionGrid(t, "stored route point lat", pair[1])
	}
	if coords[0][0][0] == fullPrecisionLng {
		t.Error("stored route's full-precision longitude reached the projection unrounded")
	}

	// The JSON on the wire must not contain the full-precision literal either
	// — belt-and-braces against a future change that rounds the in-memory
	// value but re-serializes from the original.
	raw, err := json.Marshal(projected)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if bytesContainsFloat(raw, fullPrecisionLng) {
		t.Errorf("journey JSON still contains the full-precision longitude: %s", raw)
	}
}

// TestCompileRoundsStoredRouteAndMementoGeometry runs the full compiler path
// (not just the projection helpers) against a journey and memento whose
// stored geometry is full precision, and checks the artifact JSON that would
// actually be shipped never carries it.
func TestCompileRoundsStoredRouteAndMementoGeometry(t *testing.T) {
	journalID := uuid.New()
	journey := &domain.Journey{
		ID: uuid.New(), JournalID: journalID, Slug: "tokyo", Title: "Tokyo", Place: "Tokyo",
		DateStart: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC),
		GPSRoute: orb.MultiLineString{
			orb.LineString{{fullPrecisionLng, fullPrecisionLat}, {fullPrecisionLng + 0.0005, fullPrecisionLat + 0.0005}},
		},
	}
	memento := &domain.Memento{
		ID: uuid.New(), JourneyID: journey.ID, Kind: "goods", Seq: 1,
		OccurredAt: time.Date(2026, 7, 2, 12, 0, 0, 0, time.UTC), OccurredTZ: "Asia/Tokyo",
		Geom: orb.Point{fullPrecisionLng, fullPrecisionLat}, Title: "stub", Place: "Akihabara",
		State: domain.MementoPublished,
	}
	read := &fakeReadModel{
		journeys: []*domain.Journey{journey},
		mementos: map[uuid.UUID][]*domain.Memento{journey.ID: {memento}},
		photos:   map[uuid.UUID][]*domain.MementoPhoto{},
		journal:  &domain.Journal{ID: journalID},
	}
	writer := &memoryArtifactWriter{}

	_, err := StaticCompiler{}.Compile(context.Background(), Input{}, read, fakeMediaSource{}, writer)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	journeyJSON := writer.json["api/v1/journeys/"+journey.ID.String()+".json"]
	if len(journeyJSON) == 0 {
		t.Fatal("journey detail was not written")
	}
	if bytesContainsFloat(journeyJSON, fullPrecisionLng) || bytesContainsFloat(journeyJSON, fullPrecisionLat) {
		t.Errorf("compiled journey artifact still carries full-precision route coordinates: %s", journeyJSON)
	}

	mementosJSON := writer.json["api/v1/journeys/"+journey.ID.String()+"/mementos.json"]
	if len(mementosJSON) == 0 {
		t.Fatal("mementos were not written")
	}
	if bytesContainsFloat(mementosJSON, fullPrecisionLng) || bytesContainsFloat(mementosJSON, fullPrecisionLat) {
		t.Errorf("compiled memento artifact still carries full-precision geometry: %s", mementosJSON)
	}

	var decodedJourney StaticJourney
	if err := json.Unmarshal(journeyJSON, &decodedJourney); err != nil {
		t.Fatalf("unmarshal journey: %v", err)
	}
	coords := decodedJourney.GPSRoute.Coordinates.([]any)[0].([]any)[0].([]any)
	assertOnPublicPrecisionGrid(t, "compiled route point lng", coords[0].(float64))
	assertOnPublicPrecisionGrid(t, "compiled route point lat", coords[1].(float64))
}

// bytesContainsFloat reports whether raw's JSON text contains f serialized
// at full precision (Go's default float64 -> string conversion), which is
// how an accidental full-precision leak would actually show up on the wire.
func bytesContainsFloat(raw []byte, f float64) bool {
	full, err := json.Marshal(f)
	if err != nil {
		return false
	}
	return strings.Contains(string(raw), string(full))
}
