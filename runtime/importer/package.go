package importer

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	"gopkg.in/yaml.v3"

	"github.com/azusachino/felicia/core"
	"github.com/azusachino/felicia/core/domain"
	journeypackage "github.com/azusachino/felicia/core/journeypackage"
)

// stopsMember is the optional package member carrying the producer's already
// derived, reviewable stops (ADR-0034).
const stopsMember = "stops.yaml"

// kindRegistry is the embedded kind-template registry (core/kinds/*.yaml) — the
// same data server/cmd/api loads. The import boundary and the admin API must
// agree on what a kind is, or an import seeds records the GUI can never save
// (ADR-0013, issue #77). Parsed once; the result is read-only.
var kindRegistry = sync.OnceValues(func() (*domain.Registry, error) {
	templates, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		return nil, fmt.Errorf("open embedded kind templates: %w", err)
	}
	registry, err := domain.LoadRegistry(templates)
	if err != nil {
		return nil, fmt.Errorf("load embedded kind templates: %w", err)
	}
	return registry, nil
})

// PackageDocument is the normalized, database-independent import document.
type PackageDocument struct {
	Journey  *domain.Journey
	Stops    []*domain.StopCandidate
	Mementos []*domain.Memento
	Photos   []*domain.MementoPhoto
}

// PackageStore is the first local import composition seam. The SQLite provider
// implements EnsureJournal; server/PostgreSQL composition can add the same
// idempotent operation later without changing package normalization.
type PackageStore interface {
	domain.Repository
	EnsureJournal(context.Context, *domain.Journal) error
	// UpsertStopCandidate seeds the private intake inbox. Both providers key it
	// on candidate identity and leave review-owned columns (state, merge target,
	// an authored label) untouched on conflict, which is what makes a repeated
	// package import safe — see ApplyPackage.
	UpsertStopCandidate(context.Context, *domain.StopCandidate) error
}

