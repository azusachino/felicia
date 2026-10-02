package timezone

import (
	"math"
	"testing"
	"time"

	"github.com/paulmach/orb"
)

func TestOfflineDefaults(t *testing.T) {
	for _, test := range []struct {
		point orb.Point
		want  string
	}{
		{orb.Point{139.6917, 35.6895}, "Asia/Tokyo"},
		{orb.Point{-74.006, 40.7128}, "America/New_York"},
		{orb.Point{151.2093, -33.8688}, "Australia/Sydney"},
		{orb.Point{}, ""},
		{orb.Point{181, 40}, ""},
		{orb.Point{139, 91}, ""},
		{orb.Point{math.NaN(), 35}, ""},
		{orb.Point{139, math.Inf(1)}, ""},
	} {
		if got := Lookup(test.point); got != test.want {
			t.Errorf("Lookup(%v) = %q, want %q", test.point, got, test.want)
		}
	}
	line := orb.LineString{{-74.006, 40.7128}, {139.6917, 35.6895}}
	if got := Default("", line, "UTC"); got != "America/New_York" {
		t.Fatal(got)
	}
	if got := Default("UTC", line, "Asia/Tokyo"); got != "UTC" {
		t.Fatal("explicit UTC was replaced:", got)
	}
	if got := Default("", nil, "Asia/Tokyo"); got != "Asia/Tokyo" {
		t.Fatal(got)
	}
	if got := Default("", nil, ""); got != "UTC" {
		t.Fatal(got)
	}
}

func TestLocalPreservesInstantAndDaylightSaving(t *testing.T) {
	for _, test := range []struct {
		at   string
		want int
	}{{"2026-01-01T12:00:00Z", -5 * 3600}, {"2026-07-01T12:00:00Z", -4 * 3600}} {
		at, err := time.Parse(time.RFC3339, test.at)
		if err != nil {
			t.Fatal(err)
		}
		local := Local(at, "America/New_York")
		_, offset := local.Zone()
		if !local.Equal(at) || offset != test.want {
			t.Fatalf("Local(%s) = %s", at, local)
		}
	}
	if !Local(time.Time{}, "Asia/Tokyo").IsZero() {
		t.Fatal("localized zero timestamp")
	}
}
