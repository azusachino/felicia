package intake

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

func TestPlanOfflineTimezonesAndNoGPSPhotoFallback(t *testing.T) {
	at := time.Date(2026, 7, 1, 23, 30, 0, 0, time.UTC)
	input := PlanInput{
		JourneyID: uuid.New(),
		Visits: []domain.Visit{
			{Coord: orb.Point{139.6917, 35.6895}, Arrive: at, Depart: at.Add(time.Hour)},
			{Coord: orb.Point{-74.006, 40.7128}, Arrive: at.Add(24 * time.Hour), Depart: at.Add(25 * time.Hour)},
		},
		Media: []domain.MediaAsset{
			{ID: "tokyo-gps", At: at, Coord: sourcePoint(139.6917, 35.6895)},
			{ID: "no-gps", At: at},
			{ID: "new-york", At: at.Add(24 * time.Hour), Coord: sourcePoint(-74.006, 40.7128)},
		},
	}
	plan, err := BuildPlan(input, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Mementos) != 2 || plan.Mementos[0].OccurredTZ != "Asia/Tokyo" || plan.Mementos[1].OccurredTZ != "America/New_York" {
		t.Fatalf("candidate zones: %+v", plan.Mementos)
	}
	for _, asset := range plan.Mementos[0].Media {
		if asset.At.Location().String() != "Asia/Tokyo" || !asset.At.Equal(at) {
			t.Fatalf("photo fallback/instant: %+v", asset)
		}
	}
	if plan.DateStart.Format("2006-01-02") != "2026-07-02" {
		t.Fatalf("Tokyo calendar day: %s", plan.DateStart)
	}
	if input.Visits[0].Arrive.Location() != time.UTC || input.Media[0].At.Location() != time.UTC {
		t.Fatal("planner mutated source values")
	}
}

func TestNoGPSDateBoundsUseRouteJourneyZone(t *testing.T) {
	at := time.Date(2026, 7, 1, 23, 30, 0, 0, time.UTC)
	plan, err := BuildPlan(PlanInput{
		JourneyID: uuid.New(),
		Routes:    []domain.Route{{Line: orb.LineString{{139.6917, 35.6895}, {139.7, 35.7}}}},
		Media:     []domain.MediaAsset{{At: at}},
	}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if plan.DateStart.Format("2006-01-02") != "2026-07-02" {
		t.Fatalf("no-GPS photo day: %s", plan.DateStart)
	}
}
