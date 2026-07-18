package contracts

import "testing"

func TestCanonicalVersionAndCapabilityNames(t *testing.T) {
	if CanonicalVersion != "felicia.canonical.v1" {
		t.Fatalf("canonical version = %q", CanonicalVersion)
	}
	if CapabilityMedia != "media.read" || CapabilitySuggestions != "suggestions.propose" {
		t.Fatalf("capability names changed unexpectedly")
	}
}
