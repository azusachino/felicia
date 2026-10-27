package publication

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/simplify"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// GeoJSONSchemaV1 is the official schema identifier for published journey GeoJSON exports.
const GeoJSONSchemaV1 = "felicia.journey.geojson.v1"

// GeoJSONFeatureCollection is the top-level GeoJSON export container.
type GeoJSONFeatureCollection struct {
	Type       string                 `json:"type"`
	Schema     string                 `json:"schema"`
	Generator  string                 `json:"generator"`
	Properties GeoJSONJourneyProps    `json:"properties"`
	Features   []GeoJSONExportFeature `json:"features"`
}

// GeoJSONJourneyProps holds journey-level metadata.
type GeoJSONJourneyProps struct {
	Slug      string       `json:"slug"`
	Title     string       `json:"title"`
	Place     string       `json:"place"`
	Country   string       `json:"country"`
	Region    string       `json:"region"`
	DateStart string       `json:"date_start"`
	DateEnd   string       `json:"date_end"`
	Days      []GeoJSONDay `json:"days,omitempty"`
}

// GeoJSONDay captures a structured daily segment and reflection.
type GeoJSONDay struct {
	Day         int    `json:"day"`
	Date        string `json:"date"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
}

// GeoJSONExportFeature represents a Feature within the collection (role: route, stop, or memento).
type GeoJSONExportFeature struct {
	Type       string         `json:"type"`
	ID         string         `json:"id,omitempty"`
	Properties map[string]any `json:"properties"`
	Geometry   any            `json:"geometry"`
}

// GeoJSONMedia represents an external Immich media item (Decision D11, D12).
type GeoJSONMedia struct {
	ImmichAssetID string `json:"immich_asset_id"`
	Caption       string `json:"caption,omitempty"`
	Seq           int    `json:"seq,omitempty"`
}

// ExportOptions controls export filtering and sanitization options.
type ExportOptions struct {
	Generator   string
	DeferAssets bool
	Days        []GeoJSONDay
}

// ExtractImmichAssetID parses an Immich asset ID from source_ref strings like
// "immich:asset:<uuid>", "immich:<uuid>", or "immich/<uuid>".
func ExtractImmichAssetID(sourceRef string) (string, bool) {
	s := strings.TrimSpace(sourceRef)
	if s == "" {
		return "", false
	}
	for _, prefix := range []string{"immich:asset:", "immich:", "immich/"} {
		if strings.HasPrefix(s, prefix) {
			id := strings.TrimSpace(strings.TrimPrefix(s, prefix))
			if id != "" {
				return id, true
			}
		}
	}
	return "", false
}

// ExportJourneyGeoJSON builds a published GeoJSON FeatureCollection from a ReadModel (e.g. SQLite).
func ExportJourneyGeoJSON(ctx context.Context, slug string, read ReadModel, opts ExportOptions) (*GeoJSONFeatureCollection, error) {
	journeys, err := read.ListJourneys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list journeys: %w", err)
	}

	var target *domain.Journey
	for _, j := range journeys {
		if j.Slug == slug || j.ID.String() == slug {
			target = j
			break
		}
	}
	if target == nil {
		return nil, fmt.Errorf("journey not found: %s", slug)
	}

	mementos, err := read.ListMementosByJourney(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("list mementos for %s: %w", target.ID, err)
	}
	published := PublishedMementos(mementos)

	var candidates []*domain.StopCandidate
	if reader, ok := read.(StopCandidateReader); ok {
		candidates, err = reader.ListStopCandidatesByJourney(ctx, target.ID)
		if err != nil {
			return nil, fmt.Errorf("list stop candidates for %s: %w", target.ID, err)
		}
	}

	return buildExport(target, published, candidates, readPhotos(ctx, read, published), opts)
}

func readPhotos(ctx context.Context, read ReadModel, mementos []*domain.Memento) map[uuid.UUID][]*domain.MementoPhoto {
	result := make(map[uuid.UUID][]*domain.MementoPhoto, len(mementos))
	for _, m := range mementos {
		photos, err := read.ListPhotosByMemento(ctx, m.ID)
		if err == nil && len(photos) > 0 {
			SortPhotos(photos)
			result[m.ID] = photos
		}
	}
	return result
}

func buildExport(
	journey *domain.Journey,
	publishedMementos []*domain.Memento,
	stopCandidates []*domain.StopCandidate,
	photosByMemento map[uuid.UUID][]*domain.MementoPhoto,
	opts ExportOptions,
) (*GeoJSONFeatureCollection, error) {
	gen := opts.Generator
	if gen == "" {
		gen = fmt.Sprintf("felicia-cli export --journey %s", journey.Slug)
	}

	country := ""
	if journey.Country != nil {
		country = *journey.Country
	}
	region := ""
	if journey.Region != nil {
		region = *journey.Region
	}

	collection := &GeoJSONFeatureCollection{
		Type:      "FeatureCollection",
		Schema:    GeoJSONSchemaV1,
		Generator: gen,
		Properties: GeoJSONJourneyProps{
			Slug:      journey.Slug,
			Title:     journey.Title,
			Place:     journey.Place,
			Country:   country,
			Region:    region,
			DateStart: journey.DateStart.Format("2006-01-02"),
			DateEnd:   journey.DateEnd.Format("2006-01-02"),
			Days:      opts.Days,
		},
		Features: make([]GeoJSONExportFeature, 0),
	}

	// 1. Route Feature (LineString, trimmed by 200m at start/end and rounded to 4 decimals)
	if len(journey.GPSRoute) > 0 {
		routeCoords := cleanMultiLineString(journey.GPSRoute)
		if len(routeCoords) > 0 {
			collection.Features = append(collection.Features, GeoJSONExportFeature{
				Type: "Feature",
				Properties: map[string]any{
					"role": "route",
				},
				Geometry: map[string]any{
					"type":        "LineString",
					"coordinates": routeCoords,
				},
			})
		}
	}

	// 2. Stop Features
	selectedStops := make(map[string]*domain.StopCandidate)
	for _, candidate := range stopCandidates {
		// Only include stops that are kept / selected or have a non-empty label
		if candidate.State == domain.CandidateKept || candidate.Label != "" {
			key := candidate.Identity.Key
			if key == "" {
				key = candidate.ID.String()
			}
			selectedStops[key] = candidate
		}
	}

	// Also identify stops referenced by published mementos
	mementoStopKeys := make(map[string]bool)
	for _, m := range publishedMementos {
		key := extractStopKey(m)
		if key != "" {
			mementoStopKeys[key] = true
		}
	}

	// Emit stop features in stable order
	stopKeys := make([]string, 0, len(selectedStops))
	for k := range selectedStops {
		stopKeys = append(stopKeys, k)
	}
	sort.Strings(stopKeys)

	for _, k := range stopKeys {
		s := selectedStops[k]
		// Omit stops without labels unless explicitly attached to a memento
		if s.Label == "" && !mementoStopKeys[k] {
			continue
		}
		coord := []float64{roundCoord(s.Coord.X()), roundCoord(s.Coord.Y())}
		arriveStr := ""
		if !s.Arrive.IsZero() {
			arriveStr = s.Arrive.Format(time.RFC3339)
		}
		departStr := ""
		if !s.Depart.IsZero() {
			departStr = s.Depart.Format(time.RFC3339)
		}
		collection.Features = append(collection.Features, GeoJSONExportFeature{
			Type: "Feature",
			ID:   k,
			Properties: map[string]any{
				"role":   "stop",
				"label":  s.Label,
				"arrive": arriveStr,
				"depart": departStr,
			},
			Geometry: map[string]any{
				"type":        "Point",
				"coordinates": coord,
			},
		})
	}

	// 3. Memento Features
	for _, m := range publishedMementos {
		var coord []float64
		if g, ok := m.Geom.(orb.Point); ok {
			coord = []float64{roundCoord(g.X()), roundCoord(g.Y())}
		}
		if len(coord) != 2 {
			continue
		}

		stopKey := extractStopKey(m)
		if stopKey == "" {
			// fallback: find closest selected stop
			stopKey = findClosestStopKey(m.Geom, selectedStops)
		}

		var essayStr string
		if m.Essay != nil {
			essayStr = *m.Essay
		}

		mediaList := make([]GeoJSONMedia, 0)
		if !opts.DeferAssets {
			photos := photosByMemento[m.ID]
			for _, p := range photos {
				caption := ""
				if p.Caption != nil {
					caption = *p.Caption
				}
				sourceRef := ""
				if p.SourceRef != nil {
					sourceRef = *p.SourceRef
				}
				assetID, ok := ExtractImmichAssetID(sourceRef)
				if !ok {
					return nil, fmt.Errorf("memento %s (%s): photo %s has no immich asset id in source_ref %q (Decision D12)", m.ID, m.Title, p.ID, sourceRef)
				}
				mediaList = append(mediaList, GeoJSONMedia{
					ImmichAssetID: assetID,
					Caption:       caption,
					Seq:           p.Seq,
				})
			}
		}

		props := map[string]any{
			"role":        "memento",
			"stop_key":    stopKey,
			"seq":         m.Seq,
			"kind":        m.Kind,
			"occurred_at": m.OccurredAt.Format(time.RFC3339),
			"occurred_tz": m.OccurredTZ,
			"title":       m.Title,
			"place":       m.Place,
			"essay":       essayStr,
			"kind_data":   m.KindData,
			"media":       mediaList,
		}

		collection.Features = append(collection.Features, GeoJSONExportFeature{
			Type:       "Feature",
			ID:         m.ID.String(),
			Properties: props,
			Geometry: map[string]any{
				"type":        "Point",
				"coordinates": coord,
			},
		})
	}

	return collection, nil
}

func extractStopKey(m *domain.Memento) string {
	if m.SourceRef != nil && strings.HasPrefix(*m.SourceRef, "stop:") {
		return strings.TrimPrefix(*m.SourceRef, "stop:")
	}
	if len(m.KindData) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(m.KindData, &raw); err == nil {
			if v, ok := raw["stop_key"].(string); ok && v != "" {
				return v
			}
		}
	}
	return ""
}

func findClosestStopKey(geom orb.Geometry, stops map[string]*domain.StopCandidate) string {
	point, ok := geom.(orb.Point)
	if !ok || len(stops) == 0 {
		return ""
	}
	bestKey := ""
	bestDist := 1e12
	for key, s := range stops {
		dx := s.Coord.X() - point.X()
		dy := s.Coord.Y() - point.Y()
		dist := dx*dx + dy*dy
		if dist < bestDist {
			bestDist = dist
			bestKey = key
		}
	}
	return bestKey
}

// cleanMultiLineString trims 200m ground distance at start/end, simplifies collinear points,
// rounds each ordinate to 4 decimal places, and skips identical consecutive points.
func cleanMultiLineString(value orb.MultiLineString) [][]float64 {
	if len(value) == 0 {
		return nil
	}
	result := make([][]float64, 0)
	for i, line := range value {
		trimmed := line
		if i == 0 {
			trimmed = trimLineStart(trimmed)
		}
		if i == len(value)-1 {
			trimmed = trimLineEnd(trimmed)
		}
		if len(trimmed) == 0 {
			continue
		}
		// Douglas-Peucker simplification with 0.0001 degrees (~11m ground distance)
		simplified := simplify.DouglasPeucker(0.0001).LineString(trimmed)
		for _, pt := range simplified {
			rx := roundCoord(pt.X())
			ry := roundCoord(pt.Y())
			// Skip duplicate consecutive points
			if len(result) > 0 {
				last := result[len(result)-1]
				if last[0] == rx && last[1] == ry {
					continue
				}
			}
			result = append(result, []float64{rx, ry})
		}
	}
	return result
}

// --- Folder-based Export (e.g. publication/journeys/<slug>) ---

type folderJourneyJSON struct {
	Slug      string       `json:"slug"`
	Title     string       `json:"title"`
	Place     string       `json:"place"`
	Country   string       `json:"country"`
	Region    string       `json:"region"`
	DateStart string       `json:"date_start"`
	DateEnd   string       `json:"date_end"`
	Days      []GeoJSONDay `json:"days,omitempty"`
}

type folderStopsJSON struct {
	Stops []struct {
		CandidateKey string    `json:"candidate_key"`
		Key          string    `json:"key"`
		Selected     bool      `json:"selected"`
		Label        string    `json:"label"`
		Coord        []float64 `json:"coord"`
		Arrive       string    `json:"arrive"`
		Depart       string    `json:"depart"`
	} `json:"stops"`
}

type folderMementosJSON struct {
	Mementos []struct {
		ID         string          `json:"id"`
		StopKey    string          `json:"stop_key"`
		Seq        int             `json:"seq"`
		Kind       string          `json:"kind"`
		OccurredAt string          `json:"occurred_at"`
		OccurredTZ string          `json:"occurred_tz"`
		Title      string          `json:"title"`
		Place      string          `json:"place"`
		Geom       []float64       `json:"geom"`
		State      string          `json:"state"`
		KindData   json.RawMessage `json:"kind_data"`
		Essay      string          `json:"essay"`
		Media      []struct {
			Path          string `json:"path"`
			Caption       string `json:"caption"`
			ImmichAssetID string `json:"immich_asset_id"`
			SourceRef     string `json:"source_ref"`
			Seq           int    `json:"seq"`
		} `json:"media"`
	} `json:"mementos"`
}

type folderGPXFile struct {
	Tracks []struct {
		Segments []struct {
			Points []struct {
				Latitude  float64 `xml:"lat,attr"`
				Longitude float64 `xml:"lon,attr"`
			} `xml:"trkpt"`
		} `xml:"trkseg"`
	} `xml:"trk"`
}

// ExportJourneyFolderGeoJSON builds a published GeoJSON FeatureCollection from a journey directory.
func ExportJourneyFolderGeoJSON(dir string, opts ExportOptions) (*GeoJSONFeatureCollection, error) {
	journeyPath := filepath.Join(dir, "journey.json")
	journeyBytes, err := os.ReadFile(journeyPath)
	if err != nil {
		return nil, fmt.Errorf("read journey.json: %w", err)
	}
	var jData folderJourneyJSON
	if err := json.Unmarshal(journeyBytes, &jData); err != nil {
		return nil, fmt.Errorf("parse journey.json: %w", err)
	}

	days := opts.Days
	if len(days) == 0 {
		daysPath := filepath.Join(dir, "days.json")
		if daysBytes, err := os.ReadFile(daysPath); err == nil {
			_ = json.Unmarshal(daysBytes, &days)
		}
	}

	gen := opts.Generator
	if gen == "" {
		gen = fmt.Sprintf("felicia-cli export --journey %s", jData.Slug)
	}

	collection := &GeoJSONFeatureCollection{
		Type:      "FeatureCollection",
		Schema:    GeoJSONSchemaV1,
		Generator: gen,
		Properties: GeoJSONJourneyProps{
			Slug:      jData.Slug,
			Title:     jData.Title,
			Place:     jData.Place,
			Country:   jData.Country,
			Region:    jData.Region,
			DateStart: jData.DateStart,
			DateEnd:   jData.DateEnd,
			Days:      days,
		},
		Features: make([]GeoJSONExportFeature, 0),
	}

	// 1. Route GPX
	gpxPath := filepath.Join(dir, "route.gpx")
	if gpxBytes, err := os.ReadFile(gpxPath); err == nil {
		var gpx folderGPXFile
		if err := xml.Unmarshal(gpxBytes, &gpx); err == nil {
			var mls orb.MultiLineString
			for _, trk := range gpx.Tracks {
				for _, seg := range trk.Segments {
					if len(seg.Points) < 2 {
						continue
					}
					line := make(orb.LineString, 0, len(seg.Points))
					for _, p := range seg.Points {
						line = append(line, orb.Point{p.Longitude, p.Latitude})
					}
					mls = append(mls, line)
				}
			}
			coords := cleanMultiLineString(mls)
			if len(coords) > 0 {
				collection.Features = append(collection.Features, GeoJSONExportFeature{
					Type: "Feature",
					Properties: map[string]any{
						"role": "route",
					},
					Geometry: map[string]any{
						"type":        "LineString",
						"coordinates": coords,
					},
				})
			}
		}
	}

	// 2. Stops JSON
	stopsPath := filepath.Join(dir, "stops.json")
	if stopsBytes, err := os.ReadFile(stopsPath); err == nil {
		var sData folderStopsJSON
		if err := json.Unmarshal(stopsBytes, &sData); err == nil {
			for _, s := range sData.Stops {
				if !s.Selected {
					continue
				}
				key := s.CandidateKey
				if key == "" {
					key = s.Key
				}
				// Omit unlabelled transit clusters
				if s.Label == "" {
					continue
				}
				if len(s.Coord) == 2 {
					collection.Features = append(collection.Features, GeoJSONExportFeature{
						Type: "Feature",
						ID:   key,
						Properties: map[string]any{
							"role":   "stop",
							"label":  s.Label,
							"arrive": s.Arrive,
							"depart": s.Depart,
						},
						Geometry: map[string]any{
							"type":        "Point",
							"coordinates": []float64{roundCoord(s.Coord[0]), roundCoord(s.Coord[1])},
						},
					})
				}
			}
		}
	}

	// 3. Mementos JSON
	mementosPath := filepath.Join(dir, "mementos.json")
	if mementosBytes, err := os.ReadFile(mementosPath); err == nil {
		var mData folderMementosJSON
		if err := json.Unmarshal(mementosBytes, &mData); err == nil {
			// Sort by seq
			sort.SliceStable(mData.Mementos, func(i, j int) bool {
				return mData.Mementos[i].Seq < mData.Mementos[j].Seq
			})

			for _, m := range mData.Mementos {
				if m.State != "" && m.State != "published" {
					continue
				}
				if len(m.Geom) != 2 {
					continue
				}

				mediaList := make([]GeoJSONMedia, 0)
				if !opts.DeferAssets {
					for _, p := range m.Media {
						assetID := p.ImmichAssetID
						if assetID == "" {
							if id, ok := ExtractImmichAssetID(p.SourceRef); ok {
								assetID = id
							}
						}
						if assetID == "" {
							return nil, fmt.Errorf("memento %s (%s): media %q has no immich asset id (Decision D12)", m.ID, m.Title, p.Path)
						}
						mediaList = append(mediaList, GeoJSONMedia{
							ImmichAssetID: assetID,
							Caption:       p.Caption,
							Seq:           p.Seq,
						})
					}
				}

				collection.Features = append(collection.Features, GeoJSONExportFeature{
					Type: "Feature",
					ID:   m.ID,
					Properties: map[string]any{
						"role":        "memento",
						"stop_key":    m.StopKey,
						"seq":         m.Seq,
						"kind":        m.Kind,
						"occurred_at": m.OccurredAt,
						"occurred_tz": m.OccurredTZ,
						"title":       m.Title,
						"place":       m.Place,
						"essay":       m.Essay,
						"kind_data":   m.KindData,
						"media":       mediaList,
					},
					Geometry: map[string]any{
						"type":        "Point",
						"coordinates": []float64{roundCoord(m.Geom[0]), roundCoord(m.Geom[1])},
					},
				})
			}
		}
	}

	return collection, nil
}

// MarshalGeoJSON serializes a GeoJSONFeatureCollection with 2-space indentation.
func MarshalGeoJSON(col *GeoJSONFeatureCollection) ([]byte, error) {
	return json.MarshalIndent(col, "", "  ")
}

// Ensure unused import warning is avoided
var _ = slices.Contains[[]string, string]
var _ = errors.New
