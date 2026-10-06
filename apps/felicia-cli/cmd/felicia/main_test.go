package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
)

// fixtureJPEG returns a small but real JPEG. The static compiler resizes and
// EXIF-strips every published derivative, so it fails the compile rather than
// emit media it cannot decode — a placeholder string will not do.
func fixtureJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewGray(image.Rect(0, 0, 8, 8)), nil); err != nil {
		t.Fatalf("encode fixture jpeg: %v", err)
	}
	return buf.Bytes()
}

func TestCLIImportAndStaticCompileEndToEnd(t *testing.T) {
	root := t.TempDir()
	packageFile := writeFixturePackage(t, filepath.Join(root, "journey.zip"))
	database := filepath.Join(root, "felicia.sqlite")
	mediaRoot := filepath.Join(root, ".felicia", "media")
	out := filepath.Join(root, "site")

	var report strings.Builder
	if err := execute([]string{"package", "validate", packageFile}, &report); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !strings.Contains(report.String(), "sample-1") {
		t.Fatalf("validation report omitted package ID: %s", report.String())
	}
	if err := execute([]string{"import", "--db", database, "--media-root", mediaRoot, "--apply", packageFile}, io.Discard); err != nil {
		t.Fatalf("import: %v", err)
	}
	var compileReport strings.Builder
	if err := execute([]string{"static", "compile", "--db", database, "--media-root", mediaRoot, "--out", out}, &compileReport); err != nil {
		t.Fatalf("static compile: %v", err)
	}
	for _, relative := range []string{
		"api/v1/journeys.json",
		"api/v1/journeys/00000000-0000-0000-0000-000000000001.json",
		"api/v1/journeys/00000000-0000-0000-0000-000000000001/mementos.json",
		// Derived rather than hardcoded, so this asserts the contract the
		// importer and the installer share instead of one digest literal.
		importer.MediaObjectKey("media/ticket.jpg", importer.MediaDigest(fixtureJPEG(t))),
	} {
		if _, err := os.Stat(filepath.Join(out, relative)); err != nil {
			t.Fatalf("compiled artifact missing %s: %v report=%s", relative, err, compileReport.String())
		}
	}
}

func TestCLIJourneyPlanJSONL(t *testing.T) {
	root := t.TempDir()
	gpx := filepath.Join(root, "route.gpx")
	content := `<?xml version="1.0"?><gpx><trk><trkseg><trkpt lat="35" lon="135"><time>2026-04-01T09:00:00Z</time></trkpt><trkpt lat="35.0001" lon="135.0001"><time>2026-04-01T10:00:00Z</time></trkpt></trkseg></trk></gpx>`
	if err := os.WriteFile(gpx, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	err := execute([]string{"journey", "plan", "--journey", "00000000-0000-0000-0000-000000000001", "--gpx", gpx, "--format", "jsonl"}, &output)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"type":"stop"`) || !strings.Contains(lines[1], `"type":"summary"`) {
		t.Fatalf("unexpected JSONL output: %s", output.String())
	}
}

func TestCLIJourneyPlanFromTimelineWithoutTrack(t *testing.T) {
	root := t.TempDir()
	timeline := filepath.Join(root, "Timeline.json")
	data := `{"placeVisit":{"duration":{"startTimestamp":"2026-04-01T09:00:00Z","endTimestamp":"2026-04-01T10:00:00Z"},"location":{"latitudeE7":356765000,"longitudeE7":1397440000,"placeId":"place-1","name":"Meiji Shrine"}}}`
	if err := os.WriteFile(timeline, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	err := execute([]string{"journey", "plan", "--journey", "00000000-0000-0000-0000-000000000001", "--timeline", timeline}, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"Label": "Meiji Shrine"`) || !strings.Contains(output.String(), `"system": "google-timeline"`) {
		t.Fatalf("timeline visit missing from plan: %s", output.String())
	}
}

