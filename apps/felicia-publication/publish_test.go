package publication

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// keyedMediaSource serves per-key bytes until failOnKey is requested, whose
// Open error simulates a mid-compile media failure.
type keyedMediaSource struct {
	data      map[string][]byte
	failOnKey string
}

func (s *keyedMediaSource) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if key == s.failOnKey {
		return nil, errors.New("injected media failure for " + key)
	}
	data, ok := s.data[key]
	if !ok {
		return nil, errors.New("no fixture media for " + key)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

// readModelWithTwoPhotos builds a read model whose single published memento
// carries two photos, so a compile writes two media artifacts and at least one
// target path already exists when the generation rebuilds.
func readModelWithTwoPhotos() *fakeReadModel {
	model := readModelWithPublishedPhoto("media/a.jpg")
	mementoID := model.mementos[model.journeys[0].ID][0].ID
	model.photos[mementoID] = append(model.photos[mementoID], &domain.MementoPhoto{
		ID: uuid.New(), MementoID: mementoID, ObjectKey: "media/b.jpg", ContentHash: "sha256:fixture", Seq: 2,
	})
	return model
}

// snapshotGeneration maps every regular file under root (slash-separated
// relative path) to its bytes, so a generation can be asserted byte-for-byte.
func snapshotGeneration(t *testing.T, root string) map[string][]byte {
	t.Helper()
	snapshot := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		snapshot[filepath.ToSlash(relative)] = data
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", root, err)
	}
	return snapshot
}

// assertSameGeneration fails unless root's files match want byte-for-byte.
func assertSameGeneration(t *testing.T, root string, want map[string][]byte) {
	t.Helper()
	got := snapshotGeneration(t, root)
	if len(got) != len(want) {
		t.Fatalf("generation has %d files (%v), want %d", len(got), keysOf(got), len(want))
	}
	for name, wantBytes := range want {
		gotBytes, ok := got[name]
		if !ok {
			t.Errorf("file %s disappeared from the published generation", name)
			continue
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("file %s changed in the published generation", name)
		}
	}
}

func keysOf(m map[string][]byte) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	return names
}

func assertNoStagingSiblings(t *testing.T, outRoot string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(outRoot))
	if err != nil {
		t.Fatalf("read parent of output root: %v", err)
	}
	prefix := stagingPrefix(outRoot)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			t.Errorf("stale publish sibling left behind: %s", entry.Name())
		}
	}
}

// TestFailedRebuildKeepsPreviousGenerationByteForByte covers the audit
// acceptance: a failure injected after at least one target path would be
// overwritten must leave the previously published generation readable and
// unchanged.
func TestFailedRebuildKeepsPreviousGenerationByteForByte(t *testing.T) {
	outRoot := filepath.Join(t.TempDir(), "site")
	good := &keyedMediaSource{data: map[string][]byte{
		"media/a.jpg": jpegForTest(t, 1),
		"media/b.jpg": jpegForTest(t, 2),
	}}
	if _, err := Publish(context.Background(), StaticCompiler{}, Input{}, readModelWithTwoPhotos(), good, outRoot); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	previous := snapshotGeneration(t, outRoot)
	if _, ok := previous["media/a.jpg"]; !ok {
		t.Fatalf("fixture setup: first generation lacks media/a.jpg: %v", keysOf(previous))
	}

	// The rebuild fails on the second media path — after media/a.jpg (an
	// existing, would-be-overwritten target) was already written in staging.
	failing := &keyedMediaSource{
		data:      good.data,
		failOnKey: "media/b.jpg",
	}
	if _, err := Publish(context.Background(), StaticCompiler{}, Input{}, readModelWithTwoPhotos(), failing, outRoot); err == nil {
		t.Fatal("second publish = nil error, want the injected media failure")
	}

	assertSameGeneration(t, outRoot, previous)
	assertNoStagingSiblings(t, outRoot)
}

