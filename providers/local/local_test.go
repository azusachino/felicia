package local

import (
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
