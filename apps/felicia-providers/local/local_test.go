package local

import (
	"context"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGPXSourceNormalizesTimestampedSegments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "journey.gpx")
	content := `<?xml version="1.0"?><gpx><trk><trkseg>
<trkpt lat="35.0116" lon="135.7681"><time>2026-04-01T09:00:00Z</time></trkpt>
<trkpt lat="35.0117" lon="135.7682"><time>2026-04-01T09:20:00Z</time></trkpt>
</trkseg></trk></gpx>`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	source := NewGPXSource(path)
	routes, err := source.FetchRoutes(context.Background(), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || len(routes[0].Points) != 2 {
		t.Fatalf("routes = %#v", routes)
	}
	if routes[0].Line[0].Lon() != 135.7681 || !routes[0].From.Equal(time.Date(2026, 4, 1, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("route = %#v", routes[0])
	}
	if routes[0].Provenance.Source.System != "local-gpx" || routes[0].SourceRef == "" {
		t.Fatalf("route provenance = %#v", routes[0])
	}
}

func TestGPXSourceKeepsUntimestampedGeometryAndRejectsBadCoordinates(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "route.gpx")
	if err := os.WriteFile(path, []byte(`<gpx><rte><rtept lat="35" lon="135"/><rtept lat="35.1" lon="135.1"/></rte></gpx>`), 0o600); err != nil {
		t.Fatal(err)
	}
	routes, err := NewGPXSource(path).FetchRoutes(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(routes) != 1 || !routes[0].From.IsZero() {
		t.Fatalf("untimestamped route = %#v, %v", routes, err)
	}
	if err := os.WriteFile(path, []byte(`<gpx><trk><trkseg><trkpt lat="95" lon="135"/><trkpt lat="35" lon="135"/></trkseg></trk></gpx>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewGPXSource(path).FetchRoutes(context.Background(), time.Time{}, time.Time{}); err == nil {
		t.Fatal("invalid coordinate should fail")
	}
}

func TestPhotoSourceIsDeterministicAndPreservesMissingMetadata(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "nested", "a.jpg"), []byte("jpeg bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored.txt"), []byte("not media"), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSource(dir).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("assets = %#v, %v", assets, err)
	}
	asset := assets[0]
	if asset.ID != "nested/a.jpg" || asset.SourceRef != "local-media:nested/a.jpg" || asset.At != (time.Time{}) || asset.Coord != nil {
		t.Fatalf("asset = %#v", asset)
	}
	if asset.Checksum == "" || asset.MIME != "image/jpeg" || asset.Provider != "local" {
		t.Fatalf("asset metadata = %#v", asset)
	}
}

func TestPhotoSourceAppliesValidSidecarAndSkipsMalformedRecords(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ticket.jpg"), []byte("jpeg bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "photos.jsonl")
	content := "not json\n" + `{"path":"ticket.jpg","at":"2026-04-01T09:10:00Z","coord":[135.1,35.1],"title":"Ticket"}` + "\n" + `{"path":"../private.jpg","at":"2026-04-01T09:10:00Z"}` + "\n"
	if err := os.WriteFile(sidecar, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSourceWithSidecar(dir, sidecar).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("assets = %#v, %v", assets, err)
	}
	if assets[0].At.IsZero() || assets[0].Coord == nil || assets[0].Title != "Ticket" || assets[0].Provenance.Source.System != "local-sidecar" || assets[0].Provenance.Confidence != .9 {
		t.Fatalf("sidecar metadata was not applied: %#v", assets[0])
	}
}

func TestPhotoSourceReadsEXIFTimestampAndSidecarOverridesIt(t *testing.T) {
	dir := t.TempDir()
	photo := filepath.Join(dir, "photo.jpg")
	if err := os.WriteFile(photo, minimalEXIFJPEG("2026:08:02 13:09:35"), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSource(dir).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("EXIF assets = %#v, %v", assets, err)
	}
	if got := assets[0].At.Format("2006-01-02 15:04:05"); got != "2026-08-02 13:09:35" {
		t.Fatalf("EXIF timestamp = %q, asset = %#v", got, assets[0])
	}
	if assets[0].Provenance.Source.System != "local-exif" {
		t.Fatalf("EXIF provenance = %#v", assets[0].Provenance)
	}

	sidecar := filepath.Join(dir, "photos.jsonl")
	if err := os.WriteFile(sidecar, []byte(`{"path":"photo.jpg","at":"2026-08-02T15:12:39+09:00"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err = NewPhotoSourceWithSidecar(dir, sidecar).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("sidecar assets = %#v, %v", assets, err)
	}
	if got := assets[0].At.Format(time.RFC3339); got != "2026-08-02T15:12:39+09:00" || assets[0].Provenance.Source.System != "local-sidecar" {
		t.Fatalf("sidecar did not override EXIF: %#v", assets[0])
	}
}

func minimalEXIFJPEG(timestamp string) []byte {
	value := append([]byte(timestamp), 0)
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0}
	tiff = binary.LittleEndian.AppendUint16(tiff, 0x9003)
	tiff = binary.LittleEndian.AppendUint16(tiff, 2)
	tiff = binary.LittleEndian.AppendUint32(tiff, uint32(len(value)))
	tiff = binary.LittleEndian.AppendUint32(tiff, 26)
	tiff = binary.LittleEndian.AppendUint32(tiff, 0)
	tiff = append(tiff, value...)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	result := []byte{0xff, 0xd8, 0xff, 0xe1}
	result = binary.BigEndian.AppendUint16(result, uint16(len(payload)+2))
	result = append(result, payload...)
	return append(result, 0xff, 0xd9)
}
