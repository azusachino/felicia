package domain

import "fmt"

// EvidenceKind identifies the source material that supports a draft candidate.
type EvidenceKind string

const (
	// EvidenceRoute points at a normalized route or a source route segment.
	EvidenceRoute EvidenceKind = "route"
	// EvidenceVisit points at a normalized visit supplied or derived by intake.
	EvidenceVisit EvidenceKind = "visit"
	// EvidenceMedia points at a normalized media asset.
	EvidenceMedia EvidenceKind = "media"
)

// EvidenceRef is a stable, explainable link from a candidate to source material.
// Locator identifies a segment, record, or asset within the source identity.
type EvidenceRef struct {
	Kind    EvidenceKind   `json:"kind"`
	Source  SourceIdentity `json:"source"`
	Locator string         `json:"locator"`
}

// Validate checks the minimum identity required for a durable evidence link.
func (e EvidenceRef) Validate() error {
	if e.Kind == "" {
		return fmt.Errorf("evidence kind is required")
	}
	if err := e.Source.Validate(); err != nil {
		return fmt.Errorf("evidence source: %w", err)
	}
	if e.Locator == "" {
		return fmt.Errorf("evidence locator is required")
	}
	return nil
}
