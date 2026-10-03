package publication

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// JourneyBuildStatus compares current public projections against the last local
// artifact. It reports never_built, changed or built, not remote deployment state.
// Pending count includes removed rows still in the artifact; IDs highlight live rows.
func JourneyBuildStatus(ctx context.Context, repo domain.Repository, root string, journeyID uuid.UUID) (pendingIDs []string, count int, state string, err error) {
	pendingIDs = []string{}
	_, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(ManifestPath)))
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, 0, "", statErr
	}
	mementos, err := repo.ListMementosByJourney(ctx, journeyID)
	if err != nil {
		return nil, 0, "", err
	}
	if os.IsNotExist(statErr) {
		for _, m := range mementos {
			if m.State == domain.MementoPublished {
				pendingIDs = append(pendingIDs, m.ID.String())
			}
		}
		return pendingIDs, len(pendingIDs), "never_built", nil
	}
	built, err := artifactMementos(root, journeyID)
	if err != nil {
		return nil, 0, "", err
	}
	liveIDs := make(map[string]bool, len(mementos))
	for _, memento := range mementos {
		id := memento.ID.String()
		liveIDs[id] = true
		existing, inArtifact := built[id]
		if memento.State != domain.MementoPublished {
			if inArtifact {
				pendingIDs = append(pendingIDs, id)
			}
			continue
		}
		if !inArtifact {
			pendingIDs = append(pendingIDs, id)
			continue
		}
		photos, photoErr := repo.ListPhotosByMemento(ctx, memento.ID)
		if photoErr != nil {
			return nil, 0, "", photoErr
		}
		projection := make([]StaticPhoto, 0, len(photos))
		for _, photo := range photos {
			projection = append(projection, NewStaticPhoto(photo))
		}
		projected, projectErr := canonicalJSON(NewStaticMemento(memento, projection))
		if projectErr != nil {
			return nil, 0, "", projectErr
		}
		if !bytes.Equal(projected, existing) {
			pendingIDs = append(pendingIDs, id)
		}
	}
	count = len(pendingIDs)
	for id := range built {
		if !liveIDs[id] {
			count++
		}
	}
	if count == 0 {
		return pendingIDs, 0, "built", nil
	}
	return pendingIDs, count, "changed", nil
}

func artifactMementos(root string, journeyID uuid.UUID) (map[string][]byte, error) {
	file := filepath.Join(root, "api", "v1", "journeys", journeyID.String(), "mementos.json")
	raw, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return map[string][]byte{}, nil
	}
	if err != nil {
		return nil, err
	}
	var mementos []json.RawMessage
	if err := json.Unmarshal(raw, &mementos); err != nil {
		return nil, err
	}
	built := make(map[string][]byte, len(mementos))
	for _, entry := range mementos {
		var identified struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(entry, &identified); err != nil {
			return nil, err
		}
		canonical, err := canonicalJSON(entry)
		if err != nil {
			return nil, err
		}
		built[identified.ID] = canonical
	}
	return built, nil
}

// Re-encoding normalizes key order/whitespace, so comparison is by content.
func canonicalJSON(value any) ([]byte, error) {
	var intermediate any
	switch typed := value.(type) {
	case json.RawMessage:
		if err := json.Unmarshal(typed, &intermediate); err != nil {
			return nil, err
		}
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(encoded, &intermediate); err != nil {
			return nil, err
		}
	}
	return json.Marshal(intermediate)
}
