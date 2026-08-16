package importer

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core"
	"github.com/azusachino/felicia/core/domain"
	journeypackage "github.com/azusachino/felicia/core/journeypackage"
)

// The package import path must reject what the admin API's write boundary
// rejects (ADR-0013, issue #77). Before this, `normalizeMemento` checked only
// UUID/RFC3339 parsing, a non-empty kind, and coordinate range, so a package
// could seed a record the GUI would then refuse to save forever — and no
// CLI-imported memento could legally be `transit` at all, because the package
// format could not express the ≥2-point geometry an `anchor: edge` kind needs.

const boundaryJourneyYAML = "id: 00000000-0000-0000-0000-000000000001\n" +
	"journal_id: 00000000-0000-0000-0000-000000000002\n" +
	"slug: kyoto\ntitle: Kyoto\nplace: Kyoto\n" +
	"date_start: 2026-04-01\ndate_end: 2026-04-01\n"

// boundaryStopYAML is a valid stop. ApplyPackage persists stop candidates in a
// loop that runs *before* the memento loop, so it is the evidence that an
// invalid package writes nothing (issue #76 leaves the import untransacted).
const boundaryStopYAML = "- candidate_key: derived-route:cluster-001\n" +
	"  derivation_version: gpx-stops-v1\n" +
	"  label: \"\"\n" +
	"  coord: [135.7681, 35.0116]\n" +
	"  arrive: 2026-04-01T09:00:00+09:00\n" +
	"  depart: 2026-04-01T10:30:00+09:00\n" +
	"  confidence: 0.5\n"

// completeTransit is the memento this issue says must become possible: an
// edge-anchored kind with two resolved stations and a two-point geometry.
const completeTransit = "- id: 00000000-0000-0000-0000-000000000003\n" +
	"  seq: 1\n" +
	"  kind: transit\n" +
	"  occurred_at: 2026-04-01T09:00:00+09:00\n" +
	"  occurred_tz: Asia/Tokyo\n" +
	"  title: Kyoto to Osaka\n" +
	"  place: Kyoto\n" +
	"  state: draft\n" +
	"  geom:\n" +
	"    - [135.7681, 35.0116]\n" +
	"    - [135.5023, 34.7025]\n" +
	"  kind_data:\n" +
	"    operator: JR West\n" +
	"    line: Kyoto Line\n" +
	"    from:\n" +
	"      name: Kyoto\n" +
	"      coords: [135.7681, 35.0116]\n" +
	"    to:\n" +
	"      name: Osaka\n" +
	"      coords: [135.5023, 34.7025]\n"

func boundaryPackage(mementos string, extra map[string][]byte) *journeypackage.Package {
	files := map[string][]byte{
		"journey.yaml":  []byte(boundaryJourneyYAML),
		"mementos.yaml": []byte(mementos),
	}
	for name, data := range extra {
		files[name] = data
	}
	return &journeypackage.Package{Manifest: journeypackage.Manifest{PackageID: "local-kyoto"}, Files: files}
}

// adminRegistry loads the registry exactly as server/cmd/api does, so a claim
// that the imported record is savable is a claim about the real registry.
func adminRegistry(t *testing.T) *domain.Registry {
	t.Helper()
	sub, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		t.Fatalf("sub kinds fs: %v", err)
	}
	registry, err := domain.LoadRegistry(sub)
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	return registry
}

func decodeError(t *testing.T, mementos string) string {
	t.Helper()
	document, err := DecodePackage(boundaryPackage(mementos, nil))
	if err == nil {
		t.Fatalf("DecodePackage accepted a package the admin API would reject: %#v", document.Mementos)
	}
	return err.Error()
}

func TestDecodePackageRejectsAnUnregisteredKind(t *testing.T) {
	// The GUI answers 400 kind_not_registered for this row forever, so the
	// import must not create it — and must name the kind it could not resolve.
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: transitt\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom: [135.7681, 35.0116]\n")
	if !strings.Contains(message, "transitt") {
		t.Fatalf("error does not name the unregistered kind: %s", message)
	}
}

func TestDecodePackageRejectsAnEdgeKindWithPointGeometry(t *testing.T) {
	// The sharp case from issue #77: accepted by the importer, published by the
	// compiler, and rejected by every later GUI save with anchor_mismatch.
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom: [135.7681, 35.0116]\n")
	if !strings.Contains(message, domain.CodeAnchorMismatch) {
		t.Fatalf("error does not report the anchor mismatch: %s", message)
	}
}

func TestDecodePackageRejectsAnEdgeGeometryWithOnePoint(t *testing.T) {
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom:\n    - [135.7681, 35.0116]\n")
	if !strings.Contains(message, "geom") {
		t.Fatalf("error does not point at the geometry: %s", message)
	}
}

func TestDecodePackageRejectsAnOutOfRangeCoordinate(t *testing.T) {
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom:\n"+
		"    - [135.7681, 35.0116]\n    - [181.0, 34.7025]\n")
	if !strings.Contains(message, domain.CodeInvalidCoordinate) {
		t.Fatalf("error does not report the invalid coordinate: %s", message)
	}
}

func TestDecodePackageRejectsAnUnusableTimezone(t *testing.T) {
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: goods\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Mars/Olympus\n  state: draft\n  geom: [135.7681, 35.0116]\n")
	if !strings.Contains(message, domain.CodeInvalidTimezone) {
		t.Fatalf("error does not report the invalid timezone: %s", message)
	}
}

func TestDecodePackageRejectsAnUnknownLifecycleState(t *testing.T) {
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: goods\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: reviewed\n  geom: [135.7681, 35.0116]\n"+
		"  kind_data:\n    name: A charm\n")
	if !strings.Contains(message, domain.CodeInvalidState) {
		t.Fatalf("error does not report the invalid state: %s", message)
	}
}

