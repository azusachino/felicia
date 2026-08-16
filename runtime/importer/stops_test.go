package importer

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
	journeypackage "github.com/azusachino/felicia/core/journeypackage"
)

// candidateStore is a minimal PackageStore that applies the same stop-candidate
// rules both real providers apply in SQL: identity — not row ID — is the
// idempotency key, and an upsert refreshes source-owned fields while leaving
// review state, the merge target, and an authored label alone. It exists so
// these tests can prove ApplyPackage's behaviour without depending on a
// concrete provider (which would invert the runtime→providers layering); the
// cross-provider assertions live in providers/contract.
type candidateStore struct {
	domain.Repository
	rows   []*domain.StopCandidate
	writes int
}

func (s *candidateStore) EnsureJournal(context.Context, *domain.Journal) error    { return nil }
func (s *candidateStore) UpsertPhoto(context.Context, *domain.MementoPhoto) error { return nil }
func (s *candidateStore) ApplyIngestJourneyPatch(context.Context, *domain.IngestJourneyPatch) error {
	return nil
}
func (s *candidateStore) ApplyIngestMementoPatch(context.Context, *domain.IngestMementoPatch) error {
	return nil
}
func (s *candidateStore) ApplyManualMementoPatch(context.Context, *domain.ManualMementoPatch) error {
	return nil
}

func (s *candidateStore) UpsertStopCandidate(_ context.Context, candidate *domain.StopCandidate) error {
	s.writes++
	for _, row := range s.rows {
		if row.JourneyID != candidate.JourneyID || row.Identity != candidate.Identity {
			continue
		}
		if !slices.Contains(row.AuthoredFields, "label") {
			row.Label = candidate.Label
		}
		row.Coord = candidate.Coord
		row.Arrive, row.Depart = candidate.Arrive, candidate.Depart
		row.Confidence = candidate.Confidence
		row.Provenance = candidate.Provenance
		row.Evidence = candidate.Evidence
		row.Revision++
		return nil
	}
	stored := *candidate
	if stored.ID == uuid.Nil {
		stored.ID = uuid.Must(uuid.NewV7())
	}
	if stored.State == "" {
		stored.State = domain.CandidateProposed
	}
	stored.Revision = 1
	s.rows = append(s.rows, &stored)
	return nil
}

// review is the author decision the admin GUI makes (ApplyStopReview).
func (s *candidateStore) review(id uuid.UUID, state domain.CandidateState, label string) {
	for _, row := range s.rows {
		if row.ID != id {
			continue
		}
		row.State = state
		if label != "" {
			row.Label = label
			if !slices.Contains(row.AuthoredFields, "label") {
				row.AuthoredFields = append(row.AuthoredFields, "label")
			}
		}
		row.Revision++
	}
}

func stopPackage(t *testing.T) *journeypackage.Package {
	t.Helper()
	return &journeypackage.Package{
		Manifest: journeypackage.Manifest{PackageID: "local-kyoto", CreatedAt: "2026-04-02T00:00:00Z"},
		Files: map[string][]byte{
			"journey.yaml":  []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: kyoto\ntitle: Kyoto\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
			"mementos.yaml": []byte("[]\n"),
			"stops.yaml": []byte("- candidate_key: derived-route:cluster-001\n" +
				"  derivation_version: gpx-stops-v1\n" +
				"  label: \"\"\n" +
				"  coord: [135.7681, 35.0116]\n" +
				"  arrive: 2026-04-01T09:00:00+09:00\n" +
				"  depart: 2026-04-01T10:30:00+09:00\n" +
				"  confidence: 0.5\n" +
				"  evidence:\n" +
				"    - kind: route\n" +
				"      source:\n" +
				"        system: local-track\n" +
				"        external_id: derived-route:cluster-001\n" +
				"      locator: derived-route:cluster-001\n" +
				"- candidate_key: derived-route:cluster-002\n" +
				"  derivation_version: gpx-stops-v1\n" +
				"  label: Nishiki Market\n" +
				"  coord: [135.7649, 35.0050]\n" +
				"  arrive: 2026-04-01T12:00:00+09:00\n" +
				"  depart: 2026-04-01T13:00:00+09:00\n" +
				"  confidence: 0.8\n"),
		},
	}
}

