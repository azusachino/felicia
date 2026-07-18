package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	journeypackage "github.com/azusachino/felicia/core/package"
)

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
		"media/ticket.jpg",
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

func writeFixturePackage(t *testing.T, filename string) string {
	t.Helper()
	files := map[string][]byte{
		"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: sample\ntitle: Sample journey\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
		"mementos.yaml":    []byte("- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  occurred_tz: Asia/Tokyo\n  state: published\n  title: Train ticket\n  place: Kyoto\n  geom: [135.7681, 35.0116]\n  kind_data:\n    operator: JR West\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:ticket\n      seq: 1\n"),
		"route.gpx":        []byte(`<?xml version="1.0"?><gpx><trk><trkseg><trkpt lat="35.0116" lon="135.7681"/><trkpt lat="35.6812" lon="139.7671"/></trkseg></trk></gpx>`),
		"media/ticket.jpg": []byte("fixture image"),
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
