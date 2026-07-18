package intake

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

func TestBuildPlanDerivesStopAndMementoFromTimestampedRouteAndMedia(t *testing.T) {
	journeyID := uuid.New()
	start := time.Date(2026, 4, 1, 1, 0, 0, 0, time.UTC)
	plan, err := BuildPlan(PlanInput{
		JourneyID: journeyID,
		Routes: []domain.Route{{Points: []domain.TrackPoint{
			{Coord: orb.Point{135.8430, 34.6851}, At: start},
			{Coord: orb.Point{135.8432, 34.6852}, At: start.Add(30 * time.Minute)},
			{Coord: orb.Point{135.8431, 34.6852}, At: start.Add(60 * time.Minute)},
		}}},
		Media: []domain.MediaAsset{{
			ID: "asset-1", At: start.Add(45 * time.Minute),
			Coord: sourcePoint(135.8431, 34.6852), SourceRef: "immich:asset-1",
		}},
	}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Stops) != 1 || len(plan.Mementos) != 1 {
		t.Fatalf("plan stops/mementos = %d/%d, want 1/1", len(plan.Stops), len(plan.Mementos))
	}
	if plan.Mementos[0].StopKey != plan.Stops[0].Identity.Key {
		t.Fatalf("memento stop key = %q, stop key = %q", plan.Mementos[0].StopKey, plan.Stops[0].Identity.Key)
	}
	if len(plan.Stops[0].Evidence) != 2 {
		t.Fatalf("evidence count = %d, want route/visit plus media evidence", len(plan.Stops[0].Evidence))
	}
	if plan.Stops[0].Evidence[0].Kind != domain.EvidenceRoute {
		t.Fatalf("derived evidence kind = %q, want route", plan.Stops[0].Evidence[0].Kind)
	}
}

func TestBuildPlanUsesSuppliedVisitsAndDoesNotRequireTrackPoints(t *testing.T) {
	visit := domain.Visit{
		Coord:      orb.Point{135.5, 34.7},
		Label:      "Kobe",
		Arrive:     time.Date(2026, 4, 2, 1, 0, 0, 0, time.UTC),
		Depart:     time.Date(2026, 4, 2, 3, 0, 0, 0, time.UTC),
		Confidence: 0.9,
		SourceRef:  "dawarich:visit-1",
	}
	plan, err := BuildPlan(PlanInput{JourneyID: uuid.New(), Visits: []domain.Visit{visit}}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Stops) != 1 || plan.Stops[0].Label != "Kobe" || len(plan.Issues) != 0 {
		t.Fatalf("plan = %#v, want supplied visit without derivation issue", plan)
	}
}

func TestBuildPlanReportsMissingTimestampedPoints(t *testing.T) {
	plan, err := BuildPlan(PlanInput{JourneyID: uuid.New(), Routes: []domain.Route{{Line: orb.LineString{{135.5, 34.7}, {135.6, 34.8}}}}}, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Stops) != 0 || !hasIssue(plan.Issues, "visit_derivation_unavailable") {
		t.Fatalf("plan = %#v, want derivation warning and no invented stop", plan)
	}
}

func sourcePoint(longitude, latitude float64) *orb.Point {
	point := orb.Point{longitude, latitude}
	return &point
}

func hasIssue(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