func writeFixturePackage(t *testing.T, filename string) string {
	t.Helper()
	// content_hash must be the real digest of the bytes: the importer derives
	// storage identity from the bytes and rejects a package whose declaration
	// disagrees, so a placeholder here would not be a valid package.
	ticket := fixtureJPEG(t)
	files := map[string][]byte{
		"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: sample\ntitle: Sample journey\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
		"mementos.yaml":    []byte("- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  occurred_tz: Asia/Tokyo\n  state: published\n  title: Train ticket\n  place: Kyoto\n  geom: [[135.7681, 35.0116], [139.7671, 35.6812]]\n  kind_data:\n    operator: JR West\n    from: {name: Kyoto, coords: [135.7681, 35.0116]}\n    to: {name: Tokyo, coords: [139.7671, 35.6812]}\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:" + importer.MediaDigest(ticket) + "\n      seq: 1\n"),
		"route.gpx":        []byte(`<?xml version="1.0"?><gpx><trk><trkseg><trkpt lat="35.0116" lon="135.7681"/><trkpt lat="35.6812" lon="139.7671"/></trkseg></trk></gpx>`),
		"media/ticket.jpg": ticket,
	}
	manifest := journeypackage.Manifest{SchemaVersion: journeypackage.CurrentSchemaVersion, PackageID: "sample-1"}
	for name, data := range files {
		digest := sha256.Sum256(data)
		manifest.Files = append(manifest.Files, journeypackage.FileEntry{Path: name, Kind: "fixture", Bytes: int64(len(data)), SHA256: hex.EncodeToString(digest[:])})
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(archive)
	writeZipFile(t, writer, "manifest.yaml", manifestBytes)
	for name, data := range files {
		writeZipFile(t, writer, name, data)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return filename
}

func writeZipFile(t *testing.T, writer *zip.Writer, name string, data []byte) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
}

// TestImportCommitsNoReferencesWhenMediaCannotBeInstalled is the ordering
// contract. Originals are installed before the transaction that references
// them, so a failure to write bytes leaves no committed row claiming they
// exist. Under the previous order the transaction committed first and the
// journal was left pointing at media that had never landed, which nothing
// detects and no retry repairs.
func TestImportCommitsNoReferencesWhenMediaCannotBeInstalled(t *testing.T) {
	root := t.TempDir()
	archive := writeFixturePackage(t, filepath.Join(root, "journey.zip"))
	database := filepath.Join(root, "felicia.sqlite")

	// A regular file where the media root should be: creating any directory
	// beneath it fails, so installation cannot succeed.
	blocked := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("in the way"), 0o600); err != nil {
		t.Fatal(err)
	}

	var report strings.Builder
	err := execute([]string{"import", "--db", database, "--media-root", blocked, "--apply", archive}, &report)
	if err == nil {
		t.Fatal("import must fail when its originals cannot be installed")
	}

	repo, err := sqlite.Open(database)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	defer func() { _ = repo.Close() }()
	if _, err := repo.GetJourney(context.Background(), uuid.MustParse("00000000-0000-0000-0000-000000000001")); err == nil {
		t.Fatal("a journey was committed even though its media never landed")
	}
}

// TestImportIsIdempotentAcrossRuns covers the other half of installing first:
// the path encodes the digest, so a second run finds its originals already
// present and must succeed rather than fail or rewrite them.
func TestImportIsIdempotentAcrossRuns(t *testing.T) {
	root := t.TempDir()
	archive := writeFixturePackage(t, filepath.Join(root, "journey.zip"))
	database := filepath.Join(root, "felicia.sqlite")
	mediaRoot := filepath.Join(root, "media")

	for attempt := 1; attempt <= 2; attempt++ {
		var report strings.Builder
		if err := execute([]string{"import", "--db", database, "--media-root", mediaRoot, "--apply", archive}, &report); err != nil {
			t.Fatalf("import attempt %d: %v", attempt, err)
		}
	}

	installed := importer.MediaObjectKey("media/ticket.jpg", importer.MediaDigest(fixtureJPEG(t)))
	if _, err := os.Stat(filepath.Join(mediaRoot, installed)); err != nil {
		t.Fatalf("original missing after two imports: %v", err)
	}
}

