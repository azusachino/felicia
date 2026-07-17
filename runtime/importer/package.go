package importer

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"gopkg.in/yaml.v3"

	"github.com/azusachino/felicia/core/domain"
	journeypackage "github.com/azusachino/felicia/core/package"
)

// PackageDocument is the normalized, database-independent import document.
type PackageDocument struct {
	Journey  *domain.Journey
	Mementos []*domain.Memento
	Photos   []*domain.MementoPhoto
}

// PackageStore is the first local import composition seam. The SQLite provider
// implements EnsureJournal; server/PostgreSQL composition can add the same
// idempotent operation later without changing package normalization.
type PackageStore interface {
	domain.Repository
	EnsureJournal(context.Context, *domain.Journal) error
}

// ImportReport summarizes one applied package.
type ImportReport struct {
	Journeys int
	Mementos int
	Photos   int
}

// ApplyPackage writes a normalized package into the canonical store. Source
// fields are applied through the ingest boundary; authored fields are never
// supplied by this operation.
func ApplyPackage(ctx context.Context, document *PackageDocument, store PackageStore) (ImportReport, error) {
	if document == nil || document.Journey == nil {
		return ImportReport{}, fmt.Errorf("package document and journey are required")
	}
	if store == nil {
		return ImportReport{}, fmt.Errorf("package store is required")
	}
	if err := store.EnsureJournal(ctx, &domain.Journal{ID: document.Journey.JournalID}); err != nil {
		return ImportReport{}, fmt.Errorf("ensure journal: %w", err)
	}
	if err := store.UpsertJourney(ctx, document.Journey); err != nil {
		return ImportReport{}, fmt.Errorf("upsert journey: %w", err)
	}
	for _, memento := range document.Mementos {
		fields := []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "geom", "title", "place", "kind_data"}
		if err := store.ApplyIngestMementoPatch(ctx, &domain.IngestMementoPatch{Memento: memento, Fields: fields}); err != nil {
			return ImportReport{}, fmt.Errorf("import memento %s: %w", memento.ID, err)
		}
	}
	for _, photo := range document.Photos {
		if err := store.UpsertPhoto(ctx, photo); err != nil {
			return ImportReport{}, fmt.Errorf("import photo %s: %w", photo.ID, err)
		}
	}
	return ImportReport{Journeys: 1, Mementos: len(document.Mementos), Photos: len(document.Photos)}, nil
}

type journeyFile struct {
	ID        string `yaml:"id"`
	JournalID string `yaml:"journal_id"`
	Slug      string `yaml:"slug"`
	SourceRef string `yaml:"source_ref"`
	Title     string `yaml:"title"`
	Place     string `yaml:"place"`
	Country   string `yaml:"country"`
	Region    string `yaml:"region"`
	DateStart string `yaml:"date_start"`
	DateEnd   string `yaml:"date_end"`
}

type mementoFile struct {
	ID         string         `yaml:"id"`
	Seq        int            `yaml:"seq"`
	Kind       string         `yaml:"kind"`
	OccurredAt string         `yaml:"occurred_at"`
	OccurredTZ string         `yaml:"occurred_tz"`
	Title      string         `yaml:"title"`
	Place      string         `yaml:"place"`
	Geom       []float64      `yaml:"geom"`
	KindData   map[string]any `yaml:"kind_data"`
	Photos     []photoFile    `yaml:"photos"`
}

type photoFile struct {
	ID          string `yaml:"id"`
	Path        string `yaml:"path"`
	ContentHash string `yaml:"content_hash"`
	Caption     string `yaml:"caption"`
	Seq         int    `yaml:"seq"`
	TakenAt     string `yaml:"taken_at"`
}

type gpxFile struct {
	Tracks []struct {
		Segments []struct {
			Points []struct {
				Latitude  float64 `xml:"lat,attr"`
				Longitude float64 `xml:"lon,attr"`
			} `xml:"trkpt"`
		} `xml:"trkseg"`
	} `xml:"trk"`
}

// DecodePackage normalizes package files without writing to a database.
func DecodePackage(pkg *journeypackage.Package) (*PackageDocument, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package is required")
	}
	var rawJourney journeyFile
	if err := decodeYAML(pkg, "journey.yaml", &rawJourney); err != nil {
		return nil, err
	}
	journey, err := normalizeJourney(rawJourney)
	if err != nil {
		return nil, err
	}
	if route, ok := pkg.Files["route.gpx"]; ok {
		journey.GPSRoute, err = decodeGPX(route)
		if err != nil {
			return nil, err
		}
		ref := "route.gpx"
		journey.SourceRef = &ref
	}

	var rawMementos []mementoFile
	if err := decodeYAML(pkg, "mementos.yaml", &rawMementos); err != nil {
		return nil, err
	}
	document := &PackageDocument{Journey: journey}
	for index, raw := range rawMementos {
		memento, photos, err := normalizeMemento(pkg, journey.ID, raw)
		if err != nil {
			return nil, fmt.Errorf("memento %d: %w", index+1, err)
		}
		document.Mementos = append(document.Mementos, memento)
		document.Photos = append(document.Photos, photos...)
	}
	return document, nil
}

