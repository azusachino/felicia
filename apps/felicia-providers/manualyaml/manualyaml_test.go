package manualyaml

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/azusachino/felicia/apps/felicia-core"
	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

func TestLoadMapsManualYAMLToCanonicalObservations(t *testing.T) {
	observations, err := Load(strings.NewReader(`source_system: felicia-yaml
records:
  - external_id: ticket-1
    kind: goods
    occurred_at: 2026-03-20T10:00:00Z
    occurred_tz: Asia/Tokyo
    confidence: 1
    title: 絵はがき
    place: 京都
    geom:
      type: Point
      coordinates: [135.7681, 35.0116]
    kind_data:
      name: 絵はがき
      shop: 駅売店
      price:
        amount: 500
        currency: JPY
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("observations = %d, want 1", len(observations))
	}
	got := observations[0]
	if got.Source.Ref() != "felicia-yaml:ticket-1" || got.Kind != domain.ObservationMemento {
		t.Fatalf("observation = %+v", got)
	}
	candidate, ok := got.Payload.(domain.MementoCandidate)
	if !ok || candidate.Kind != "goods" || candidate.Source != got.Source {
		t.Fatalf("candidate = %#v", got.Payload)
	}
	kinds, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		t.Fatalf("KindsFS: %v", err)
	}
	registry, err := domain.LoadRegistry(kinds)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	template, ok := registry.Template(candidate.Kind)
	if !ok {
		t.Fatalf("template %q not registered", candidate.Kind)
	}
	if issues := domain.Validate(template, candidate.KindData); len(issues) != 0 {
		t.Fatalf("template validation issues = %+v", issues)
	}
}

func TestLoadRejectsUnstableOrMalformedRecords(t *testing.T) {
	_, err := Load(strings.NewReader(`source_system: felicia-yaml
records:
  - kind: goods
    occurred_at: not-a-time
`))
	if err == nil {
		t.Fatal("missing external ID should fail")
	}
	_, err = Load(strings.NewReader(`source_system: felicia-yaml
records:
  - external_id: ticket-1
    kind: goods
    occurred_at: 2026-03-20T10:00:00Z
    confidence: 1.2
`))
	if err == nil {
		t.Fatal("out-of-range confidence should fail")
	}
}
