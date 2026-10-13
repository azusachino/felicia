package domain

import (
	"fmt"
	"slices"
)

// LegMode is the enforced transport-mode vocabulary for journey legs.
// Free strings are not accepted at intake; an unknown mode fails loudly
// instead of silently meaning car (architecture review finding 4).
type LegMode string

// Canonical transport modes accepted at intake and emitted in publication.
const (
	LegModeCar   LegMode = "car"
	LegModeFerry LegMode = "ferry"
	LegModeTrain LegMode = "train"
	LegModePlane LegMode = "plane"
	LegModeFoot  LegMode = "foot"
)

// legModeWalk is the pre-contract spelling of LegModeFoot that existing
// journeys may already carry in metadata. It is accepted on decode and
// normalized to foot; new intake should use the canonical values.
const legModeWalk LegMode = "walk"

var validLegModes = []LegMode{LegModeCar, LegModeFerry, LegModeTrain, LegModePlane, LegModeFoot, legModeWalk}

// ParseLegMode validates one transport mode and normalizes legacy spellings.
func ParseLegMode(raw string) (LegMode, error) {
	mode := LegMode(raw)
	if !slices.Contains(validLegModes, mode) {
		return "", fmt.Errorf("unknown leg mode %q (want one of car, ferry, train, plane, foot)", raw)
	}
	if mode == legModeWalk {
		return LegModeFoot, nil
	}
	return mode, nil
}

// WeatherDay is one typed daily weather record. Icon is required; the label
// is kept so already-published JSON keeps rendering unchanged, while the
// numeric range and provider details are preserved as typed data for later
// localized rendering (review finding 6 follow-up).
type WeatherDay struct {
	Icon     string   `json:"icon"`
	Label    string   `json:"label,omitempty"`
	Min      *float64 `json:"min,omitempty"`
	Max      *float64 `json:"max,omitempty"`
	Provider string   `json:"provider,omitempty"`
	Code     *int     `json:"code,omitempty"`
	Units    string   `json:"units,omitempty"`
}

// Weather maps an ISO date to its typed daily record.
type Weather map[string]WeatherDay

// ValidateWeather rejects malformed weather records loudly: every day needs a
// date key and an icon, and a day entry without an icon cannot render.
func ValidateWeather(w Weather) error {
	for date, day := range w {
		if date == "" {
			return fmt.Errorf("weather entry with empty date")
		}
		if day.Icon == "" {
			return fmt.Errorf("weather for %s: icon is required", date)
		}
	}
	return nil
}
