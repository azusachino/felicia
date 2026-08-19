package importer

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"gopkg.in/yaml.v3"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
	"github.com/azusachino/felicia/apps/felicia-core/ports"
)

// PackageDocument is the normalized, database-independent import document.
type PackageDocument struct {
	Journey  *domain.Journey
	Stops    []*domain.StopCandidate
	Mementos []*domain.Memento
	Photos   []*domain.MementoPhoto
}

// PackageStore is the import composition seam. Providers implement the
// idempotent journal operation alongside the domain repository.
type PackageStore interface {
	domain.Repository
	EnsureJournal(context.Context, *domain.Journal) error
}

// ImportReport summarizes one applied package.
type ImportReport struct {
	Journeys   int
	Candidates int
	Mementos   int
	Photos     int
}

// ApplyPackage writes a normalized package into the canonical store. Source
// fields are applied through the ingest boundary; authored fields are never
// supplied by this operation.
func ApplyPackage(ctx context.Context, document *PackageDocument, store PackageStore) (ImportReport, error) {
	// Label every memento write this import performs as importer-sourced in the
	// lifecycle log (docs/contracts/memento-lifecycle.md §8).
	ctx = domain.WithEventSource(ctx, domain.EventSourceImporter)
	if document == nil || document.Journey == nil {
		return ImportReport{}, fmt.Errorf("package document and journey are required")
	}
	if store == nil {
		return ImportReport{}, fmt.Errorf("package store is required")
	}
	registry, err := DefaultRegistry()
	if err != nil {
		return ImportReport{}, fmt.Errorf("load kind registry: %w", err)
	}
	if err := ValidatePackageDocument(document, registry); err != nil {
		return ImportReport{}, err
	}
	transactional, ok := store.(domain.TransactionalRepository)
	if !ok {
		return ImportReport{}, fmt.Errorf("package store does not support transactions")
	}
	var report ImportReport
	err = transactional.WithTransaction(ctx, func(repository domain.Repository) error {
		packageStore, ok := repository.(PackageStore)
		if !ok {
			return fmt.Errorf("transaction repository does not support package import")
		}
		var err error
		report, err = applyPackage(ctx, document, packageStore)
		return err
	})
	if err != nil {
		return ImportReport{}, err
	}
	return report, nil
}

func applyPackage(ctx context.Context, document *PackageDocument, store PackageStore) (ImportReport, error) {
	if err := store.EnsureJournal(ctx, &domain.Journal{ID: document.Journey.JournalID}); err != nil {
		return ImportReport{}, fmt.Errorf("ensure journal: %w", err)
	}
	// The journey is written through the ingest seam, not UpsertJourney: a
	// package import must not overwrite authored journey fields or reset the
	// journey's authored mask (ADR-0033). gps_route is only claimed when the
	// package actually carries a route, so a route-less re-import cannot blank
	// an already-imported track.
	journeyFields := []string{"slug", "source_ref", "title", "place", "country", "region", "date_start", "date_end"}
	if len(document.Journey.GPSRoute) > 0 {
		journeyFields = append(journeyFields, "gps_route")
	}
	if err := store.ApplyIngestJourneyPatch(ctx, &domain.IngestJourneyPatch{Journey: document.Journey, Fields: journeyFields}); err != nil {
		return ImportReport{}, fmt.Errorf("upsert journey: %w", err)
	}
	for _, memento := range document.Mementos {
		fields := []string{"journey_id", "kind", "seq", "occurred_at", "occurred_tz", "geom", "title", "place", "kind_data"}
		if err := store.ApplyIngestMementoPatch(ctx, &domain.IngestMementoPatch{Memento: memento, Fields: fields}); err != nil {
			return ImportReport{}, fmt.Errorf("import memento %s: %w", memento.ID, err)
		}
		if len(memento.AuthoredFields) > 0 || (memento.State != "" && memento.State != domain.MementoCandidateState) {
			if err := store.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{Memento: memento, Fields: memento.AuthoredFields, State: memento.State}); err != nil {
				return ImportReport{}, fmt.Errorf("apply authored memento %s: %w", memento.ID, err)
			}
		}
	}
	for _, photo := range document.Photos {
		if err := store.UpsertPhoto(ctx, photo); err != nil {
			return ImportReport{}, fmt.Errorf("import photo %s: %w", photo.ID, err)
		}
	}
	if len(document.Stops) > 0 {
		candidates, ok := store.(ports.StopCandidateStore)
		if !ok {
			return ImportReport{}, fmt.Errorf("package store does not support intake candidates")
		}
		for _, candidate := range document.Stops {
			if err := candidates.UpsertStopCandidate(ctx, candidate); err != nil {
				return ImportReport{}, fmt.Errorf("import stop candidate %s: %w", candidate.Identity.Key, err)
			}
		}
	}
	return ImportReport{Journeys: 1, Candidates: len(document.Stops), Mementos: len(document.Mementos), Photos: len(document.Photos)}, nil
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

