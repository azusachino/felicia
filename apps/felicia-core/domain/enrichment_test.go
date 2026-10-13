package domain

import "testing"

func TestParseLegModeAcceptsCanonicalModes(t *testing.T) {
	for _, raw := range []string{"car", "ferry", "train", "plane", "foot"} {
		mode, err := ParseLegMode(raw)
		if err != nil {
			t.Fatalf("ParseLegMode(%q): %v", raw, err)
		}
		if mode != LegMode(raw) {
			t.Errorf("ParseLegMode(%q) = %q", raw, mode)
		}
	}
}

func TestParseLegModeRejectsUnknownMode(t *testing.T) {
	if _, err := ParseLegMode("hovercraft"); err == nil {
		t.Fatal("expected unknown leg mode to fail loudly")
	}
}

func TestParseLegModeNormalizesLegacyWalk(t *testing.T) {
	mode, err := ParseLegMode("walk")
	if err != nil {
		t.Fatalf("ParseLegMode(walk): %v", err)
	}
	if mode != LegModeFoot {
		t.Errorf("expected walk to normalize to foot, got %q", mode)
	}
}

func TestValidateWeatherRejectsMalformedRecords(t *testing.T) {
	if err := ValidateWeather(Weather{"2026-07-01": {Icon: "sun"}}); err != nil {
		t.Fatalf("valid weather rejected: %v", err)
	}
	if err := ValidateWeather(Weather{"": {Icon: "sun"}}); err == nil {
		t.Error("expected empty date key to fail")
	}
	if err := ValidateWeather(Weather{"2026-07-01": {Label: "no icon"}}); err == nil {
		t.Error("expected missing icon to fail")
	}
}
