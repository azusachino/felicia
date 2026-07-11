package domain

import (
	"context"
	"time"

	"github.com/paulmach/orb"
)

// Ingest value objects — the normalized shapes every external source is mapped
// into at the edge (source-connectors.md). The core stays generic over these,
// never over a specific provider.

// Route is a normalized track segment: a polyline with the window it covers.
// Dawarich tracks map to this; the journey's gps_route is the union of routes.
type Route struct {
	Line      orb.LineString
	From      time.Time
	To        time.Time
	DistanceM int
	Mode      string // dominant transport mode, e.g. "walking", "car"
	SourceRef string // provenance, e.g. "dawarich:track:42"
}

// Visit is a normalized stay — a derived place (place-as-derived-visit, ADR
// 0005). Dawarich visits map to this; a memento anchors to the nearest visit.
type Visit struct {
	Coord      orb.Point // [lng, lat]
	Label      string
	Arrive     time.Time
	Depart     time.Time
	Confidence float64
	SourceRef  string // "dawarich:visit:7"
}

// PhotoAsset is a normalized photo. Immich assets map to this; Coord is nil when
// the photo carries no GPS EXIF (filled from the track by timestamp, or on drop).
type PhotoAsset struct {
	ID        string
	At        time.Time
	Coord     *orb.Point
	Checksum  string
	SourceRef string // "immich:asset:{id}"
}

// TrackSource yields routes and visits for a time range. Dawarich implements it.
type TrackSource interface {
	FetchRoutes(ctx context.Context, from, to time.Time) ([]Route, error)
	FetchVisits(ctx context.Context, from, to time.Time) ([]Visit, error)
}

// PhotoSource yields photo assets for a time range. Immich implements it.
type PhotoSource interface {
	FetchAssets(ctx context.Context, from, to time.Time) ([]PhotoAsset, error)
}
