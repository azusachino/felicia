package publication

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

func TestNewJourneyListItem(t *testing.T) {
	journey := &domain.Journey{Slug: "japan-spring-2026", Title: "日本春旅 2026"}
	mementos := []*domain.Memento{
		{Place: "東京駅", Geom: orb.Point{139.7671, 35.6812}},
		{Place: "東京駅", Geom: orb.Point{139.7003, 35.6895}},
		{Place: "東海道", Geom: orb.LineString{{139.7671, 35.6812}, {135.7583, 34.9859}}},
		{Place: "geometry missing"},
		{Place: "京都", Geom: orb.Point{135.7583, 34.9859}},
		{Place: "大阪", Geom: orb.Point{135.5013, 34.6687}},
	}

	got := NewJourneyListItem(journey, mementos)
	want := JourneyListItem{
		Slug: "japan-spring-2026", Title: "日本春旅 2026", MementoCount: 6,
		RepresentativeDots: []RepresentativeDot{
			{Coord: []float64{139.7671, 35.6812}, Label: "東京駅"},
			{Coord: []float64{139.7671, 35.6812}, Label: "東海道"},
			{Coord: []float64{135.7583, 34.9859}, Label: "京都"},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("NewJourneyListItem mismatch (-want +got):\n%s", diff)
	}
}

func TestNewJourneyListItemJSONShape(t *testing.T) {
	data, err := json.Marshal(NewJourneyListItem(
		&domain.Journey{Slug: "japan", Title: "日本"},
		[]*domain.Memento{{Place: "東京", Geom: orb.Point{139.7, 35.6}}},
	))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":"00000000-0000-0000-0000-000000000000","slug":"japan","title":"日本","memento_count":1,"representative_dots":[{"coord":[139.7,35.6],"label":"東京"}]}`
	if string(data) != want {
		t.Errorf("unexpected JSON shape:\n got: %s\nwant: %s", data, want)
	}
}
