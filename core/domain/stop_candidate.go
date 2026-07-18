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
	DerivationVersion string
	Key               string
}

// StopCandidate is a private, reviewable grouping of route and media evidence.
// It is deliberately separate from Visit evidence and authored Memento data.
type StopCandidate struct {
	ID         uuid.UUID
	JourneyID  uuid.UUID
	Identity   CandidateIdentity
	Label      string
	Coord      orb.Point
	Arrive     time.Time
	Depart     time.Time
	Confidence float64
	Evidence   []EvidenceRef
	State      CandidateState
	MergedInto *uuid.UUID
	Provenance []Provenance
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
