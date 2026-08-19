package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
)

func TestSourceIdentity(t *testing.T) {
	tests := []struct {
		name  string
		id    domain.SourceIdentity
		ref   string
		valid bool
	}{
		{name: "namespaced id", id: domain.SourceIdentity{System: "immich", ExternalID: "asset-42"}, ref: "immich:asset-42", valid: true},
		{name: "missing system", id: domain.SourceIdentity{ExternalID: "asset-42"}, valid: false},
		{name: "missing external id", id: domain.SourceIdentity{System: "immich"}, valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.id.Valid(); got != tt.valid {
				t.Fatalf("Valid() = %v, want %v", got, tt.valid)
			}
			if got := tt.id.Ref(); got != tt.ref {
				t.Errorf("Ref() = %q, want %q", got, tt.ref)
			}
			if err := tt.id.Validate(); (err == nil) != tt.valid {
				t.Errorf("Validate() error = %v, valid = %v", err, tt.valid)
			}
		})
	}
}

func TestObservationEnvelopeKeepsCanonicalPayload(t *testing.T) {
	observed := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	payload := domain.PhotoAsset{ID: "asset-42"}
	obs := domain.Observation{
		Kind:       domain.ObservationPhoto,
		Source:     domain.SourceIdentity{System: "immich", ExternalID: "asset-42"},
		ObservedAt: observed,
		Confidence: 0.98,
		Payload:    payload,
	}

	if obs.Kind != domain.ObservationPhoto || obs.Source.Ref() != "immich:asset-42" {
		t.Fatalf("unexpected observation envelope: %+v", obs)
	}
	if obs.ObservedAt != observed || obs.Confidence != 0.98 {
		t.Errorf("observation metadata changed: %+v", obs)
	}
	if got, ok := obs.Payload.(domain.PhotoAsset); !ok || got.ID != payload.ID {
		t.Errorf("payload = %#v, want canonical photo asset", obs.Payload)
	}
}

func TestObservationSupportsMediaKindsAndMemoryLinks(t *testing.T) {
	link := domain.MemoryLink{EntityType: "memento", EntityID: mustUUID(t), Relation: "attached_to"}
	for _, kind := range []domain.MediaKind{
		domain.MediaImage,
		domain.MediaVideo,
		domain.MediaAudio,
		domain.MediaDocument,
		domain.MediaLink,
		domain.MediaEmbed,
	} {
		t.Run(string(kind), func(t *testing.T) {
			asset := domain.MediaAsset{ID: string(kind) + "-1", Kind: kind, URI: "https://example.com/item", MemoryLinks: []domain.MemoryLink{link}}
			observation := domain.Observation{Kind: domain.ObservationMedia, Payload: asset}
			got, ok := observation.Payload.(domain.MediaAsset)
			if !ok || got.Kind != kind || len(got.MemoryLinks) != 1 {
				t.Fatalf("observation = %#v, want %s media linked to memory", observation, kind)
			}
		})
	}
}

func TestMementoCandidateSeparatesCandidateFromAuthorship(t *testing.T) {
	candidate := domain.MementoCandidate{
		Source: domain.SourceIdentity{System: "manual", ExternalID: "note-1"},
		Kind:   "goods",
		Title:  "駅の絵はがき",
	}

	if candidate.Source.Ref() != "manual:note-1" || candidate.Kind != "goods" {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	if candidate.Provenance.Source.Valid() {
		t.Fatal("candidate provenance should remain independent from source identity")
	}
}

func TestMediaAssetSupportsExternalEmbedWithoutRawHTML(t *testing.T) {
	asset := domain.MediaAsset{
		ID:       "video-42",
		Kind:     domain.MediaEmbed,
		Provider: "youtube",
		EmbedURL: "https://www.youtube.com/embed/video-42",
	}

	if asset.Kind != domain.MediaEmbed || asset.Provider != "youtube" {
		t.Fatalf("unexpected embed asset: %+v", asset)
	}
	if asset.URI != "" {
		t.Fatal("canonical embed assets must not require raw HTML")
	}
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
