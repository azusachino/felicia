package pg

import (
	"testing"

	"github.com/azusachino/felicia/internal/domain"
)

func TestMergeMementoFieldsDoesNotTouchUnmaskedAuthoredValues(t *testing.T) {
	current := &domain.Memento{
		Title:          "Authored title",
		Kind:           "goods",
		KindData:       []byte(`{"name":"old"}`),
		AuthoredFields: []string{"title"},
	}
	incoming := &domain.Memento{
		Title:     "Stale title",
		Kind:      "receipt",
		KindData:  []byte(`{"shop":"new"}`),
		SourceRef: stringPtr("immich:asset-1"),
	}

	mergeMementoFields(current, incoming, []string{"kind", "kind_data", "source_ref"})

	if current.Title != "Authored title" {
		t.Errorf("unmasked title = %q, want authored value", current.Title)
	}
	if current.Kind != "receipt" || string(current.KindData) != `{"shop":"new"}` {
		t.Errorf("masked fields were not applied: %+v", current)
	}
	if current.SourceRef == nil || *current.SourceRef != "immich:asset-1" {
		t.Errorf("source ref was not applied: %+v", current.SourceRef)
	}
}

func TestUnionFieldsPreservesExistingOwnership(t *testing.T) {
	got := unionFields([]string{"title", "essay"}, []string{"title", "kind_data"})
	want := []string{"title", "essay", "kind_data"}
	if len(got) != len(want) {
		t.Fatalf("unionFields() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("unionFields()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSourceIdentityFromLegacyRefKeepsNamespacedExternalID(t *testing.T) {
	got, ok := sourceIdentityFromRef("immich:asset:asset-42")
	if !ok {
		t.Fatal("legacy source ref should produce a source identity")
	}
	if got.System != "immich" || got.ExternalID != "asset:asset-42" {
		t.Fatalf("source identity = %+v, want immich / asset:asset-42", got)
	}
}

func TestSourceColumnsDerivesLegacyRefForCanonicalIdentity(t *testing.T) {
	identity := domain.SourceIdentity{System: "dawarich", ExternalID: "visit:7"}
	system, externalID, ref := sourceColumns(&domain.Memento{SourceIdentity: &identity})
	if system == nil || externalID == nil || ref == nil {
		t.Fatal("canonical identity should produce all persistence columns")
	}
	if *system != "dawarich" || *externalID != "visit:7" || *ref != "dawarich:visit:7" {
		t.Fatalf("source columns = %q/%q/%q", *system, *externalID, *ref)
	}
}

func TestMementoStateOrDefaultIsNonVisible(t *testing.T) {
	if got := mementoStateOrDefault(""); got != domain.MementoDraft {
		t.Fatalf("empty state = %q, want draft", got)
	}
	if got := mementoStateOrDefault(domain.MementoPublished); got != domain.MementoPublished {
		t.Fatalf("explicit state = %q, want published", got)
	}
}

func stringPtr(value string) *string {
	return &value
}
