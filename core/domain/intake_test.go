package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestEvidenceRefValidateRequiresStableSourceLocator(t *testing.T) {
	valid := EvidenceRef{
		Kind:    EvidenceMedia,
		Source:  SourceIdentity{System: "immich", ExternalID: "asset-1"},
		Locator: "asset-1",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid evidence: %v", err)
	}

	invalid := valid
	invalid.Locator = ""
	if err := invalid.Validate(); err == nil {
		t.Fatal("evidence without locator should fail validation")
	}
}

func TestStopCandidateSeparatesReviewStateAndEvidence(t *testing.T) {
	stopID := uuid.New()
	candidate := StopCandidate{
		ID:       stopID,
		Identity: CandidateIdentity{DerivationVersion: "gpx-stops-v1", Key: "track-a:segment-2"},
		State:    CandidateProposed,
		Evidence: []EvidenceRef{{Kind: EvidenceRoute, Source: SourceIdentity{System: "gpx", ExternalID: "track-a"}, Locator: "segment-2"}},
	}
	if candidate.State != CandidateProposed || candidate.Identity.Key == "" || len(candidate.Evidence) != 1 {
		t.Fatalf("candidate = %#v", candidate)
	}
	if candidate.Evidence[0].Source.Ref() != "gpx:track-a" {
		t.Fatalf("evidence source = %q", candidate.Evidence[0].Source.Ref())
	}
}