func TestDecodePackageRejectsIncompleteKindDataOutsideDraft(t *testing.T) {
	// A published record must satisfy its template (ADR-0013); the compiler
	// would otherwise publish a row the GUI cannot save.
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: goods\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: published\n  geom: [135.7681, 35.0116]\n"+
		"  kind_data:\n    shop: Station kiosk\n")
	if !strings.Contains(message, domain.CodeRequiredMissing) {
		t.Fatalf("error does not report the missing required field: %s", message)
	}
}

func TestDecodePackageRejectsAnUnknownKindDataField(t *testing.T) {
	// The field set is closed in every state, drafts included.
	message := decodeError(t, "- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: goods\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom: [135.7681, 35.0116]\n"+
		"  kind_data:\n    memory: not a goods field\n")
	if !strings.Contains(message, domain.CodeUnknownField) {
		t.Fatalf("error does not report the unknown kind_data field: %s", message)
	}
}

func TestDecodePackageKeepsIngestedCandidatesIncomplete(t *testing.T) {
	// A `candidate` is source-derived and awaiting authoring
	// (docs/contracts/memento-lifecycle.md §1): required fields and geometry may
	// still be missing, or the importer could not seed intake at all.
	document, err := DecodePackage(boundaryPackage("- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: goods\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  kind_data: {}\n", nil))
	if err != nil {
		t.Fatalf("an ingested candidate must import while incomplete: %v", err)
	}
	if len(document.Mementos) != 1 || document.Mementos[0].State != domain.MementoCandidateState {
		t.Fatalf("unexpected decoded candidate: %#v", document.Mementos)
	}
}

func TestDecodePackageImportsATransitMementoTheAdminAPIWouldAccept(t *testing.T) {
	document, err := DecodePackage(boundaryPackage(completeTransit, nil))
	if err != nil {
		t.Fatalf("a transit memento with edge geometry must import: %v", err)
	}
	if len(document.Mementos) != 1 {
		t.Fatalf("unexpected memento count: %#v", document.Mementos)
	}
	memento := document.Mementos[0]
	line, ok := memento.Geom.(orb.LineString)
	if !ok || len(line) != 2 {
		t.Fatalf("edge geometry did not survive decoding: %#v", memento.Geom)
	}
	if line[0] != (orb.Point{135.7681, 35.0116}) || line[1] != (orb.Point{135.5023, 34.7025}) {
		t.Fatalf("edge geometry points are wrong: %#v", line)
	}

	// Now save it the way the admin GUI does: the same four checks
	// server/api/server.go runs, at the state an author marks it.
	registry := adminRegistry(t)
	template, registered := registry.Template(memento.Kind)
	if !registered {
		t.Fatalf("imported kind %q is not registered", memento.Kind)
	}
	var kindData map[string]any
	if err := json.Unmarshal(memento.KindData, &kindData); err != nil {
		t.Fatalf("imported kind_data is not an object: %v", err)
	}
	if issues := domain.ValidateForState(template, kindData, domain.MementoAuthored); len(issues) > 0 {
		t.Fatalf("the admin API would reject the imported kind_data: %#v", issues)
	}
	if issues := domain.ValidateOccurredTimezone(memento.OccurredTZ); len(issues) > 0 {
		t.Fatalf("the admin API would reject the imported timezone: %#v", issues)
	}
	if issues := domain.ValidateMementoGeometry(template.Anchor, memento.Geom); len(issues) > 0 {
		t.Fatalf("the admin API would reject the imported geometry: %#v", issues)
	}
}

// writeRecorder counts every write ApplyPackage performs, so "nothing was
// persisted" is an assertion rather than an assumption.
type writeRecorder struct {
	domain.Repository
	stops    int
	mementos int
	photos   int
}

func (s *writeRecorder) EnsureJournal(context.Context, *domain.Journal) error { return nil }
func (s *writeRecorder) ApplyIngestJourneyPatch(context.Context, *domain.IngestJourneyPatch) error {
	return nil
}
func (s *writeRecorder) UpsertStopCandidate(context.Context, *domain.StopCandidate) error {
	s.stops++
	return nil
}
func (s *writeRecorder) ApplyIngestMementoPatch(context.Context, *domain.IngestMementoPatch) error {
	s.mementos++
	return nil
}
func (s *writeRecorder) ApplyManualMementoPatch(context.Context, *domain.ManualMementoPatch) error {
	s.mementos++
	return nil
}
func (s *writeRecorder) UpsertPhoto(context.Context, *domain.MementoPhoto) error {
	s.photos++
	return nil
}

func TestAnInvalidPackagePersistsNothing(t *testing.T) {
	// ApplyPackage writes stop candidates before it reaches the memento loop and
	// runs in no transaction (issue #76), so validation has to happen at decode:
	// a package rejected there never reaches a store at all.
	pkg := boundaryPackage("- id: 00000000-0000-0000-0000-000000000003\n"+
		"  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n"+
		"  occurred_tz: Asia/Tokyo\n  state: draft\n  geom: [135.7681, 35.0116]\n",
		map[string][]byte{stopsMember: []byte(boundaryStopYAML)})

	store := &writeRecorder{}
	// The exact sequence cli/cmd/felicia runs for `import --apply`.
	document, err := DecodePackage(pkg)
	if err == nil {
		if _, applyErr := ApplyPackage(context.Background(), document, store); applyErr != nil {
			t.Fatalf("apply package: %v", applyErr)
		}
		t.Fatal("DecodePackage accepted a transit memento with point geometry")
	}
	if store.stops != 0 || store.mementos != 0 || store.photos != 0 {
		t.Fatalf("a rejected package persisted rows: stops=%d mementos=%d photos=%d", store.stops, store.mementos, store.photos)
	}
}