func normalizeJourney(raw journeyFile) (*domain.Journey, error) {
	id, err := uuid.Parse(raw.ID)
	if err != nil {
		return nil, fmt.Errorf("journey id: %w", err)
	}
	journalID, err := uuid.Parse(raw.JournalID)
	if err != nil {
		return nil, fmt.Errorf("journal_id: %w", err)
	}
	start, err := time.Parse("2006-01-02", raw.DateStart)
	if err != nil {
		return nil, fmt.Errorf("date_start: %w", err)
	}
	end, err := time.Parse("2006-01-02", raw.DateEnd)
	if err != nil {
		return nil, fmt.Errorf("date_end: %w", err)
	}
	if raw.Slug == "" || raw.Title == "" {
		return nil, fmt.Errorf("slug and title are required")
	}
	return &domain.Journey{ID: id, JournalID: journalID, Slug: raw.Slug, SourceRef: optional(raw.SourceRef), Title: raw.Title, Place: raw.Place, Country: optional(raw.Country), Region: optional(raw.Region), DateStart: start, DateEnd: end, GPSRoute: orb.MultiLineString{}}, nil
}

func normalizeMemento(pkg *journeypackage.Package, journeyID uuid.UUID, raw mementoFile) (*domain.Memento, []*domain.MementoPhoto, error) {
	id, err := uuid.Parse(raw.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("id: %w", err)
	}
	occurredAt, err := time.Parse(time.RFC3339, raw.OccurredAt)
	if err != nil {
		return nil, nil, fmt.Errorf("occurred_at: %w", err)
	}
	if raw.Kind == "" || raw.Seq < 1 {
		return nil, nil, fmt.Errorf("kind and positive seq are required")
	}
	if len(raw.Geom) != 0 && len(raw.Geom) != 2 {
		return nil, nil, fmt.Errorf("geom must contain longitude and latitude")
	}
	var geom orb.Geometry
	if len(raw.Geom) == 2 {
		if raw.Geom[0] < -180 || raw.Geom[0] > 180 || raw.Geom[1] < -90 || raw.Geom[1] > 90 {
			return nil, nil, fmt.Errorf("geom coordinate is out of range")
		}
		geom = orb.Point{raw.Geom[0], raw.Geom[1]}
	}
	kindData, err := json.Marshal(raw.KindData)
	if err != nil {
		return nil, nil, fmt.Errorf("kind_data: %w", err)
	}
	source := domain.SourceIdentity{System: "package:" + pkg.Manifest.PackageID, ExternalID: raw.ID}
	memento := &domain.Memento{ID: id, JourneyID: journeyID, Kind: raw.Kind, Seq: raw.Seq, OccurredAt: occurredAt, OccurredTZ: raw.OccurredTZ, Geom: geom, Title: raw.Title, Place: raw.Place, KindData: kindData, SourceIdentity: &source, State: domain.MementoCandidateState}
	photos := make([]*domain.MementoPhoto, 0, len(raw.Photos))
	for index, rawPhoto := range raw.Photos {
		photoID, err := uuid.Parse(rawPhoto.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("photo %d id: %w", index+1, err)
		}
		if rawPhoto.Path == "" || rawPhoto.ContentHash == "" || rawPhoto.Seq < 1 {
			return nil, nil, fmt.Errorf("photo %d requires path, content_hash, and positive seq", index+1)
		}
		if _, ok := pkg.Files[rawPhoto.Path]; !ok {
			return nil, nil, fmt.Errorf("photo %q is missing from package", rawPhoto.Path)
		}
		photos = append(photos, &domain.MementoPhoto{ID: photoID, MementoID: id, ObjectKey: rawPhoto.Path, ContentHash: rawPhoto.ContentHash, Caption: optional(rawPhoto.Caption), Seq: rawPhoto.Seq, SourceRef: optional("package:" + pkg.Manifest.PackageID + ":" + rawPhoto.ID)})
	}
	return memento, photos, nil
}

func decodeGPX(data []byte) (orb.MultiLineString, error) {
	var raw gpxFile
	if err := xml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode route.gpx: %w", err)
	}
	var route orb.MultiLineString
	for _, track := range raw.Tracks {
		for _, segment := range track.Segments {
			if len(segment.Points) < 2 {
				continue
			}
			line := make(orb.LineString, 0, len(segment.Points))
			for _, point := range segment.Points {
				line = append(line, orb.Point{point.Longitude, point.Latitude})
			}
			route = append(route, line)
		}
	}
	if len(route) == 0 {
		return nil, fmt.Errorf("route.gpx contains no track segment with two points")
	}
	return route, nil
}

func decodeYAML(pkg *journeypackage.Package, name string, value any) error {
	data, ok := pkg.Files[name]
	if !ok {
		return fmt.Errorf("%s is required", name)
	}
	if err := yaml.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode %s: %w", name, err)
	}
	return nil
}

func optional(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
