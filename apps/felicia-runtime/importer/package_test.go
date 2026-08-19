package importer

import (
	"testing"

	journeypackage "github.com/azusachino/felicia/core/journeypackage"
)

func TestDecodePackageNormalizesRouteMementoAndMedia(t *testing.T) {
	pkg := &journeypackage.Package{
		Manifest: journeypackage.Manifest{PackageID: "timeline-1"},
		Files: map[string][]byte{
			"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: kyoto\ntitle: Kyoto\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
			"mementos.yaml":    []byte("- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  title: Train\n  place: Kyoto\n  vendor: JR East\n  essay: A quiet departure.\n  price_amount: 1800\n  price_currency: JPY\n  authored_fields: [title, vendor, essay, price_amount, price_currency]\n  geom: [135.7681, 35.0116]\n  kind_data:\n    operator: JR West\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:ticket\n      seq: 1\n"),
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
	if got.Mementos[0].Vendor == nil || *got.Mementos[0].Vendor != "JR East" || got.Mementos[0].Essay == nil || got.Mementos[0].PriceAmount == nil || *got.Mementos[0].PriceAmount != 1800 || len(got.Mementos[0].AuthoredFields) != 5 {
		t.Fatalf("authored fields were not normalized: %#v", got.Mementos[0])
	}
}
