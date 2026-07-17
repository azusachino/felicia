package dawarich

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

// trackFeatureCollection is the minimal GeoJSON shape of GET /api/v1/tracks.
// Hand-decoded to avoid orb/geojson's heavy transitive bson dependency.
type trackFeatureCollection struct {
	Features []struct {
		Geometry struct {
			Type        string      `json:"type"`
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
		Properties struct {
			ID           json.Number `json:"id"`
			StartAt      any         `json:"start_at"`
			EndAt        any         `json:"end_at"`
			Distance     int         `json:"distance"`
			DominantMode string      `json:"dominant_mode"`
		} `json:"properties"`
	} `json:"features"`
}

// parseTracks maps the tracks FeatureCollection into normalized routes.
// Non-LineString features are skipped.
func parseTracks(body []byte) ([]domain.Route, error) {
	var fc trackFeatureCollection
	if err := json.Unmarshal(body, &fc); err != nil {
		return nil, fmt.Errorf("decode tracks featurecollection: %w", err)
	}
	routes := make([]domain.Route, 0, len(fc.Features))
	for _, f := range fc.Features {
		if f.Geometry.Type != "LineString" {
			continue
		}
		line := make(orb.LineString, 0, len(f.Geometry.Coordinates))
		for _, c := range f.Geometry.Coordinates {
			if len(c) >= 2 {
				line = append(line, orb.Point{c[0], c[1]}) // [lng, lat]
			}
		}
		from, err := parseTime(f.Properties.StartAt)
		if err != nil {
			return nil, fmt.Errorf("track %s start_at: %w", f.Properties.ID, err)
		}
		to, err := parseTime(f.Properties.EndAt)
		if err != nil {
			return nil, fmt.Errorf("track %s end_at: %w", f.Properties.ID, err)
		}
		routes = append(routes, domain.Route{
			Line:      line,
			From:      from,
			To:        to,
			DistanceM: f.Properties.Distance,
			Mode:      f.Properties.DominantMode,
			SourceRef: "dawarich:track:" + f.Properties.ID.String(),
		})
	}
	return routes, nil
}

// visitDTO mirrors one element of the Dawarich GET /api/v1/visits array. The
// place carries coordinates + id only; the visit's name is the label.
type visitDTO struct {
	ID         json.Number `json:"id"`
	StartedAt  string      `json:"started_at"`
	EndedAt    string      `json:"ended_at"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
	Confidence float64     `json:"confidence"`
	Place      struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"place"`
}

// parseVisits maps the Dawarich visits array into normalized visits (derived
// places). Declined visits are dropped.
func parseVisits(body []byte) ([]domain.Visit, error) {
	var dtos []visitDTO
	if err := json.Unmarshal(body, &dtos); err != nil {
		return nil, fmt.Errorf("decode visits: %w", err)
	}
	visits := make([]domain.Visit, 0, len(dtos))
	for _, d := range dtos {
		if d.Status == "declined" {
			continue
		}
		arrive, err := parseTime(d.StartedAt)
		if err != nil {
			return nil, fmt.Errorf("visit %s started_at: %w", d.ID, err)
		}
		depart, err := parseTime(d.EndedAt)
		if err != nil {
			return nil, fmt.Errorf("visit %s ended_at: %w", d.ID, err)
		}
		visits = append(visits, domain.Visit{
			Coord:      orb.Point{d.Place.Longitude, d.Place.Latitude},
			Label:      d.Name,
			Arrive:     arrive,
			Depart:     depart,
			Confidence: d.Confidence,
			SourceRef:  "dawarich:visit:" + d.ID.String(),
		})
	}
	return visits, nil
}

// parseTime accepts either an RFC3339 string or a Unix-epoch number, since
// Dawarich renders timestamps both ways across endpoints. A nil/empty value
// yields the zero time.
func parseTime(v any) (time.Time, error) {
	switch t := v.(type) {
	case nil:
		return time.Time{}, nil
	case string:
		if t == "" {
			return time.Time{}, nil
		}
		return time.Parse(time.RFC3339, t)
	case float64:
		return time.Unix(int64(t), 0).UTC(), nil
	case json.Number:
		n, err := t.Int64()
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(n, 0).UTC(), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported time value %T", v)
	}
}
