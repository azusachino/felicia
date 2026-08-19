package intake

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
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

// Acceptance criterion 1 of issue #57: the bounds span every dated source,
// not just one of them.
func TestBuildPlanDerivesDateBoundsAcrossSources(t *testing.T) {
	tokyo := time.FixedZone("JST", 9*60*60)
	cases := []struct {
		name             string
		input            PlanInput
		wantStart        string
		wantEnd          string
		wantZeroBoundary bool
	}{
		{
			name: "route samples, visits and media together",
			input: PlanInput{
				Routes: []domain.Route{{Points: []domain.TrackPoint{
					{At: time.Date(2026, 4, 2, 8, 0, 0, 0, time.UTC)},
					{At: time.Date(2026, 4, 3, 8, 0, 0, 0, time.UTC)},
				}}},
				Visits: []domain.Visit{{
					Arrive: time.Date(2026, 4, 1, 22, 0, 0, 0, time.UTC),
					Depart: time.Date(2026, 4, 2, 1, 0, 0, 0, time.UTC),
				}},
				Media: []domain.MediaAsset{{At: time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)}},
			},
			wantStart: "2026-04-01", wantEnd: "2026-04-05",
		},
		{
			// Route.From/To carry the span even when a source hands us no
			// individual samples.
			name: "route span without samples",
			input: PlanInput{Routes: []domain.Route{{
				From: time.Date(2026, 6, 10, 0, 0, 0, 0, time.UTC),
				To:   time.Date(2026, 6, 12, 0, 0, 0, 0, time.UTC),
			}}},
			wantStart: "2026-06-10", wantEnd: "2026-06-12",
		},
		{
			// 08:00 in Tokyo is the previous day in UTC; the journey should
			// take the day the author actually lived, not the UTC one.
			name: "a local morning stays on its own calendar day",
			input: PlanInput{Media: []domain.MediaAsset{
				{At: time.Date(2026, 7, 1, 8, 0, 0, 0, tokyo)},
			}},
			wantStart: "2026-07-01", wantEnd: "2026-07-01",
		},
		{
			// Zero timestamps carry no information and must not drag the
			// bounds back to the zero year.
			name: "undated sources are ignored",
			input: PlanInput{
				Routes: []domain.Route{{Points: []domain.TrackPoint{{}, {At: time.Date(2026, 5, 4, 6, 0, 0, 0, time.UTC)}}}},
				Media:  []domain.MediaAsset{{}},
			},
			wantStart: "2026-05-04", wantEnd: "2026-05-04",
		},
		{
			name:             "no dated source at all leaves the bounds unset",
			input:            PlanInput{Routes: []domain.Route{{Points: []domain.TrackPoint{{}}}}},
			wantZeroBoundary: true,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			testCase.input.JourneyID = uuid.New()
			plan, err := BuildPlan(testCase.input, DefaultConfig())
			if err != nil {
				t.Fatal(err)
			}
			if testCase.wantZeroBoundary {
				if !plan.DateStart.IsZero() || !plan.DateEnd.IsZero() {
					t.Fatalf("bounds = %s..%s, want both unset", plan.DateStart, plan.DateEnd)
				}
				return
			}
			if got := plan.DateStart.Format("2006-01-02"); got != testCase.wantStart {
				t.Errorf("DateStart = %s, want %s", got, testCase.wantStart)
			}
			if got := plan.DateEnd.Format("2006-01-02"); got != testCase.wantEnd {
				t.Errorf("DateEnd = %s, want %s", got, testCase.wantEnd)
			}
		})
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
