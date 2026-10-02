package local

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// TimelineSource reads Google Maps Timeline JSON exports as semantic visits.
// The export may contain concatenated JSON documents rather than one array.
type TimelineSource struct{ path string }

var _ domain.VisitSource = (*TimelineSource)(nil)

// NewTimelineSource creates an offline Google Timeline visit source.
func NewTimelineSource(path string) *TimelineSource { return &TimelineSource{path: path} }

type timelineDocument struct {
	SemanticSegments []timelineSegment   `json:"semanticSegments"`
	TimelineObjects  []timelineSegment   `json:"timelineObjects"`
	PlaceVisit       *timelinePlaceVisit `json:"placeVisit"`
	StartTime        string              `json:"startTime"`
	EndTime          string              `json:"endTime"`
}
type timelineSegment struct {
	StartTime  string              `json:"startTime"`
	EndTime    string              `json:"endTime"`
	PlaceVisit *timelinePlaceVisit `json:"placeVisit"`
}
type timelinePlaceVisit struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Duration  struct {
		Start       string `json:"startTimestamp"`
		End         string `json:"endTimestamp"`
		StartMillis string `json:"startTimestampMs"`
		EndMillis   string `json:"endTimestampMs"`
	} `json:"duration"`
	Location struct {
		LatitudeE7  *int64 `json:"latitudeE7"`
		LongitudeE7 *int64 `json:"longitudeE7"`
		PlaceID     string `json:"placeId"`
		Name        string `json:"name"`
	} `json:"location"`
}

// FetchVisits returns named place visits overlapping [from,to]. Activity
// segments are intentionally not converted into synthetic routes.
func (s *TimelineSource) FetchVisits(ctx context.Context, from, to time.Time) ([]domain.Visit, error) {
	if s == nil || s.path == "" {
		return nil, errors.New("timeline path is required")
	}
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open timeline %s: %w", s.path, err)
	}
	defer func() { _ = file.Close() }()
	decoder := json.NewDecoder(file)
	var visits []domain.Visit
	segmentIndex := 0
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		var document timelineDocument
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode timeline export: %w", err)
		}
		segments := make([]timelineSegment, 0, len(document.SemanticSegments)+len(document.TimelineObjects)+1)
		segments = append(segments, document.SemanticSegments...)
		segments = append(segments, document.TimelineObjects...)
		if document.PlaceVisit != nil {
			segments = append(segments, timelineSegment{StartTime: document.StartTime, EndTime: document.EndTime, PlaceVisit: document.PlaceVisit})
		}
		for _, segment := range segments {
			segmentIndex++
			if segment.PlaceVisit == nil {
				continue
			}
			p := segment.PlaceVisit
			start, err := timelineTime(p.StartTime, p.Duration.Start, p.Duration.StartMillis, segment.StartTime)
			if err != nil {
				return nil, fmt.Errorf("place visit %d start: %w", segmentIndex, err)
			}
			end, err := timelineTime(p.EndTime, p.Duration.End, p.Duration.EndMillis, segment.EndTime)
			if err != nil {
				return nil, fmt.Errorf("place visit %d end: %w", segmentIndex, err)
			}
			if !overlaps(start, end, from, to) {
				continue
			}
			if p.Location.LatitudeE7 == nil || p.Location.LongitudeE7 == nil {
				return nil, fmt.Errorf("place visit %d is missing coordinates", segmentIndex)
			}
			lat, lon := float64(*p.Location.LatitudeE7)/1e7, float64(*p.Location.LongitudeE7)/1e7
			if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
				return nil, fmt.Errorf("place visit %d has invalid coordinates", segmentIndex)
			}
			if end.Before(start) {
				return nil, fmt.Errorf("place visit %d ends before it starts", segmentIndex)
			}
			externalID := p.Location.PlaceID + "@" + start.UTC().Format(time.RFC3339Nano)
			if p.Location.PlaceID == "" {
				externalID = fmt.Sprintf("%s@%.7f,%.7f", start.UTC().Format(time.RFC3339Nano), lon, lat)
			}
			identity := domain.SourceIdentity{System: "google-timeline", ExternalID: externalID}
			visits = append(visits, domain.Visit{Coord: orb.Point{lon, lat}, Label: p.Location.Name, Arrive: start, Depart: end, Confidence: 1, SourceRef: identity.Ref(), Provenance: domain.Provenance{Source: identity, ObservedAt: start, Confidence: 1}})
		}
	}
	return visits, nil
}

func timelineTime(values ...string) (time.Time, error) {
	for _, value := range values {
		if value == "" {
			continue
		}
		if at, err := time.Parse(time.RFC3339Nano, value); err == nil {
			return at, nil
		}
		// Some exports encode epoch milliseconds as a JSON string.
		if strings.HasSuffix(value, "Z") {
			continue
		}
		millis, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			return time.UnixMilli(millis).UTC(), nil
		}
		return time.Time{}, fmt.Errorf("invalid timestamp %q", value)
	}
	return time.Time{}, errors.New("timestamp is missing")
}
