package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTimelineSourceReadsConcatenatedVisitsAndSemanticSegments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Timeline.json")
	data := `{"placeVisit":{"duration":{"startTimestamp":"2026-04-01T09:00:00Z","endTimestamp":"2026-04-01T10:00:00Z"},"location":{"latitudeE7":356765000,"longitudeE7":1397440000,"placeId":"place-1","name":"明治神宮"}}}` +
		`{"semanticSegments":[{"startTime":"2026-04-02T09:00:00Z","endTime":"2026-04-02T10:00:00Z","placeVisit":{"location":{"latitudeE7":356765000,"longitudeE7":1397440000,"placeId":"place-1","name":"明治神宮"}}},{"activity":{"topCandidate":{"type":"WALKING"}}}]}` +
		`{"timelineObjects":[{"placeVisit":{"duration":{"startTimestampMs":"1775120400000","endTimestampMs":"1775124000000"},"location":{"latitudeE7":356765000,"longitudeE7":1397440000,"placeId":"place-2","name":"Takeout visit"}}},{"activitySegment":{}}]}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	visits, err := NewTimelineSource(path).FetchVisits(context.Background(), time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(visits) != 3 {
		t.Fatalf("visits = %#v", visits)
	}
	if visits[0].Label != "明治神宮" || visits[0].Coord.Lon() != 139.744 || visits[0].Coord.Lat() != 35.6765 {
		t.Fatalf("visit = %#v", visits[0])
	}
	if visits[0].Provenance.Source.System != "google-timeline" || visits[0].Provenance.Source.ExternalID == visits[1].Provenance.Source.ExternalID || visits[2].Label != "Takeout visit" || visits[2].Arrive.UnixMilli() != 1775120400000 {
		t.Fatalf("identities = %#v, %#v", visits[0].Provenance, visits[1].Provenance)
	}
	from := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	visits, err = NewTimelineSource(path).FetchVisits(context.Background(), from, from.Add(24*time.Hour))
	if err != nil || len(visits) != 2 || visits[0].Label != "明治神宮" || visits[1].Label != "Takeout visit" {
		t.Fatalf("range visits = %#v, %v", visits, err)
	}
	visits, err = NewTimelineSource(path).FetchVisits(context.Background(), from, time.Time{})
	if err != nil || len(visits) != 2 {
		t.Fatalf("from-only visits = %#v, %v", visits, err)
	}
	visits, err = NewTimelineSource(path).FetchVisits(context.Background(), time.Time{}, from)
	if err != nil || len(visits) != 1 || visits[0].Label != "明治神宮" {
		t.Fatalf("to-only visits = %#v, %v", visits, err)
	}
}

func TestTimelineSourceRejectsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Timeline.json")
	for _, input := range []string{
		`{"placeVisit":{"duration":{"startTimestamp":"bad","endTimestamp":"2026-04-01T10:00:00Z"},"location":{"latitudeE7":0,"longitudeE7":0}}}`,
		`{"placeVisit":{"duration":{"startTimestamp":"2026-04-01T09:00:00Z","endTimestamp":"2026-04-01T10:00:00Z"},"location":{"latitudeE7":910000000,"longitudeE7":0}}}`,
		`{"placeVisit":{"duration":{"startTimestamp":"2026-04-01T09:00:00Z","endTimestamp":"2026-04-01T10:00:00Z"},"location":{"name":"Missing coordinates"}}}`,
	} {
		if err := os.WriteFile(path, []byte(input), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := NewTimelineSource(path).FetchVisits(context.Background(), time.Time{}, time.Time{}); err == nil {
			t.Fatalf("accepted invalid export %s", input)
		}
	}
}