// TestSuccessfulPromotionRemovesStaleContent keeps the manifest-based
// stale-content guarantee working across generation promotion, and preserves
// co-located non-manifest files (the SPA build).
func TestSuccessfulPromotionRemovesStaleContent(t *testing.T) {
	outRoot := filepath.Join(t.TempDir(), "site")
	good := &keyedMediaSource{data: map[string][]byte{
		"media/a.jpg": jpegForTest(t, 1),
		"media/b.jpg": jpegForTest(t, 2),
	}}
	if _, err := Publish(context.Background(), StaticCompiler{}, Input{}, readModelWithTwoPhotos(), good, outRoot); err != nil {
		t.Fatalf("first publish: %v", err)
	}
	previousA := snapshotGeneration(t, outRoot)["media/a.jpg"]
	// A co-located SPA build is not manifest-owned and must survive.
	spa := filepath.Join(outRoot, "index.html")
	if err := os.WriteFile(spa, []byte("<html>spa</html>"), 0o644); err != nil {
		t.Fatalf("seed spa file: %v", err)
	}

	// The rebuild only publishes one photo: media/b.jpg is stale content.
	smaller := &keyedMediaSource{data: map[string][]byte{"media/a.jpg": jpegForTest(t, 3)}}
	model := readModelWithPublishedPhoto("media/a.jpg")
	report, err := Publish(context.Background(), StaticCompiler{}, Input{}, model, smaller, outRoot)
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	// Removed includes media/b.jpg plus gen A's journey JSON (the fixture
	// rebuilds with a fresh journey ID); at minimum the stale photo must go.
	if report.Removed < 1 {
		t.Errorf("report.Removed = %d, want at least the stale media artifact", report.Removed)
	}
	if _, err := os.Stat(filepath.Join(outRoot, "media", "b.jpg")); !os.IsNotExist(err) {
		t.Errorf("stale media/b.jpg survived promotion (stat err = %v)", err)
	}
	manifest := snapshotGeneration(t, outRoot)["api/v1/manifest.json"]
	if bytes.Contains(manifest, []byte("media/b.jpg")) {
		t.Errorf("promoted manifest still lists stale media/b.jpg: %s", manifest)
	}
	if _, err := os.Stat(spa); err != nil {
		t.Errorf("co-located SPA file did not survive promotion: %v", err)
	}
	promotedA := snapshotGeneration(t, outRoot)["media/a.jpg"]
	if len(promotedA) == 0 {
		t.Error("media/a.jpg missing after promotion")
	}
	if bytes.Equal(promotedA, previousA) {
		t.Error("media/a.jpg was not replaced by the new generation's derivative")
	}
	assertNoStagingSiblings(t, outRoot)
}

// TestFirstPublishIntoEmptyRoot covers promotion into a root that does not
// exist yet.
func TestFirstPublishIntoEmptyRoot(t *testing.T) {
	outRoot := filepath.Join(t.TempDir(), "site")
	media := &keyedMediaSource{data: map[string][]byte{"media/a.jpg": jpegForTest(t, 1)}}
	report, err := Publish(context.Background(), StaticCompiler{}, Input{}, readModelWithPublishedPhoto("media/a.jpg"), media, outRoot)
	if err != nil {
		t.Fatalf("first publish: %v", err)
	}
	if report.Removed != 0 {
		t.Errorf("report.Removed = %d, want 0 on first publish", report.Removed)
	}
	for _, name := range []string{"api/v1/manifest.json", "api/v1/site.json", "media/a.jpg"} {
		if _, err := os.Stat(filepath.Join(outRoot, filepath.FromSlash(name))); err != nil {
			t.Errorf("%s missing after first publish: %v", name, err)
		}
	}
	assertNoStagingSiblings(t, outRoot)
}

// TestFailedFirstPublishLeavesNoRoot asserts a failing first attempt does not
// create a partial output root at all.
func TestFailedFirstPublishLeavesNoRoot(t *testing.T) {
	outRoot := filepath.Join(t.TempDir(), "site")
	media := &keyedMediaSource{data: map[string][]byte{"media/a.jpg": jpegForTest(t, 1)}, failOnKey: "media/a.jpg"}
	if _, err := Publish(context.Background(), StaticCompiler{}, Input{}, readModelWithPublishedPhoto("media/a.jpg"), media, outRoot); err == nil {
		t.Fatal("publish = nil error, want the injected media failure")
	}
	if _, err := os.Stat(outRoot); !os.IsNotExist(err) {
		t.Errorf("output root exists after failed first publish (stat err = %v)", err)
	}
	assertNoStagingSiblings(t, outRoot)
}

// jpegForTest produces a distinct valid JPEG per variant (distinct dimensions,
// distinct bytes) so byte-for-byte comparisons can tell generations apart.
func jpegForTest(t *testing.T, variant byte) []byte {
	t.Helper()
	return jpegWithGPSExif(t, gradient(16, 16+int(variant)), 1)
}
