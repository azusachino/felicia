package journeypackage

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestReadValidPackage(t *testing.T) {
	filename := writePackage(t, map[string][]byte{
		"manifest.yaml": nil,
		"journey.yaml":  []byte("slug: kyoto\n"),
		"route.gpx":     []byte("<gpx/>"),
	})
	got, err := Read(filename)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.PackageID != "package-1" || string(got.Files["journey.yaml"]) != "slug: kyoto\n" {
		t.Fatalf("unexpected package: %+v", got.Manifest)
	}
}

func TestReadRejectsChecksumMismatch(t *testing.T) {
	filename := writePackage(t, map[string][]byte{
		"manifest.yaml": nil,
		"journey.yaml":  []byte("slug: kyoto\n"),
	})
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	archive.Close()
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Fatal("test ZIP is empty")
	}
	// The manifest is intentionally constructed with the wrong digest below;
	// this assertion ensures the failure is from validation, not ZIP opening.
	bad := filepath.Join(t.TempDir(), "bad.zip")
	writeZip(t, bad, Manifest{SchemaVersion: CurrentSchemaVersion, PackageID: "package-1", Files: []FileEntry{{Path: "journey.yaml", Kind: "metadata", Bytes: 12, SHA256: "00"}}}, map[string][]byte{"journey.yaml": []byte("slug: kyoto\n")})
	if _, err := Read(bad); err == nil {
		t.Fatal("expected checksum failure")
	}
}

func TestReadRejectsTraversal(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "traversal.zip")
	writeRawZip(t, filename, map[string][]byte{"../journey.yaml": []byte("bad")})
	if _, err := Read(filename); err == nil {
		t.Fatal("expected traversal failure")
	}
}

func writePackage(t *testing.T, members map[string][]byte) string {
	t.Helper()
	journey := members["journey.yaml"]
	route := members["route.gpx"]
	manifest := Manifest{
		SchemaVersion: CurrentSchemaVersion,
		PackageID:     "package-1",
		Files: []FileEntry{
			{Path: "journey.yaml", Kind: "metadata", Bytes: int64(len(journey)), SHA256: digest(journey)},
		},
	}
	if route != nil {
		manifest.Files = append(manifest.Files, FileEntry{Path: "route.gpx", Kind: "route", Bytes: int64(len(route)), SHA256: digest(route)})
	}
	filename := filepath.Join(t.TempDir(), "journey.zip")
	writeZip(t, filename, manifest, map[string][]byte{"journey.yaml": journey, "route.gpx": route})
	return filename
}

func writeZip(t *testing.T, filename string, manifest Manifest, members map[string][]byte) {
	t.Helper()
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeMember(t, writer, "manifest.yaml", manifestBytes)
	for name, data := range members {
		if data != nil {
			writeMember(t, writer, name, data)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeRawZip(t *testing.T, filename string, members map[string][]byte) {
	t.Helper()
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(file)
	for name, data := range members {
		writeMember(t, writer, name, data)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeMember(t *testing.T, writer *zip.Writer, name string, data []byte) {
	t.Helper()
	entry, err := writer.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(data); err != nil {
		t.Fatal(err)
	}
}

func digest(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