func TestCLIWorkspaceResolutionDefaults(t *testing.T) {
	wsDir := t.TempDir()
	t.Setenv("FELICIA_WORKSPACE", wsDir)

	packageFile := writeFixturePackage(t, filepath.Join(t.TempDir(), "journey.zip"))
	out := filepath.Join(t.TempDir(), "site")

	// 1. import with --apply omitting --db and --media-root
	var importReport strings.Builder
	if err := execute([]string{"import", "--apply", packageFile}, &importReport); err != nil {
		t.Fatalf("import with default workspace failed: %v", err)
	}

	expectedDB := filepath.Join(wsDir, "felicia.sqlite")
	if _, err := os.Stat(expectedDB); err != nil {
		t.Fatalf("expected database at %s, got: %v", expectedDB, err)
	}

	installed := importer.MediaObjectKey("media/ticket.jpg", importer.MediaDigest(fixtureJPEG(t)))
	expectedMedia := filepath.Join(wsDir, "media", installed)
	if _, err := os.Stat(expectedMedia); err != nil {
		t.Fatalf("expected media file at %s, got: %v", expectedMedia, err)
	}

	// 2. static compile omitting --db and --media-root
	var compileReport strings.Builder
	if err := execute([]string{"static", "compile", "--out", out}, &compileReport); err != nil {
		t.Fatalf("static compile with default workspace failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "api/v1/journeys.json")); err != nil {
		t.Fatalf("compiled artifact missing api/v1/journeys.json: %v", err)
	}

	// 3. journey apply omitting --db
	planFile := filepath.Join(t.TempDir(), "plan.json")
	planData := []byte(`{"schema":"v1","version":"draft","journey_id":"00000000-0000-0000-0000-000000000001","source_fingerprint":"sha256:dummy","stops":[]}`)
	if err := os.WriteFile(planFile, planData, 0o600); err != nil {
		t.Fatal(err)
	}
	var applyReport strings.Builder
	if err := execute([]string{"journey", "apply", planFile}, &applyReport); err != nil {
		t.Fatalf("journey apply with default workspace failed: %v", err)
	}
	if !strings.Contains(applyReport.String(), `"mode": "apply"`) {
		t.Fatalf("unexpected apply output: %s", applyReport.String())
	}

	// 4. journey delete omitting --db
	var deleteReport strings.Builder
	journeyID := "00000000-0000-0000-0000-000000000001"
	if err := execute([]string{"journey", "delete", "--journey", journeyID}, &deleteReport); err != nil {
		t.Fatalf("journey delete with default workspace failed: %v", err)
	}
	if !strings.Contains(deleteReport.String(), "deleted journey") {
		t.Fatalf("unexpected delete output: %s", deleteReport.String())
	}
}

func TestCLIJourneyIngestSyntheticTripFolder(t *testing.T) {
	wsDir := t.TempDir()
	tripDir := filepath.Join(t.TempDir(), "2026-10-hakone-trip")
	photosDir := filepath.Join(tripDir, "photos")
	if err := os.MkdirAll(photosDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 1. Write route.gpx
	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <name>Hakone Track</name>
    <trkseg>
      <trkpt lat="35.2323" lon="139.0411">
        <time>2026-10-01T10:00:00Z</time>
      </trkpt>
      <trkpt lat="35.2325" lon="139.0415">
        <time>2026-10-01T10:25:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
	if err := os.WriteFile(filepath.Join(tripDir, "route.gpx"), []byte(gpxContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Write fixture photo
	jpegBytes := fixtureJPEG(t)
	photoFile := filepath.Join(photosDir, "DSC_0001.JPG")
	if err := os.WriteFile(photoFile, jpegBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Write photos.jsonl sidecar
	sidecarContent := `{"filename": "DSC_0001.JPG", "timestamp": "2026-10-01T10:10:00Z", "lat": 35.2323, "lon": 139.0411, "caption": "Hakone Station", "kind": "transit"}` + "\n"
	if err := os.WriteFile(filepath.Join(tripDir, "photos.jsonl"), []byte(sidecarContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 4. Run ingest via CLI
	var out1 strings.Builder
	if err := execute([]string{"journey", "ingest", "--dir", tripDir, "--workspace", wsDir}, &out1); err != nil {
		t.Fatalf("first journey ingest failed: %v", err)
	}

	var res1 struct {
		Mode       string   `json:"mode"`
		JourneyID  string   `json:"journey_id"`
		Slug       string   `json:"slug"`
		Candidates int      `json:"candidates"`
		Mementos   int      `json:"mementos"`
		Photos     int      `json:"photos"`
		Conflicts  []string `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(out1.String()), &res1); err != nil {
		t.Fatalf("decode first ingest output: %v, raw: %s", err, out1.String())
	}

	if res1.Mode != "ingest" {
		t.Errorf("expected mode 'ingest', got %q", res1.Mode)
	}
	if res1.Slug != "2026-10-hakone-trip" {
		t.Errorf("expected slug '2026-10-hakone-trip', got %q", res1.Slug)
	}
	if res1.JourneyID == "" {
		t.Errorf("expected non-empty journey_id")
	}
	if res1.Photos != 1 {
		t.Errorf("expected 1 photo ingested, got %d", res1.Photos)
	}

	// Verify photo is installed in workspace media root
	digest := importer.MediaDigest(jpegBytes)
	installedMedia := filepath.Join(wsDir, "media", "media", digest, "DSC_0001.JPG")
	data, err := os.ReadFile(installedMedia)
	if err != nil {
		t.Fatalf("expected installed photo at %s: %v", installedMedia, err)
	}
	if !bytes.Equal(data, jpegBytes) {
		t.Fatalf("installed photo bytes do not match fixture")
	}

	// 5. Test idempotency (re-ingest same folder)
	var out2 strings.Builder
	if err := execute([]string{"journey", "ingest", "--dir", tripDir, "--workspace", wsDir}, &out2); err != nil {
		t.Fatalf("second journey ingest failed: %v", err)
	}

	var res2 struct {
		Mode       string   `json:"mode"`
		JourneyID  string   `json:"journey_id"`
		Slug       string   `json:"slug"`
		Candidates int      `json:"candidates"`
		Mementos   int      `json:"mementos"`
		Photos     int      `json:"photos"`
		Conflicts  []string `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(out2.String()), &res2); err != nil {
		t.Fatalf("decode second ingest output: %v, raw: %s", err, out2.String())
	}

	if res2.JourneyID != res1.JourneyID {
		t.Errorf("expected re-ingest to reuse journey ID %s, got %s", res1.JourneyID, res2.JourneyID)
	}
	if res2.Photos != 1 {
		t.Errorf("expected 1 photo on re-ingest, got %d", res2.Photos)
	}

	// Query sqlite repo directly to verify no duplicates
	dbPath := filepath.Join(wsDir, "felicia.sqlite")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer func() { _ = repo.Close() }()

	journeys, err := repo.ListJourneys(context.Background())
	if err != nil {
		t.Fatalf("list journeys: %v", err)
	}
	if len(journeys) != 1 {
		t.Errorf("expected exactly 1 journey in DB, found %d", len(journeys))
	}

	// 6. Test with positional dir argument and flag overrides
	tripDir2 := filepath.Join(t.TempDir(), "2026-10-custom")
	if err := os.MkdirAll(tripDir2, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tripDir2, "route.gpx"), []byte(gpxContent), 0o644); err != nil {
		t.Fatal(err)
	}
	var out3 strings.Builder
	if err := execute([]string{"journey", "ingest", tripDir2, "--workspace", wsDir, "--slug", "custom-slug", "--title", "Custom Title", "--place", "Custom Place"}, &out3); err != nil {
		t.Fatalf("positional ingest failed: %v", err)
	}
	var res3 struct {
		Slug string `json:"slug"`
	}
	if err := json.Unmarshal([]byte(out3.String()), &res3); err != nil {
		t.Fatalf("decode output 3: %v", err)
	}
	if res3.Slug != "custom-slug" {
		t.Errorf("expected slug 'custom-slug', got %q", res3.Slug)
	}

	j, err := repo.GetJourneyBySlug(context.Background(), "custom-slug")
	if err != nil {
		t.Fatalf("get custom journey: %v", err)
	}
	if j.Title != "Custom Title" || j.Place != "Custom Place" {
		t.Errorf("expected Title 'Custom Title' and Place 'Custom Place', got title=%q, place=%q", j.Title, j.Place)
	}
}

func TestCLIJourneyIngestAuthorshipProtection(t *testing.T) {
	ctx := context.Background()
	wsDir := t.TempDir()
	tripDir := filepath.Join(t.TempDir(), "2026-10-hakone-trip")
	photosDir := filepath.Join(tripDir, "photos")
	if err := os.MkdirAll(photosDir, 0o755); err != nil {
		t.Fatal(err)
	}

	gpxContent := `<?xml version="1.0" encoding="UTF-8"?>
<gpx version="1.1" creator="test">
  <trk>
    <name>Hakone Track</name>
    <trkseg>
      <trkpt lat="35.2323" lon="139.0411">
        <time>2026-10-01T10:00:00Z</time>
      </trkpt>
      <trkpt lat="35.2325" lon="139.0415">
        <time>2026-10-01T10:25:00Z</time>
      </trkpt>
    </trkseg>
  </trk>
</gpx>`
	if err := os.WriteFile(filepath.Join(tripDir, "route.gpx"), []byte(gpxContent), 0o644); err != nil {
		t.Fatal(err)
	}

	photo1Bytes := fixtureJPEG(t)
	photo1Path := filepath.Join(photosDir, "PXL_20261001_101000.jpg")
	if err := os.WriteFile(photo1Path, photo1Bytes, 0o644); err != nil {
		t.Fatal(err)
	}

	sidecarContent := `{"filename": "PXL_20261001_101000.jpg", "timestamp": "2026-10-01T10:10:00Z", "lat": 35.2323, "lon": 139.0411, "caption": "Hakone Railway", "kind": "transit"}` + "\n"
	if err := os.WriteFile(filepath.Join(tripDir, "photos.jsonl"), []byte(sidecarContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. Initial Ingest via CLI
	var out1 strings.Builder
	if err := execute([]string{"journey", "ingest", "--dir", tripDir, "--workspace", wsDir, "--title", "Original Hakone Trip"}, &out1); err != nil {
		t.Fatalf("first journey ingest failed: %v", err)
	}
	var res1 struct {
		JourneyID string `json:"journey_id"`
		Photos    int    `json:"photos"`
	}
	if err := json.Unmarshal([]byte(out1.String()), &res1); err != nil {
		t.Fatalf("decode output 1: %v", err)
	}
	journeyUUID, err := uuid.Parse(res1.JourneyID)
	if err != nil {
		t.Fatalf("invalid journey UUID %q: %v", res1.JourneyID, err)
	}

	// 2. Simulate human desktop/admin authoring on the SQLite database:
	dbPath := filepath.Join(wsDir, "felicia.sqlite")
	repo, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	journey, err := repo.GetJourney(ctx, journeyUUID)
	if err != nil {
		t.Fatalf("get journey: %v", err)
	}
	humanJourneyTitle := "Human Curated Hakone Adventure"
	journey.Title = humanJourneyTitle
	journey.AuthoredFields = []string{"title"}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("author journey: %v", err)
	}

	mementos, err := repo.ListMementosByJourney(ctx, journeyUUID)
	if err != nil || len(mementos) != 1 {
		t.Fatalf("list mementos: %v (count=%d)", err, len(mementos))
	}
	targetMemento := mementos[0]
	humanMementoTitle := "Handcrafted Memento Title"
	humanEssay := "A wonderfully crafted human essay about the Hakone journey."
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{ID: targetMemento.ID, JourneyID: targetMemento.JourneyID},
		State:   domain.MementoDraft,
	}); err != nil {
		t.Fatalf("transition memento to draft: %v", err)
	}
	if err := repo.ApplyManualMementoPatch(ctx, &domain.ManualMementoPatch{
		Memento: &domain.Memento{
			ID:        targetMemento.ID,
			JourneyID: targetMemento.JourneyID,
			Title:     humanMementoTitle,
			Essay:     &humanEssay,
		},
		Fields: []string{"title", "essay"},
		State:  domain.MementoAuthored,
	}); err != nil {
		t.Fatalf("author memento: %v", err)
	}

	candidates, err := repo.ListStopCandidatesByJourney(ctx, journeyUUID)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("list stop candidates: %v (count=%d)", err, len(candidates))
	}
	targetCandidate := candidates[0]
	curatedStopLabel := "Curated Hakone Railway Stop"
	if err := repo.ApplyStopReview(ctx, &domain.StopReviewPatch{
		CandidateID: targetCandidate.ID,
		State:       domain.CandidateKept,
		Label:       &curatedStopLabel,
	}); err != nil {
		t.Fatalf("review stop candidate: %v", err)
	}
	_ = repo.Close()

	// 3. Add second photo (distinct bytes so it has a distinct SHA-256 digest)
	photo2Bytes := append(fixtureJPEG(t), 0x00)
	photo2Path := filepath.Join(photosDir, "PXL_20261001_102000.jpg")
	if err := os.WriteFile(photo2Path, photo2Bytes, 0o644); err != nil {
		t.Fatal(err)
	}
	sidecarEntry2 := `{"filename": "PXL_20261001_102000.jpg", "timestamp": "2026-10-01T10:20:00Z", "lat": 35.2324, "lon": 139.0413, "caption": "Hakone Scenery", "kind": "transit"}` + "\n"
	sidecarFile, err := os.OpenFile(filepath.Join(tripDir, "photos.jsonl"), os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open sidecar: %v", err)
	}
	if _, err := sidecarFile.WriteString(sidecarEntry2); err != nil {
		_ = sidecarFile.Close()
		t.Fatalf("append sidecar: %v", err)
	}
	if err := sidecarFile.Close(); err != nil {
		t.Fatalf("close sidecar: %v", err)
	}

	// 4. Re-ingest trip folder via CLI
	var out2 strings.Builder
	if err := execute([]string{"journey", "ingest", "--dir", tripDir, "--workspace", wsDir, "--title", "Original Hakone Trip"}, &out2); err != nil {
		t.Fatalf("second journey ingest failed: %v", err)
	}
	var res2 struct {
		JourneyID string   `json:"journey_id"`
		Photos    int      `json:"photos"`
		Conflicts []string `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(out2.String()), &res2); err != nil {
		t.Fatalf("decode output 2: %v", err)
	}

	// 5. Verify preserved human authoring
	repo2, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer func() { _ = repo2.Close() }()

	updatedJourney, err := repo2.GetJourney(ctx, journeyUUID)
	if err != nil {
		t.Fatalf("get updated journey: %v", err)
	}
	if updatedJourney.Title != humanJourneyTitle {
		t.Errorf("journey title = %q, want human edited %q", updatedJourney.Title, humanJourneyTitle)
	}

	updatedMemento, err := repo2.GetMemento(ctx, targetMemento.ID)
	if err != nil {
		t.Fatalf("get updated memento: %v", err)
	}
	if updatedMemento.Title != humanMementoTitle {
		t.Errorf("memento title = %q, want human authored %q", updatedMemento.Title, humanMementoTitle)
	}
	if updatedMemento.Essay == nil || *updatedMemento.Essay != humanEssay {
		t.Errorf("memento essay = %v, want human authored %q", updatedMemento.Essay, humanEssay)
	}
	if updatedMemento.State != domain.MementoAuthored {
		t.Errorf("memento state = %q, want %q", updatedMemento.State, domain.MementoAuthored)
	}

	updatedCandidate, err := repo2.GetStopCandidate(ctx, targetCandidate.ID)
	if err != nil {
		t.Fatalf("get updated stop candidate: %v", err)
	}
	if updatedCandidate.State != domain.CandidateKept {
		t.Errorf("candidate state = %q, want %q", updatedCandidate.State, domain.CandidateKept)
	}
	if updatedCandidate.Label != curatedStopLabel {
		t.Errorf("candidate label = %q, want %q", updatedCandidate.Label, curatedStopLabel)
	}

	journeys, err := repo2.ListJourneys(ctx)
	if err != nil || len(journeys) != 1 {
		t.Errorf("journeys count = %d, want 1", len(journeys))
	}
	allMementos, err := repo2.ListMementosByJourney(ctx, journeyUUID)
	if err != nil || len(allMementos) != 1 {
		t.Errorf("mementos count = %d, want 1", len(allMementos))
	}
	allPhotos, err := repo2.ListPhotosByMemento(ctx, targetMemento.ID)
	if err != nil || len(allPhotos) != 2 {
		t.Errorf("photos count = %d, want 2", len(allPhotos))
	}
}
