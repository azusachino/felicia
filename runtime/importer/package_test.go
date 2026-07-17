package importer

import (
	"testing"

	journeypackage "github.com/azusachino/felicia/core/package"
)

func TestDecodePackageNormalizesRouteMementoAndMedia(t *testing.T) {
	pkg := &journeypackage.Package{
		Manifest: journeypackage.Manifest{PackageID: "timeline-1"},
		Files: map[string][]byte{
			"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: kyoto\ntitle: Kyoto\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
			"mementos.yaml":    []byte("- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  title: Train\n  place: Kyoto\n  geom: [135.7681, 35.0116]\n  kind_data:\n    operator: JR West\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:ticket\n      seq: 1\n"),
			"route.gpx":        []byte(`<?xml version="1.0"?><gpx><trk><trkseg><trkpt lat="35.0116" lon="135.7681"/><trkpt lat="35.6812" lon="139.7671"/></trkseg></trk></gpx>`),
			"media/ticket.jpg": []byte("image"),
		},
	}

	got, err := DecodePackage(pkg)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Journey.GPSRoute) != 1 || len(got.Journey.GPSRoute[0]) != 2 {
		t.Fatalf("unexpected route: %#v", got.Journey.GPSRoute)
	}
	if len(got.Mementos) != 1 || got.Mementos[0].Kind != "transit" || len(got.Photos) != 1 {
		t.Fatalf("unexpected normalized document: %#v", got)
	}
	if got.Mementos[0].SourceIdentity == nil || got.Mementos[0].SourceIdentity.Ref() != "package:timeline-1:00000000-0000-0000-0000-000000000003" {
		t.Fatalf("missing source identity: %#v", got.Mementos[0].SourceIdentity)
	}
}
