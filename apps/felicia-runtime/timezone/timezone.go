// Package timezone supplies offline, editable timezone defaults for intake.
package timezone

import (
	"sync"
	"time"
	_ "time/tzdata" // Keep IANA validation usable without a host zoneinfo install.

	"github.com/paulmach/orb"
	tzf "github.com/ringsaturn/tzf/v2"
)

// The embedded finder is concurrent-safe and uses simplified boundaries (~111 m
// uncertainty). Reuse it rather than initializing one for every photo.
// https://github.com/ringsaturn/tzf/tree/v2.1.2#finders
var finder = sync.OnceValues(tzf.NewEmbeddedFinder)

// Lookup returns an IANA name for a valid longitude/latitude point, or empty
// when coordinates are absent/invalid. (0,0) is Felicia's missing-GPS sentinel.
func Lookup(point orb.Point) string {
	if point == (orb.Point{}) || !(point[0] >= -180 && point[0] <= 180 && point[1] >= -90 && point[1] <= 90) {
		return ""
	}
	f, err := finder()
	if err != nil {
		// A broken compiled-in dataset is not a recoverable lookup miss.
		panic(err)
	}
	return f.GetTimezoneName(point[0], point[1])
}

// Default preserves a supplied zone, otherwise resolves a point (or an edge's
// departure point), then uses the journey fallback, and finally UTC.
func Default(explicit string, geom orb.Geometry, fallback string) string {
	if explicit != "" {
		return explicit
	}
	var point orb.Point
	switch value := geom.(type) {
	case orb.Point:
		point = value
	case orb.LineString:
		if len(value) > 0 {
			point = value[0]
		}
	}
	if zone := Lookup(point); zone != "" {
		return zone
	}
	if fallback != "" {
		return fallback
	}
	return "UTC"
}

// Journey derives a deterministic fallback from the first usable route point.
func Journey(route orb.MultiLineString) string {
	for _, line := range route {
		for _, point := range line {
			if zone := Lookup(point); zone != "" {
				return zone
			}
		}
	}
	return ""
}

// Local preserves the instant while displaying it in the resolved zone. Empty
// zones leave source offsets alone; zero timestamps must stay zero.
func Local(at time.Time, zone string) time.Time {
	if at.IsZero() || zone == "" {
		return at
	}
	location, err := time.LoadLocation(zone)
	if err != nil {
		return at // Explicit invalid zones are rejected at the input boundary.
	}
	return at.In(location)
}