// The stops a package carries must reach the intake inbox: the admin GUI lists
// exactly what ListStopCandidatesByJourney returns, so a CLI-imported trip with
// no persisted candidates cannot be curated at all (issue #79).
func TestDecodePackageNormalizesStopCandidates(t *testing.T) {
	document, err := DecodePackage(stopPackage(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Stops) != 2 {
		t.Fatalf("decoded stops = %d, want 2", len(document.Stops))
	}
	first := document.Stops[0]
	if first.JourneyID != document.Journey.ID {
		t.Fatalf("stop journey = %s, want %s", first.JourneyID, document.Journey.ID)
	}
	if first.Identity.DerivationVersion != "gpx-stops-v1" || first.Identity.Key != "derived-route:cluster-001" {
		t.Fatalf("stop identity = %#v", first.Identity)
	}
	if first.Coord != (orb.Point{135.7681, 35.0116}) {
		t.Fatalf("stop coord = %#v", first.Coord)
	}
	if !first.Arrive.Equal(time.Date(2026, 4, 1, 9, 0, 0, 0, time.FixedZone("", 9*3600))) {
		t.Fatalf("stop arrive = %s", first.Arrive)
	}
	if first.Confidence != 0.5 || first.State != domain.CandidateProposed {
		t.Fatalf("stop confidence/state = %v/%q", first.Confidence, first.State)
	}
	if len(first.Evidence) != 1 || first.Evidence[0].Kind != domain.EvidenceRoute || first.Evidence[0].Source.System != "local-track" {
		t.Fatalf("stop evidence = %#v", first.Evidence)
	}
	if len(first.AuthoredFields) != 0 {
		t.Fatalf("import claimed authorship of %v", first.AuthoredFields)
	}
	if document.Stops[1].Label != "Nishiki Market" {
		t.Fatalf("second stop label = %q", document.Stops[1].Label)
	}
	// A package member without its own evidence still has to say where the
	// candidate came from (ADR-0010): derived stops carrying blank provenance
	// was already a defect once (docs/roadmap/user-journey.md, 2026-08-16).
	if len(document.Stops[1].Evidence) != 1 || document.Stops[1].Evidence[0].Source.System != "package:local-kyoto" {
		t.Fatalf("fallback evidence = %#v", document.Stops[1].Evidence)
	}
	if got := document.Stops[1].Evidence[0].Locator; got != "stops.yaml#derived-route:cluster-002" {
		t.Fatalf("fallback evidence locator = %q", got)
	}
}

// Every imported candidate names the package as its source, with a real
// observation time — not a blank Provenance the GUI cannot explain.
func TestDecodePackageStopsCarryPackageProvenance(t *testing.T) {
	document, err := DecodePackage(stopPackage(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, stop := range document.Stops {
		if len(stop.Provenance) != 1 {
			t.Fatalf("stop %s provenance = %#v", stop.Identity.Key, stop.Provenance)
		}
		provenance := stop.Provenance[0]
		if provenance.Source.Ref() != "package:local-kyoto:"+stop.Identity.Key {
			t.Fatalf("stop %s provenance source = %q", stop.Identity.Key, provenance.Source.Ref())
		}
		if err := provenance.Source.Validate(); err != nil {
			t.Fatalf("stop %s provenance source: %v", stop.Identity.Key, err)
		}
		if provenance.ObservedAt.IsZero() || !provenance.ObservedAt.Equal(stop.Arrive) {
			t.Fatalf("stop %s observed_at = %s, want %s", stop.Identity.Key, provenance.ObservedAt, stop.Arrive)
		}
		if provenance.Confidence != stop.Confidence {
			t.Fatalf("stop %s provenance confidence = %v", stop.Identity.Key, provenance.Confidence)
		}
	}
}

func TestDecodePackageRejectsIncompleteStop(t *testing.T) {
	for name, member := range map[string]string{
		"no candidate_key":      "- derivation_version: gpx-stops-v1\n  coord: [135.7, 35.0]\n  arrive: 2026-04-01T09:00:00Z\n  depart: 2026-04-01T10:00:00Z\n",
		"no derivation_version": "- candidate_key: stop-1\n  coord: [135.7, 35.0]\n  arrive: 2026-04-01T09:00:00Z\n  depart: 2026-04-01T10:00:00Z\n",
		"no coord":              "- candidate_key: stop-1\n  derivation_version: gpx-stops-v1\n  arrive: 2026-04-01T09:00:00Z\n  depart: 2026-04-01T10:00:00Z\n",
		"coord out of range":    "- candidate_key: stop-1\n  derivation_version: gpx-stops-v1\n  coord: [200, 35.0]\n  arrive: 2026-04-01T09:00:00Z\n  depart: 2026-04-01T10:00:00Z\n",
		"depart before arrive":  "- candidate_key: stop-1\n  derivation_version: gpx-stops-v1\n  coord: [135.7, 35.0]\n  arrive: 2026-04-01T10:00:00Z\n  depart: 2026-04-01T09:00:00Z\n",
		"confidence above one":  "- candidate_key: stop-1\n  derivation_version: gpx-stops-v1\n  coord: [135.7, 35.0]\n  arrive: 2026-04-01T09:00:00Z\n  depart: 2026-04-01T10:00:00Z\n  confidence: 1.5\n",
	} {
		t.Run(name, func(t *testing.T) {
			pkg := stopPackage(t)
			pkg.Files["stops.yaml"] = []byte(member)
			if _, err := DecodePackage(pkg); err == nil {
				t.Fatal("DecodePackage accepted an invalid stop")
			}
		})
	}
}

// A package with no stops.yaml is still a valid package: the member is
// optional, and its absence must not fail an import (or resurrect stops).
func TestDecodePackageWithoutStopsMember(t *testing.T) {
	pkg := stopPackage(t)
	delete(pkg.Files, "stops.yaml")
	document, err := DecodePackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(document.Stops) != 0 {
		t.Fatalf("stops without a stops.yaml member = %d", len(document.Stops))
	}
}

// Re-import is safe for stop candidates in the same sense ADR-0022/ADR-0033
// make it safe for authored fields: identity keeps the row unique, and the
// author's own review decision — discard, merge, rename — outlives the import
// that seeded the candidate.
func TestApplyPackageStopsAreIdempotentAndPreserveReview(t *testing.T) {
	ctx := context.Background()
	document, err := DecodePackage(stopPackage(t))
	if err != nil {
		t.Fatal(err)
	}
	store := &candidateStore{}

	report, err := ApplyPackage(ctx, document, store)
	if err != nil {
		t.Fatalf("first import: %v", err)
	}
	if report.Stops != 2 || len(store.rows) != 2 {
		t.Fatalf("report.Stops = %d, stored = %d, want 2/2", report.Stops, len(store.rows))
	}

	// The author curates in the GUI: discard one candidate, rename the other.
	discarded, renamed := store.rows[0], store.rows[1]
	store.review(discarded.ID, domain.CandidateIgnored, "")
	store.review(renamed.ID, domain.CandidateKept, "Nishiki Market (morning)")

	if _, err := ApplyPackage(ctx, document, store); err != nil {
		t.Fatalf("re-import: %v", err)
	}
	if len(store.rows) != 2 {
		t.Fatalf("re-import duplicated candidates: %d rows", len(store.rows))
	}
	if discarded.State != domain.CandidateIgnored {
		t.Fatalf("re-import resurrected a discarded candidate: state = %q", discarded.State)
	}
	if renamed.State != domain.CandidateKept || renamed.Label != "Nishiki Market (morning)" {
		t.Fatalf("re-import overwrote the author's review: %q / %q", renamed.State, renamed.Label)
	}
	if !discarded.Arrive.Equal(document.Stops[0].Arrive) {
		t.Fatalf("re-import failed to refresh source-owned fields: %s", discarded.Arrive)
	}
	// The document must survive an apply unchanged, so a caller can validate,
	// report, and then apply the same decoded package.
	if document.Stops[0].State != domain.CandidateProposed || document.Stops[0].Label != "" {
		t.Fatalf("ApplyPackage mutated the decoded document: %#v", document.Stops[0])
	}
}
