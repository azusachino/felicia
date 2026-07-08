package domain_test

import (
	"os"
	"testing"

	"github.com/azusachino/felicia/internal/domain"
)

// loadTestRegistry loads the transit + live template fixtures.
func loadTestRegistry(t *testing.T) *domain.Registry {
	t.Helper()
	reg, err := domain.LoadRegistry(os.DirFS("testdata"))
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	return reg
}

func TestLoadRegistry(t *testing.T) {
	reg := loadTestRegistry(t)

	transit, ok := reg.Template("transit")
	if !ok {
		t.Fatal("transit template not registered")
	}
	if transit.Anchor != domain.AnchorEdge {
		t.Errorf("transit anchor = %q, want edge", transit.Anchor)
	}
	if len(transit.Fields) != 5 {
		t.Errorf("transit fields = %d, want 5", len(transit.Fields))
	}

	live, ok := reg.Template("live")
	if !ok {
		t.Fatal("live template not registered")
	}
	if live.Anchor != domain.AnchorPoint {
		t.Errorf("live anchor = %q, want point", live.Anchor)
	}

	if _, ok := reg.Template("goshuin"); ok {
		t.Error("unregistered kind goshuin reported as present")
	}
}