type stopFile struct {
	ID                string    `yaml:"id"`
	DerivationVersion string    `yaml:"derivation_version"`
	Key               string    `yaml:"key"`
	Label             string    `yaml:"label"`
	Coord             []float64 `yaml:"coord"`
	Arrive            string    `yaml:"arrive"`
	Depart            string    `yaml:"depart"`
	Confidence        float64   `yaml:"confidence"`
}

type mementoFile struct {
	ID             string         `yaml:"id"`
	Seq            int            `yaml:"seq"`
	Kind           string         `yaml:"kind"`
	OccurredAt     string         `yaml:"occurred_at"`
	OccurredTZ     string         `yaml:"occurred_tz"`
	Title          string         `yaml:"title"`
	Place          string         `yaml:"place"`
	Geom           any            `yaml:"geom"`
	Vendor         string         `yaml:"vendor"`
	Essay          string         `yaml:"essay"`
	PriceAmount    *int64         `yaml:"price_amount"`
	PriceCurrency  string         `yaml:"price_currency"`
	AuthoredFields []string       `yaml:"authored_fields"`
	KindData       map[string]any `yaml:"kind_data"`
	State          string         `yaml:"state"`
	Photos         []photoFile    `yaml:"photos"`
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
	var rawStops []stopFile
	if _, ok := pkg.Files["stops.yaml"]; ok {
		if err := decodeYAML(pkg, "stops.yaml", &rawStops); err != nil {
			return nil, err
		}
		for index, raw := range rawStops {
			candidate, err := normalizeStop(pkg, journey.ID, raw)
			if err != nil {
				return nil, fmt.Errorf("stop %d: %w", index+1, err)
			}
			document.Stops = append(document.Stops, candidate)
		}
	}
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

func normalizeStop(pkg *journeypackage.Package, journeyID uuid.UUID, raw stopFile) (*domain.StopCandidate, error) {
	if raw.ID == "" {
		raw.ID = uuid.NewSHA1(uuid.Nil, []byte("package:"+pkg.Manifest.PackageID+":"+raw.Key)).String()
	}
	id, err := uuid.Parse(raw.ID)
	if err != nil {
		return nil, fmt.Errorf("id: %w", err)
	}
	if raw.DerivationVersion == "" || raw.Key == "" || len(raw.Coord) != 2 {
		return nil, fmt.Errorf("derivation_version, key, and coordinate are required")
	}
	if raw.Coord[0] < -180 || raw.Coord[0] > 180 || raw.Coord[1] < -90 || raw.Coord[1] > 90 || (raw.Coord[0] == 0 && raw.Coord[1] == 0) {
		return nil, fmt.Errorf("coordinate is invalid")
	}
	arrive, err := time.Parse(time.RFC3339, raw.Arrive)
	if err != nil {
		return nil, fmt.Errorf("arrive: %w", err)
	}
	depart, err := time.Parse(time.RFC3339, raw.Depart)
	if err != nil {
		return nil, fmt.Errorf("depart: %w", err)
	}
	if depart.Before(arrive) || raw.Confidence < 0 || raw.Confidence > 1 {
		return nil, fmt.Errorf("time range or confidence is invalid")
	}
	source := domain.SourceIdentity{System: "package:" + pkg.Manifest.PackageID, ExternalID: raw.Key}
	return &domain.StopCandidate{
		ID: id, JourneyID: journeyID,
		Identity: domain.CandidateIdentity{DerivationVersion: raw.DerivationVersion, Key: raw.Key},
		Label:    raw.Label, Coord: orb.Point{raw.Coord[0], raw.Coord[1]}, Arrive: arrive, Depart: depart,
		Confidence: raw.Confidence, State: domain.CandidateProposed,
		Evidence:   []domain.EvidenceRef{{Kind: domain.EvidenceRoute, Source: source, Locator: raw.Key}},
		Provenance: []domain.Provenance{{Source: source, ObservedAt: arrive, Confidence: raw.Confidence}},
	}, nil
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
	geom, err := normalizeGeometry(raw.Geom)
	if err != nil {
		return nil, nil, err
	}
	kindData, err := json.Marshal(raw.KindData)
	if err != nil {
		return nil, nil, fmt.Errorf("kind_data: %w", err)
	}
	source := domain.SourceIdentity{System: "package:" + pkg.Manifest.PackageID, ExternalID: raw.ID}
	state := domain.MementoState(raw.State)
	if state == "" {
		state = domain.MementoCandidateState
	}
	memento := &domain.Memento{ID: id, JourneyID: journeyID, Kind: raw.Kind, Seq: raw.Seq, OccurredAt: occurredAt, OccurredTZ: raw.OccurredTZ, Geom: geom, Title: raw.Title, Place: raw.Place, Vendor: optional(raw.Vendor), Essay: optional(raw.Essay), PriceAmount: raw.PriceAmount, PriceCurrency: optional(raw.PriceCurrency), AuthoredFields: raw.AuthoredFields, KindData: kindData, SourceIdentity: &source, State: state}
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

func normalizeGeometry(raw any) (orb.Geometry, error) {
	if raw == nil {
		return nil, nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("geom: %w", err)
	}
	var value any
	if err := json.Unmarshal(encoded, &value); err != nil {
		return nil, fmt.Errorf("geom: %w", err)
	}
	coords, ok := value.([]any)
	if !ok || len(coords) == 0 {
		return nil, fmt.Errorf("geom must be a coordinate pair or line")
	}
	if len(coords) == 2 && isCoordinateNumber(coords[0]) && isCoordinateNumber(coords[1]) {
		point := orb.Point{number(coords[0]), number(coords[1])}
		if !validCoordinate(point) {
			return nil, fmt.Errorf("geom coordinate is out of range")
		}
		return point, nil
	}
	line := make(orb.LineString, 0, len(coords))
	for _, rawPoint := range coords {
		pair, ok := rawPoint.([]any)
		if !ok || len(pair) != 2 || !isCoordinateNumber(pair[0]) || !isCoordinateNumber(pair[1]) {
			return nil, fmt.Errorf("geom line must contain longitude and latitude pairs")
		}
		point := orb.Point{number(pair[0]), number(pair[1])}
		if !validCoordinate(point) {
			return nil, fmt.Errorf("geom coordinate is out of range")
		}
		line = append(line, point)
	}
	if len(line) < 2 {
		return nil, fmt.Errorf("geom line must contain at least two points")
	}
	return line, nil
}

func isCoordinateNumber(value any) bool {
	n, ok := value.(float64)
	return ok && !math.IsNaN(n) && !math.IsInf(n, 0)
}

func number(value any) float64 { return value.(float64) }

func validCoordinate(point orb.Point) bool {
	return point.X() >= -180 && point.X() <= 180 && point.Y() >= -90 && point.Y() <= 90
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