// ImportReport summarizes one applied package.
type ImportReport struct {
	Journeys int
	Stops    int
	Mementos int
	Photos   int
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
	// The stops the package carries are the trip's intake inbox: the admin
	// review surface lists exactly what ListStopCandidatesByJourney returns, so
	// without this write a CLI-imported journey could never be curated at all
	// (issue #79). This is a *source* write like the two above — candidate
	// identity, not the supplied row ID, is the idempotency key, and both
	// providers leave review-owned columns (state, merge target, an authored
	// label) alone on conflict. So a re-import refreshes derived values without
	// resurrecting a discarded stop, undoing a merge, or renaming a stop the
	// author named.
	for _, stop := range document.Stops {
		// Upsert reflects the stored row back into its argument; copy so a
		// decoded document can be validated, reported, and applied repeatedly.
		candidate := *stop
		if err := store.UpsertStopCandidate(ctx, &candidate); err != nil {
			return ImportReport{}, fmt.Errorf("import stop candidate %s: %w", stop.Identity.Key, err)
		}
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
	return ImportReport{Journeys: 1, Stops: len(document.Stops), Mementos: len(document.Mementos), Photos: len(document.Photos)}, nil
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

// stopFile is one entry of the optional `stops.yaml` package member: the
// reviewable stops the producer's planner already derived, transported rather
// than re-derived. Review state is deliberately absent — a package may seed the
// intake inbox but may not claim an author decision, which stays the GUI's
// (ADR-0030). A producer that has already discarded a stop simply omits it.
type stopFile struct {
	CandidateKey      string         `yaml:"candidate_key"`
	DerivationVersion string         `yaml:"derivation_version"`
	Label             string         `yaml:"label"`
	Coord             []float64      `yaml:"coord"`
	Arrive            string         `yaml:"arrive"`
	Depart            string         `yaml:"depart"`
	Confidence        float64        `yaml:"confidence"`
	Evidence          []evidenceFile `yaml:"evidence"`
}

type evidenceFile struct {
	Kind    string     `yaml:"kind"`
	Source  sourceFile `yaml:"source"`
	Locator string     `yaml:"locator"`
}

type sourceFile struct {
	System     string `yaml:"system"`
	ExternalID string `yaml:"external_id"`
}

type mementoFile struct {
	ID             string          `yaml:"id"`
	Seq            int             `yaml:"seq"`
	Kind           string          `yaml:"kind"`
	OccurredAt     string          `yaml:"occurred_at"`
	OccurredTZ     string          `yaml:"occurred_tz"`
	Title          string          `yaml:"title"`
	Place          string          `yaml:"place"`
	Geom           mementoGeometry `yaml:"geom"`
	Vendor         string          `yaml:"vendor"`
	Essay          string          `yaml:"essay"`
	PriceAmount    *int64          `yaml:"price_amount"`
	PriceCurrency  string          `yaml:"price_currency"`
	AuthoredFields []string        `yaml:"authored_fields"`
	KindData       map[string]any  `yaml:"kind_data"`
	State          string          `yaml:"state"`
	Photos         []photoFile     `yaml:"photos"`
}

// mementoGeometry is the `geom` value of a `mementos.yaml` entry. A memento
// holds exactly one geometry, and the registry declares its shape per kind
// through `anchor` (core/kinds/*.yaml), so one key carries both shapes: a
// `[longitude, latitude]` pair for a point-anchored kind, or a list of two or
// more such pairs for an edge-anchored one such as `transit` (ADR-0035). Which
// shape is legal is not the format's decision — `domain.ValidateMementoGeometry`
// checks it against the kind's anchor, exactly as the admin API does.
type mementoGeometry struct {
	points [][]float64
	line   bool
}

// UnmarshalYAML accepts the two shapes above and rejects everything else. The
// element type decides the shape (numbers = one point, sequences = a line); an
// absent, null, or empty value means the memento carries no geometry yet.
func (g *mementoGeometry) UnmarshalYAML(node *yaml.Node) error {
	var point []float64
	if err := node.Decode(&point); err == nil {
		if len(point) == 0 {
			return nil
		}
		if len(point) != 2 {
			return fmt.Errorf("geom must contain longitude and latitude")
		}
		g.points = [][]float64{point}
		return nil
	}
	var line [][]float64
	if err := node.Decode(&line); err != nil {
		return fmt.Errorf("geom must be a [longitude, latitude] pair or a list of such pairs")
	}
	for _, pair := range line {
		if len(pair) != 2 {
			return fmt.Errorf("geom must contain longitude and latitude")
		}
	}
	g.points, g.line = line, true
	return nil
}

// geometry converts the decoded value into the domain geometry. It performs no
// validation: range and anchor are the write boundary's business.
func (g mementoGeometry) geometry() orb.Geometry {
	if len(g.points) == 0 {
		return nil
	}
	if !g.line {
		return orb.Point{g.points[0][0], g.points[0][1]}
	}
	line := make(orb.LineString, 0, len(g.points))
	for _, pair := range g.points {
		line = append(line, orb.Point{pair[0], pair[1]})
	}
	return line
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

// DecodePackage normalizes package files without writing to a database, and is
// where the write boundary is enforced for the import path: every memento is
// checked against the same registry and the same domain validators the admin API
// uses before a save (ADR-0013). It has to happen here rather than in
// ApplyPackage, which persists stop candidates before it reaches the memento
// loop and runs in no transaction (issue #76) — a memento rejected there would
// leave written stops behind. A package is therefore accepted whole or not at
// all, and nothing is persisted for a rejected one.
func DecodePackage(pkg *journeypackage.Package) (*PackageDocument, error) {
	if pkg == nil {
		return nil, fmt.Errorf("package is required")
	}
	registry, err := kindRegistry()
	if err != nil {
		return nil, err
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
	// stops.yaml is optional: older packages and hand-written ones may omit it,
	// and an absent member must import cleanly rather than fail (or invent
	// stops).
	if _, ok := pkg.Files[stopsMember]; ok {
		var rawStops []stopFile
		if err := decodeYAML(pkg, stopsMember, &rawStops); err != nil {
			return nil, err
		}
		for index, raw := range rawStops {
			stop, err := normalizeStop(pkg, journey.ID, raw)
			if err != nil {
				return nil, fmt.Errorf("stop %d: %w", index+1, err)
			}
			document.Stops = append(document.Stops, stop)
		}
	}
	for index, raw := range rawMementos {
		memento, photos, err := normalizeMemento(pkg, registry, journey.ID, raw)
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

// normalizeStop turns one `stops.yaml` entry into a reviewable stop candidate.
//
// Nothing here is invented. The candidate's identity is the producer's own
// derivation version and key, so re-planning the same source material lands on
// the same row instead of a near-duplicate; both are required, because a
// candidate whose derivation is unknown cannot be matched or explained.
// Provenance names the package member the candidate arrived in — a derived stop
// with blank provenance is unattributable, which ADR-0010 forbids and this repo
// has already shipped once.
func normalizeStop(pkg *journeypackage.Package, journeyID uuid.UUID, raw stopFile) (*domain.StopCandidate, error) {
	key := strings.TrimSpace(raw.CandidateKey)
	derivation := strings.TrimSpace(raw.DerivationVersion)
	if key == "" || derivation == "" {
		return nil, fmt.Errorf("candidate_key and derivation_version are required")
	}
	if len(raw.Coord) != 2 {
		return nil, fmt.Errorf("coord must contain longitude and latitude")
	}
	if raw.Coord[0] < -180 || raw.Coord[0] > 180 || raw.Coord[1] < -90 || raw.Coord[1] > 90 {
		return nil, fmt.Errorf("coord is out of range")
	}
	arrive, err := time.Parse(time.RFC3339, raw.Arrive)
	if err != nil {
		return nil, fmt.Errorf("arrive: %w", err)
	}
	depart, err := time.Parse(time.RFC3339, raw.Depart)
	if err != nil {
		return nil, fmt.Errorf("depart: %w", err)
	}
	if depart.Before(arrive) {
		return nil, fmt.Errorf("depart must not precede arrive")
	}
	if raw.Confidence < 0 || raw.Confidence > 1 {
		return nil, fmt.Errorf("confidence must be between 0 and 1")
	}
	source := domain.SourceIdentity{System: "package:" + pkg.Manifest.PackageID, ExternalID: key}
	evidence := make([]domain.EvidenceRef, 0, len(raw.Evidence))
	for index, rawEvidence := range raw.Evidence {
		ref := domain.EvidenceRef{
			Kind:    domain.EvidenceKind(rawEvidence.Kind),
			Source:  domain.SourceIdentity{System: rawEvidence.Source.System, ExternalID: rawEvidence.Source.ExternalID},
			Locator: rawEvidence.Locator,
		}
		if err := ref.Validate(); err != nil {
			return nil, fmt.Errorf("evidence %d: %w", index+1, err)
		}
		evidence = append(evidence, ref)
	}
	if len(evidence) == 0 {
		// A producer that carries no upstream evidence still owes the reviewer an
		// explanation of where the stop came from. The package member is the true
		// answer, so say that rather than fabricating an upstream observation.
		evidence = append(evidence, domain.EvidenceRef{Kind: domain.EvidenceVisit, Source: source, Locator: stopsMember + "#" + key})
	}
	// AuthoredFields stays empty and State stays proposed: an import seeds
	// candidates, it never claims authorship or an author's review decision.
	return &domain.StopCandidate{
		ID:         uuid.NewSHA1(journeyID, []byte(derivation+"\x00"+key)),
		JourneyID:  journeyID,
		Identity:   domain.CandidateIdentity{DerivationVersion: derivation, Key: key},
		Label:      raw.Label,
		Coord:      orb.Point{raw.Coord[0], raw.Coord[1]},
		Arrive:     arrive,
		Depart:     depart,
		Confidence: raw.Confidence,
		Evidence:   evidence,
		State:      domain.CandidateProposed,
		// ObservedAt is the stop's own arrival, the same convention the planner
		// uses for a derived visit: the package manifest carries no dependable
		// creation time, and a zero observation time is precisely the blank
		// provenance ADR-0010 rules out.
		Provenance: []domain.Provenance{{Source: source, ObservedAt: arrive, Confidence: raw.Confidence}},
	}, nil
}

func normalizeMemento(pkg *journeypackage.Package, registry *domain.Registry, journeyID uuid.UUID, raw mementoFile) (*domain.Memento, []*domain.MementoPhoto, error) {
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
	geom := raw.Geom.geometry()
	kindData, err := json.Marshal(raw.KindData)
	if err != nil {
		return nil, nil, fmt.Errorf("kind_data: %w", err)
	}
	source := domain.SourceIdentity{System: "package:" + pkg.Manifest.PackageID, ExternalID: raw.ID}
	state := domain.MementoState(raw.State)
	if state == "" {
		state = domain.MementoCandidateState
	}
	if err := checkWriteBoundary(registry, raw.Kind, raw.KindData, state, raw.OccurredTZ, geom); err != nil {
		return nil, nil, err
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

// checkWriteBoundary runs the ADR-0013 write-boundary checks on one memento:
// the kind must be registered, kind_data must satisfy its template, the
// occurrence timezone must be a usable IANA zone, and the geometry must match
// the kind's anchor. These are the same registry and the same domain validators
// the admin API runs before a save (server/api/server.go), which is the point:
// before this, an import could seed a memento the GUI would then reject forever
// — an unregistered kind with 400 kind_not_registered, or an edge-anchored
// `transit` carrying a single point with anchor_mismatch (issue #77).
//
// Completeness follows lifecycle state, as it does at the API. The one
// difference is `candidate`: it is the importer's own creation state and means
// "source-derived, awaiting authoring" (docs/contracts/memento-lifecycle.md §1),
// so it is held to a draft's leniency rather than a finished record's. Required
// fields and geometry may still be missing on a candidate; everything present is
// still type-checked, the field set is still closed, and the kind must still be
// registered. Refusing incomplete candidates would refuse intake itself.
func checkWriteBoundary(registry *domain.Registry, kind string, kindData map[string]any, state domain.MementoState, occurredTZ string, geom orb.Geometry) error {
	template, registered := registry.Template(kind)
	if !registered {
		return fmt.Errorf("kind %q is not a registered memento kind (registered kinds: %s)", kind, strings.Join(registeredKinds(registry), ", "))
	}
	if issues := domain.ValidateForState(template, kindData, completenessState(state)); len(issues) > 0 {
		for _, issue := range issues {
			if issue.Code == domain.CodeInvalidState {
				return fmt.Errorf("state %q is not a memento lifecycle state (%s)", state, domain.CodeInvalidState)
			}
		}
		return fmt.Errorf("kind_data does not satisfy the %q template (%s)", kind, formatIssues(issues))
	}
	complete := completenessState(state) != domain.MementoDraft
	switch {
	case occurredTZ != "":
		if issues := domain.ValidateOccurredTimezone(occurredTZ); len(issues) > 0 {
			return fmt.Errorf("occurred_tz %q is not a usable IANA timezone (%s)", occurredTZ, formatIssues(issues))
		}
	case complete:
		return fmt.Errorf("occurred_tz is required for a memento in state %q (%s)", state, domain.CodeInvalidTimezone)
	}
	if geom != nil || complete {
		if issues := domain.ValidateMementoGeometry(template.Anchor, geom); len(issues) > 0 {
			return fmt.Errorf("geom does not match the %q anchor of kind %q (%s)", template.Anchor, kind, formatIssues(issues))
		}
	}
	return nil
}

// completenessState maps a lifecycle state onto the completeness the import
// boundary demands of it — see checkWriteBoundary on why `candidate` is mapped
// to a draft. Any other value, including an unregistered one, passes through so
// domain.ValidateForState can report it as invalid_state.
func completenessState(state domain.MementoState) domain.MementoState {
	if state == domain.MementoCandidateState {
		return domain.MementoDraft
	}
	return state
}

func registeredKinds(registry *domain.Registry) []string {
	kinds := registry.Kinds()
	sort.Strings(kinds)
	return kinds
}

// formatIssues renders validation issues as "field: code" so an import failure
// names the same machine codes the admin API returns.
func formatIssues(issues []domain.Issue) string {
	parts := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Field == "" {
			parts = append(parts, issue.Code)
			continue
		}
		parts = append(parts, issue.Field+": "+issue.Code)
	}
	return strings.Join(parts, ", ")
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
