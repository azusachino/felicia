// Package local adapts files already present on the user's machine. It does
// not contact a service and keeps source references relative to the selected
// input root.
package local

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// GPXSource reads timestamped track and route points from one local GPX file.
type GPXSource struct{ path string }

var _ domain.RouteSource = (*GPXSource)(nil)

// NewGPXSource creates an offline route source for path.
func NewGPXSource(path string) *GPXSource { return &GPXSource{path: path} }

// FetchRoutes parses all track/route segments. Segments without timestamps
// remain usable as route geometry, but have zero time bounds for the planner
// to report rather than inventing capture times.
func (s *GPXSource) FetchRoutes(ctx context.Context, from, to time.Time) ([]domain.Route, error) {
	if s == nil || s.path == "" {
		return nil, errors.New("gpx path is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	file, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open gpx %s: %w", s.path, err)
	}
	defer func() { _ = file.Close() }()
	hash, err := fileSHA256(file)
	if err != nil {
		return nil, fmt.Errorf("fingerprint gpx %s: %w", s.path, err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("rewind gpx %s: %w", s.path, err)
	}
	segments, err := parseGPX(file)
	if err != nil {
		return nil, fmt.Errorf("parse gpx %s: %w", s.path, err)
	}
	routes := make([]domain.Route, 0, len(segments))
	for index, segment := range segments {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(segment) < 2 {
			continue
		}
		route := routeFromPoints(segment, fmt.Sprintf("local:gpx:%s:segment-%03d", hash, index+1))
		if !overlaps(route.From, route.To, from, to) {
			continue
		}
		routes = append(routes, route)
	}
	return routes, nil
}

// PhotoSource walks a local media directory. Local files have no dependable
// capture timestamp without EXIF decoding; At is therefore zero and the
// planner leaves them visible as unattached evidence instead of treating file
// modification time as a memory timestamp.
type PhotoSource struct {
	root    string
	sidecar string
}

var _ domain.PhotoSource = (*PhotoSource)(nil)

// NewPhotoSource creates an offline media source rooted at root.
func NewPhotoSource(root string) *PhotoSource { return &PhotoSource{root: root} }

// NewPhotoSourceWithSidecar loads optional user-authored JSONL metadata. Each
// record is keyed by a path relative to root; malformed records are ignored so
// one bad line cannot prevent the remaining photo tray from opening.
func NewPhotoSourceWithSidecar(root, sidecar string) *PhotoSource {
	return &PhotoSource{root: root, sidecar: sidecar}
}

// FetchAssets returns deterministic image/video assets. The requested range
// is intentionally not applied to timestamp-less local files.
func (s *PhotoSource) FetchAssets(ctx context.Context, _, _ time.Time) ([]domain.PhotoAsset, error) {
	if s == nil || s.root == "" {
		return nil, errors.New("photo root is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var paths []string
	err := filepath.WalkDir(s.root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if isMediaFile(path) {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk media root %s: %w", s.root, err)
	}
	sort.Strings(paths)
	metadata, err := readSidecar(s.sidecar)
	if err != nil {
		return nil, err
	}
	assets := make([]domain.PhotoAsset, 0, len(paths))
	for _, path := range paths {
		asset, err := localAsset(s.root, path)
		if err != nil {
			return nil, err
		}
		if record, ok := metadata[asset.ID]; ok {
			applySidecar(&asset, record)
		}
		assets = append(assets, asset)
	}
	return assets, nil
}

type sidecarRecord struct {
	Path      string    `json:"path"`
	At        string    `json:"at"`
	Timestamp string    `json:"timestamp"`
	Coord     []float64 `json:"coord"`
	Title     string    `json:"title"`
	Kind      string    `json:"kind"`
}

func readSidecar(filename string) (map[string]sidecarRecord, error) {
	metadata := make(map[string]sidecarRecord)
	if filename == "" {
		return metadata, nil
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open photo sidecar %s: %w", filename, err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record sidecarRecord
		if json.Unmarshal(scanner.Bytes(), &record) != nil || !safeRelativePath(record.Path) {
			continue
		}
		metadata[filepath.ToSlash(record.Path)] = record
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read photo sidecar %s: %w", filename, err)
	}
	return metadata, nil
}

func applySidecar(asset *domain.PhotoAsset, record sidecarRecord) {
	value := record.At
	if value == "" {
		value = record.Timestamp
	}
	if value != "" {
		if at, err := time.Parse(time.RFC3339, value); err == nil {
			asset.At = at
			asset.Provenance = domain.Provenance{Source: domain.SourceIdentity{System: "local-sidecar", ExternalID: asset.ID}, ObservedAt: at, Confidence: 0.9}
		}
	}
	if len(record.Coord) >= 2 && record.Coord[0] >= -180 && record.Coord[0] <= 180 && record.Coord[1] >= -90 && record.Coord[1] <= 90 {
		point := orb.Point{record.Coord[0], record.Coord[1]}
		asset.Coord = &point
	}
	if record.Title != "" {
		asset.Title = record.Title
	}
	if record.Kind != "" {
		asset.Kind = domain.MediaKind(record.Kind)
	}
}

func safeRelativePath(value string) bool {
	clean := filepath.ToSlash(filepath.Clean(value))
	return value != "" && clean == filepath.ToSlash(value) && clean != "." && !strings.HasPrefix(clean, "/") && !strings.HasPrefix(clean, "../")
}

type gpxSegment []domain.TrackPoint

type gpxDocument struct {
	Tracks []gpxTrack `xml:"trk"`
	Routes []gpxRoute `xml:"rte"`
}
type gpxTrack struct {
	Segments []gpxSegmentXML `xml:"trkseg"`
}
type gpxRoute struct {
	Points []gpxPoint `xml:"rtept"`
}
type gpxSegmentXML struct {
	Points []gpxPoint `xml:"trkpt"`
}
type gpxPoint struct {
	Lat  float64 `xml:"lat,attr"`
	Lon  float64 `xml:"lon,attr"`
	Time string  `xml:"time"`
}

func parseGPX(reader io.Reader) ([]gpxSegment, error) {
	var document gpxDocument
	decoder := xml.NewDecoder(bufio.NewReader(reader))
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	segments := make([]gpxSegment, 0)
	for _, track := range document.Tracks {
		for _, segment := range track.Segments {
			points, err := normalizePoints(segment.Points)
			if err != nil {
				return nil, err
			}
			segments = append(segments, points)
		}
	}
	for _, route := range document.Routes {
		points, err := normalizePoints(route.Points)
		if err != nil {
			return nil, err
		}
		segments = append(segments, points)
	}
	return segments, nil
}

func normalizePoints(points []gpxPoint) (gpxSegment, error) {
	result := make(gpxSegment, 0, len(points))
	for index, point := range points {
		if point.Lat < -90 || point.Lat > 90 || point.Lon < -180 || point.Lon > 180 {
			return nil, fmt.Errorf("point %d has invalid coordinate", index)
		}
		at := time.Time{}
		if point.Time != "" {
			parsed, err := time.Parse(time.RFC3339, point.Time)
			if err != nil {
				return nil, fmt.Errorf("point %d time: %w", index, err)
			}
			at = parsed
		}
		result = append(result, domain.TrackPoint{Coord: orb.Point{point.Lon, point.Lat}, At: at})
	}
	return result, nil
}

func routeFromPoints(points gpxSegment, sourceRef string) domain.Route {
	line := make(orb.LineString, 0, len(points))
	from, to := time.Time{}, time.Time{}
	for _, point := range points {
		line = append(line, point.Coord)
		if point.At.IsZero() {
			continue
		}
		if from.IsZero() || point.At.Before(from) {
			from = point.At
		}
		if to.IsZero() || point.At.After(to) {
			to = point.At
		}
	}
	return domain.Route{Line: line, Points: points, From: from, To: to, SourceRef: sourceRef, Provenance: domain.Provenance{Source: domain.SourceIdentity{System: "local-gpx", ExternalID: sourceRef}, ObservedAt: from, Confidence: 1}}
}

func overlaps(routeFrom, routeTo, from, to time.Time) bool {
	if routeFrom.IsZero() || routeTo.IsZero() || from.IsZero() || to.IsZero() {
		return true
	}
	return !routeTo.Before(from) && !routeFrom.After(to)
}

func fileSHA256(file *os.File) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func isMediaFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".mp4", ".mov", ".m4v":
		return true
	default:
		return false
	}
}

func localAsset(root, path string) (domain.PhotoAsset, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.PhotoAsset{}, fmt.Errorf("read local media %s: %w", path, err)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return domain.PhotoAsset{}, err
	}
	relative = filepath.ToSlash(relative)
	sum := sha256.Sum256(data)
	kind := domain.MediaImage
	if strings.HasPrefix(mime.TypeByExtension(filepath.Ext(path)), "video/") {
		kind = domain.MediaVideo
	}
	source := domain.SourceIdentity{System: "local-media", ExternalID: relative}
	return domain.PhotoAsset{ID: relative, Kind: kind, Checksum: "sha256:" + hex.EncodeToString(sum[:]), SourceRef: source.Ref(), Provenance: domain.Provenance{Source: source, Confidence: 1}, URI: relative, MIME: mime.TypeByExtension(filepath.Ext(path)), Title: strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)), Provider: "local"}, nil
}
