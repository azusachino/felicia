package domain

import (
	"net/url"
	"regexp"
	"sort"
	"time"
)

// Issue is a single validation failure: a machine code plus the field path it
// applies to ("" for a template-level issue such as an anchor mismatch).
type Issue struct {
	Field string
	Code  string
}

// Issue codes. Stable strings — the API and tests match on these.
const (
	CodeRequiredMissing = "required_missing"
	CodeUnknownField    = "unknown_field"
	CodeTypeMismatch    = "type_mismatch"
	CodeAnchorMismatch  = "anchor_mismatch"
	CodeBadCurrency     = "bad_currency"
)

var (
	dateRE     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	currencyRE = regexp.MustCompile(`^[A-Z]{3}$`)
)

// Validate checks a kind_data blob against its template. It is pure and
// deterministic, and reports *all* issues (not fail-fast), sorted by field then
// code so callers and tests see a stable order. An empty result means valid.
//
// The contract (docs/research/memento-templates.md §"The validation contract"):
//   - closed field set — a key not in the template is unknown_field;
//   - required fields must be present and non-nil;
//   - present values are checked against their field type;
//   - the anchor drives geometry: edge needs ≥2 resolvable coord-bearing fields,
//     point needs exactly one.
func Validate(tpl Template, data map[string]any) []Issue {
	var issues []Issue

	// Required present, and every present value type-checked.
	for _, f := range tpl.Fields {
		v, present := data[f.Name]
		if !present || v == nil {
			if f.Required {
				issues = append(issues, Issue{Field: f.Name, Code: CodeRequiredMissing})
			}
			continue
		}
		issues = append(issues, checkType(f, v)...)
	}

	// Closed field set: any key without a matching field is unknown.
	for k := range data {
		if _, ok := tpl.field(k); !ok {
			issues = append(issues, Issue{Field: k, Code: CodeUnknownField})
		}
	}

	// Anchor ↔ geometry invariant: count coord-bearing fields that are present
	// and actually resolve to coordinates.
	coords := 0
	for _, f := range tpl.Fields {
		if !f.Type.coordBearing() {
			continue
		}
		if v, ok := data[f.Name]; ok && v != nil && hasCoords(v) {
			coords++
		}
	}
	switch tpl.Anchor {
	case AnchorEdge:
		if coords < 2 {
			issues = append(issues, Issue{Code: CodeAnchorMismatch})
		}
	case AnchorPoint:
		if coords != 1 {
			issues = append(issues, Issue{Code: CodeAnchorMismatch})
		}
	}

	sort.Slice(issues, func(i, j int) bool {
		if issues[i].Field != issues[j].Field {
			return issues[i].Field < issues[j].Field
		}
		return issues[i].Code < issues[j].Code
	})
	return issues
}

// checkType validates a present, non-nil value against its field's type.
func checkType(f Field, v any) []Issue {
	mismatch := []Issue{{Field: f.Name, Code: CodeTypeMismatch}}
	switch f.Type {
	case FieldText:
		if _, ok := v.(string); !ok {
			return mismatch
		}
	case FieldURL:
		s, ok := v.(string)
		if !ok {
			return mismatch
		}
		u, err := url.Parse(s)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return mismatch
		}
	case FieldDate:
		s, ok := v.(string)
		if !ok || !dateRE.MatchString(s) {
			return mismatch
		}
		if _, err := time.Parse("2006-01-02", s); err != nil {
			return mismatch
		}
	case FieldDatetime:
		s, ok := v.(string)
		if !ok {
			return mismatch
		}
		if _, err := time.Parse(time.RFC3339, s); err != nil {
			return mismatch
		}
	case FieldEnum:
		s, ok := v.(string)
		if !ok || !contains(f.Values, s) {
			return mismatch
		}
	case FieldMoney:
		m, ok := v.(map[string]any)
		if !ok {
			return mismatch
		}
		if _, ok := asInt(m["amount"]); !ok {
			return mismatch
		}
		cur, ok := m["currency"].(string)
		if !ok {
			return mismatch
		}
		if !currencyRE.MatchString(cur) {
			return []Issue{{Field: f.Name, Code: CodeBadCurrency}}
		}
	case FieldStation, FieldVenue:
		m, ok := v.(map[string]any)
		if !ok {
			return mismatch
		}
		if _, ok := m["name"].(string); !ok {
			return mismatch
		}
		if !hasCoords(v) {
			return mismatch
		}
	}
	return nil
}

// hasCoords reports whether a value is a map carrying a [lon, lat] coord pair
// under "coords". Coordinate resolution (station catalog / venue picker) happens
// at the edge; by validation time the resolved pair must be present.
func hasCoords(v any) bool {
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	c, ok := m["coords"].([]any)
	if !ok || len(c) != 2 {
		return false
	}
	for _, n := range c {
		if !isNumber(n) {
			return false
		}
	}
	return true
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

// asInt accepts the numeric shapes JSON/YAML decoders produce for an integer
// (int, int64, float64 with no fractional part).
func asInt(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		if n == float64(int64(n)) {
			return int64(n), true
		}
	}
	return 0, false
}

func isNumber(v any) bool {
	switch v.(type) {
	case int, int64, float64:
		return true
	}
	return false
}
