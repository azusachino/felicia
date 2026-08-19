package importer

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"strings"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	core "github.com/azusachino/felicia/apps/felicia-core"
	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// DefaultRegistry returns the embedded canonical memento-kind registry used
// by package validation and the local CLI.
func DefaultRegistry() (*domain.Registry, error) {
	kinds, err := fs.Sub(core.KindsFS, "kinds")
	if err != nil {
		return nil, fmt.Errorf("open embedded kinds: %w", err)
	}
	return domain.LoadRegistry(kinds)
}

// ValidatePackageDocument checks the normalized package before any provider
// write. It rejects unknown kinds, malformed kind data, invalid lifecycle
// states, duplicate identities, and geometry that disagrees with the kind
// anchor.
func ValidatePackageDocument(document *PackageDocument, registry *domain.Registry) error {
	if document == nil || document.Journey == nil {
		return fmt.Errorf("package document and journey are required")
	}
	if registry == nil {
		return fmt.Errorf("kind registry is required")
	}
	if document.Journey.ID == uuid.Nil || document.Journey.JournalID == uuid.Nil {
		return fmt.Errorf("journey and journal IDs are required")
	}
	if document.Journey.Slug == "" || document.Journey.Title == "" {
		return fmt.Errorf("journey slug and title are required")
	}
	if document.Journey.DateStart.IsZero() || document.Journey.DateEnd.IsZero() || document.Journey.DateEnd.Before(document.Journey.DateStart) {
		return fmt.Errorf("journey date range is invalid")
	}
	seenMementos := make(map[string]struct{}, len(document.Mementos))
	seenSequences := make(map[int]struct{}, len(document.Mementos))
	seenPhotos := make(map[string]struct{}, len(document.Photos))
	seenStops := make(map[string]struct{}, len(document.Stops))
	for index, stop := range document.Stops {
		if stop == nil || stop.ID == uuid.Nil || stop.JourneyID != document.Journey.ID {
			return fmt.Errorf("stop candidate %d has an invalid journey identity", index+1)
		}
		if stop.Identity.DerivationVersion == "" || stop.Identity.Key == "" {
			return fmt.Errorf("stop candidate %d has an invalid source identity", index+1)
		}
		if _, exists := seenStops[stop.Identity.DerivationVersion+":"+stop.Identity.Key]; exists {
			return fmt.Errorf("stop candidate %s is duplicated", stop.Identity.Key)
		}
		seenStops[stop.Identity.DerivationVersion+":"+stop.Identity.Key] = struct{}{}
		if stop.Coord == (orb.Point{}) || stop.Depart.Before(stop.Arrive) || stop.Confidence < 0 || stop.Confidence > 1 {
			return fmt.Errorf("stop candidate %s has invalid geometry, time range, or confidence", stop.Identity.Key)
		}
	}
	for index, memento := range document.Mementos {
		if memento == nil {
			return fmt.Errorf("memento %d is required", index+1)
		}
		if memento.ID == uuid.Nil || memento.JourneyID != document.Journey.ID {
			return fmt.Errorf("memento %d has an invalid journey identity", index+1)
		}
		if _, exists := seenMementos[memento.ID.String()]; exists {
			return fmt.Errorf("memento %s is duplicated", memento.ID)
		}
		seenMementos[memento.ID.String()] = struct{}{}
		if memento.Seq < 1 {
			return fmt.Errorf("memento %s has an invalid sequence", memento.ID)
		}
		if _, exists := seenSequences[memento.Seq]; exists {
			return fmt.Errorf("memento sequence %d is duplicated", memento.Seq)
		}
		seenSequences[memento.Seq] = struct{}{}
		template, ok := registry.Template(memento.Kind)
		if !ok {
			return fmt.Errorf("memento %s uses unregistered kind %q", memento.ID, memento.Kind)
		}
		state := memento.State
		if state == "" {
			state = domain.MementoCandidateState
		}
		data := map[string]any{}
		if len(memento.KindData) > 0 {
			if err := json.Unmarshal(memento.KindData, &data); err != nil {
				return fmt.Errorf("memento %s kind_data: %w", memento.ID, err)
			}
		}
		if issues := domain.ValidateForState(template, data, state); len(issues) > 0 {
			return fmt.Errorf("memento %s kind_data invalid: %s", memento.ID, formatIssues(issues))
		}
		if !memento.OccurredAt.IsZero() && len(domain.ValidateOccurredTimezone(memento.OccurredTZ)) > 0 {
			return fmt.Errorf("memento %s has an invalid timezone", memento.ID)
		}
		if memento.Geom != nil {
			if issues := domain.ValidateMementoGeometry(template.Anchor, memento.Geom); len(issues) > 0 {
				return fmt.Errorf("memento %s geometry invalid: %s", memento.ID, formatIssues(issues))
			}
		} else if state == domain.MementoAuthored || state == domain.MementoPublished || state == domain.MementoArchived {
			return fmt.Errorf("memento %s requires geometry for state %q", memento.ID, state)
		}
	}
	for _, photo := range document.Photos {
		if photo == nil || photo.ID == uuid.Nil || photo.MementoID == uuid.Nil {
			return fmt.Errorf("photo identity is invalid")
		}
		if _, exists := seenPhotos[photo.ID.String()]; exists {
			return fmt.Errorf("photo %s is duplicated", photo.ID)
		}
		seenPhotos[photo.ID.String()] = struct{}{}
		if _, exists := seenMementos[photo.MementoID.String()]; !exists {
			return fmt.Errorf("photo %s references an unknown memento", photo.ID)
		}
		if photo.ObjectKey == "" || photo.ContentHash == "" || photo.Seq < 1 {
			return fmt.Errorf("photo %s is incomplete", photo.ID)
		}
	}
	return nil
}

func formatIssues(issues []domain.Issue) string {
	values := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Field == "" {
			values = append(values, issue.Code)
		} else {
			values = append(values, issue.Field+":"+issue.Code)
		}
	}
	return strings.Join(values, ", ")
}
