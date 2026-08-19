// Package manualyaml is a thin YAML source adapter used by the third-source
// spike. It produces canonical observations and does not write persistence.
package manualyaml

import (
	"fmt"
	"io"
	"time"

	"github.com/paulmach/orb"
	"gopkg.in/yaml.v3"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

type document struct {
	SourceSystem string   `yaml:"source_system"`
	Records      []record `yaml:"records"`
}

type record struct {
	ExternalID string         `yaml:"external_id"`
	Kind       string         `yaml:"kind"`
	ObservedAt string         `yaml:"observed_at"`
	OccurredAt string         `yaml:"occurred_at"`
	OccurredTZ string         `yaml:"occurred_tz"`
	Confidence float64        `yaml:"confidence"`
	Title      string         `yaml:"title"`
	Place      string         `yaml:"place"`
	KindData   map[string]any `yaml:"kind_data"`
	Geom       *geometry      `yaml:"geom"`
}

type geometry struct {
	Type        string `yaml:"type"`
	Coordinates any    `yaml:"coordinates"`
}

// Load reads YAML records into canonical observations. The source system and
// external IDs form a stable identity, including for Felicia-owned manual
// records whose IDs are assigned in the YAML document.
func Load(r io.Reader) ([]domain.Observation, error) {
	var doc document
	if err := yaml.NewDecoder(r).Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode manual YAML: %w", err)
	}
	if doc.SourceSystem == "" {
		return nil, fmt.Errorf("source_system is required")
	}
	observations := make([]domain.Observation, 0, len(doc.Records))
	for i, raw := range doc.Records {
		observation, err := normalizeRecord(doc.SourceSystem, raw)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i+1, err)
		}
		observations = append(observations, observation)
	}
	return observations, nil
}

func normalizeRecord(sourceSystem string, raw record) (domain.Observation, error) {
	if raw.ExternalID == "" || raw.Kind == "" {
		return domain.Observation{}, fmt.Errorf("external_id and kind are required")
	}
	occurredAt, err := time.Parse(time.RFC3339, raw.OccurredAt)
	if err != nil {
		return domain.Observation{}, fmt.Errorf("occurred_at must be RFC3339: %w", err)
	}
	observedAt := occurredAt
	if raw.ObservedAt != "" {
		observedAt, err = time.Parse(time.RFC3339, raw.ObservedAt)
		if err != nil {
			return domain.Observation{}, fmt.Errorf("observed_at must be RFC3339: %w", err)
		}
	}
	if raw.Confidence < 0 || raw.Confidence > 1 {
		return domain.Observation{}, fmt.Errorf("confidence must be between 0 and 1")
	}
	var geom orb.Geometry
	if raw.Geom != nil {
		geom, err = normalizeGeometry(raw.Geom)
		if err != nil {
			return domain.Observation{}, err
		}
	}
	identity := domain.SourceIdentity{System: sourceSystem, ExternalID: raw.ExternalID}
	candidate := domain.MementoCandidate{
		Source: identity, Kind: raw.Kind, OccurredAt: occurredAt, OccurredTZ: raw.OccurredTZ,
		Geom: geom, Title: raw.Title, Place: raw.Place, KindData: raw.KindData,
		Provenance: domain.Provenance{Source: identity, ObservedAt: observedAt, Confidence: raw.Confidence},
	}
	return domain.Observation{
		Kind: domain.ObservationMemento, Source: identity, ObservedAt: observedAt,
		Confidence: raw.Confidence, Payload: candidate,
	}, nil
}

func normalizeGeometry(raw *geometry) (orb.Geometry, error) {
	switch raw.Type {
	case "Point":
		coords, ok := numberSlice(raw.Coordinates)
		if !ok || len(coords) != 2 {
			return nil, fmt.Errorf("point geometry requires two numeric coordinates")
		}
		return orb.Point{coords[0], coords[1]}, nil
	case "LineString":
		rows, ok := raw.Coordinates.([]any)
		if !ok || len(rows) < 2 {
			return nil, fmt.Errorf("linestring geometry requires at least two coordinates")
		}
		line := make(orb.LineString, 0, len(rows))
		for _, row := range rows {
			coords, ok := numberSlice(row)
			if !ok || len(coords) != 2 {
				return nil, fmt.Errorf("linestring coordinates must be numeric pairs")
			}
			line = append(line, orb.Point{coords[0], coords[1]})
		}
		return line, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", raw.Type)
	}
}

func numberSlice(value any) ([]float64, bool) {
	values, ok := value.([]any)
	if !ok {
		return nil, false
	}
	result := make([]float64, 0, len(values))
	for _, value := range values {
		number, ok := value.(float64)
		if !ok {
			return nil, false
		}
		result = append(result, number)
	}
	return result, true
}
