// Package domain is felicia's pure core: no I/O, no network — types and
// functions that hold the model invariants and are exercised by fixture-driven
// tests. The memento-template registry (see docs/research/memento-templates.md,
// backend-stack.md D9) lives here: a kind is a declarative Template, and a
// submitted kind_data blob is validated against it before it ever reaches the DB.
package domain

// Anchor is the one structural axis of a memento kind: whether it sits at a
// single place (Point geom) or spans a from→to (LineString geom that composes
// into the journey's display route). See transit-tickets.md.
type Anchor string

// Anchor values.
const (
	AnchorPoint Anchor = "point"
	AnchorEdge  Anchor = "edge"
)

// FieldType is the closed catalog of kind_data field types (backend-stack.md
// D9). Each type fixes how a value is validated and, on the frontend, its
// default widget and any resolver (station/venue).
type FieldType string

// FieldType values — the closed catalog (D9).
const (
	FieldText     FieldType = "text"
	FieldMoney    FieldType = "money"
	FieldDate     FieldType = "date"     // YYYY-MM-DD
	FieldDatetime FieldType = "datetime" // RFC3339 instant
	FieldStation  FieldType = "station"  // resolves via the bundled station catalog (D4)
	FieldVenue    FieldType = "venue"    // a named place + coord
	FieldURL      FieldType = "url"
	FieldEnum     FieldType = "enum"
)

// coordBearing reports whether a field type carries coordinates, and so counts
// toward the anchor invariant (an edge needs two, a point needs exactly one).
func (t FieldType) coordBearing() bool {
	return t == FieldStation || t == FieldVenue
}

// Field is one declared entry of a kind's kind_data schema.
type Field struct {
	Name         string    `yaml:"name"`
	Type         FieldType `yaml:"type"`
	Required     bool      `yaml:"required"`
	Translatable bool      `yaml:"translatable"` // lifted into the translations sidecar (D3); only meaningful on text
	Values       []string  `yaml:"values"`       // the legal set for FieldEnum
}

// Template is one memento kind declared as data. The same declaration drives
// the admin authoring form, kind_data validation, and the stub render + i18n
// keys — so they cannot drift.
type Template struct {
	Kind   string  `yaml:"kind"`
	Anchor Anchor  `yaml:"anchor"`
	Stub   string  `yaml:"stub"` // frontend component id
	Fields []Field `yaml:"fields"`
}

// field returns the named field and whether it exists.
func (t Template) field(name string) (Field, bool) {
	for _, f := range t.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return Field{}, false
}
