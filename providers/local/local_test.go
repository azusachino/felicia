package local

import (
	"bytes"
	"context"
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

func TestPhotoSourceReadsExifTimestampAndGPSWithoutASidecar(t *testing.T) {
	dir := t.TempDir()
	photo := buildTIFF(nil,
		[]exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")},
		[]exifFieldSpec{
			gpsRef(tagGPSLatitudeRef, "N"),
			rationalField(tagGPSLatitude, degreesToRational(35.0116)),
			gpsRef(tagGPSLongitudeRef, "E"),
			rationalField(tagGPSLongitude, degreesToRational(135.7681)),
		})
	if err := os.WriteFile(filepath.Join(dir, "capture.jpg"), photo, 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSource(dir).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("assets = %#v, %v", assets, err)
	}
	asset := assets[0]
	want := time.Date(2026, 4, 1, 9, 15, 30, 0, time.UTC)
	if !asset.At.Equal(want) {
		t.Fatalf("at = %v, want %v", asset.At, want)
	}
	if asset.Coord == nil || !almostEqual(asset.Coord.Lon(), 135.7681) || !almostEqual(asset.Coord.Lat(), 35.0116) {
		t.Fatalf("coord = %#v", asset.Coord)
	}
}

func TestPhotoSourceSidecarOverridesExifTimestampAndCoord(t *testing.T) {
	dir := t.TempDir()
	photo := buildTIFF(nil,
		[]exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")},
		[]exifFieldSpec{
			gpsRef(tagGPSLatitudeRef, "N"),
			rationalField(tagGPSLatitude, degreesToRational(35.0116)),
			gpsRef(tagGPSLongitudeRef, "E"),
			rationalField(tagGPSLongitude, degreesToRational(135.7681)),
		})
	if err := os.WriteFile(filepath.Join(dir, "capture.jpg"), photo, 0o600); err != nil {
		t.Fatal(err)
	}
	sidecar := filepath.Join(dir, "photos.jsonl")
	// The sidecar disagrees with the EXIF on both fields; the sidecar must win
	// on both, since authors use it to correct a wrong camera clock or GPS fix.
	content := `{"path":"capture.jpg","at":"2026-04-02T10:00:00Z","coord":[139.6917,35.6895]}` + "\n"
	if err := os.WriteFile(sidecar, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSourceWithSidecar(dir, sidecar).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 1 {
		t.Fatalf("assets = %#v, %v", assets, err)
	}
	asset := assets[0]
	want := time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC)
	if !asset.At.Equal(want) {
		t.Fatalf("at = %v, want sidecar value %v (EXIF must not win)", asset.At, want)
	}
	if asset.Coord == nil || !almostEqual(asset.Coord.Lon(), 139.6917) || !almostEqual(asset.Coord.Lat(), 35.6895) {
		t.Fatalf("coord = %#v, want sidecar value (EXIF must not win)", asset.Coord)
	}
}

func TestPhotoSourceDegradesOnUnreadableExifWithoutFailingImport(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "no-exif.jpg"), []byte("not a real jpeg, no exif block"), 0o600); err != nil {
		t.Fatal(err)
	}
	corrupt := buildTIFF(nil, []exifFieldSpec{asciiField(tagDateTimeOriginal, "2026:04:01 09:15:30")}, nil)
	marker := bytes.Index(corrupt, []byte("Exif\x00\x00"))
	corrupt = corrupt[:marker+len("Exif\x00\x00")+8] // cut off mid-IFD0
	if err := os.WriteFile(filepath.Join(dir, "truncated.jpg"), corrupt, 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := NewPhotoSource(dir).FetchAssets(context.Background(), time.Time{}, time.Time{})
	if err != nil || len(assets) != 2 {
		t.Fatalf("assets = %#v, %v", assets, err)
	}
	for _, asset := range assets {
		if !asset.At.IsZero() || asset.Coord != nil {
			t.Fatalf("expected degraded (zero At, nil Coord) for %s, got %#v", asset.ID, asset)
		}
	}
}
