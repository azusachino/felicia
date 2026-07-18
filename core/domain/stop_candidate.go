package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
)

// CandidateState is the authoring decision for a derived stop candidate.
type CandidateState string

const (
	// CandidateProposed is produced by intake and awaits review.
	CandidateProposed CandidateState = "proposed"
	// CandidateKept is accepted as useful journey structure.
	CandidateKept CandidateState = "kept"
	// CandidateIgnored is intentionally excluded from authoring.
	CandidateIgnored CandidateState = "ignored"
	// CandidateMerged is retained as history after joining another candidate.
	CandidateMerged CandidateState = "merged"
)

// CandidateIdentity is stable across repeated planning of the same source
// material and derivation version. It is not a public place identity.
type CandidateIdentity struct {
	DerivationVersion string `json:"derivation_version"`
	Key               string `json:"key"`
}

// StopCandidate is a private, reviewable grouping of route and media evidence.
// It is deliberately separate from Visit evidence and authored Memento data.
type StopCandidate struct {
	ID             uuid.UUID         `json:"id"`
	JourneyID      uuid.UUID         `json:"journey_id"`
	Identity       CandidateIdentity `json:"identity"`
	Label          string            `json:"label"`
	AuthoredFields []string          `json:"authored_fields,omitempty"`
	Coord          orb.Point         `json:"coord"`
	Arrive         time.Time         `json:"arrive"`
	Depart         time.Time         `json:"depart"`
	Confidence     float64           `json:"confidence"`
	Evidence       []EvidenceRef     `json:"evidence"`
	State          CandidateState    `json:"state"`
	MergedInto     *uuid.UUID        `json:"merged_into,omitempty"`
	Provenance     []Provenance      `json:"provenance,omitempty"`
	Revision       int64             `json:"revision"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// StopReviewPatch records an explicit author decision without changing source
// evidence or pretending the candidate was authored content.
type StopReviewPatch struct {
	CandidateID      uuid.UUID
	State            CandidateState
	Label            *string
	MergedInto       *uuid.UUID
	ExpectedRevision *int64
}
