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

func stringPtr(value string) *string {
	return &value
}
