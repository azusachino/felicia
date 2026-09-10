package importer

import (
	"path"
	"strings"
	"testing"

	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
)

func TestDecodePackageNormalizesRouteMementoAndMedia(t *testing.T) {
	pkg := &journeypackage.Package{
		Manifest: journeypackage.Manifest{PackageID: "timeline-1"},
		Files: map[string][]byte{
			"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: kyoto\ntitle: Kyoto\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
			"mementos.yaml":    []byte("- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  occurred_tz: Asia/Tokyo\n  title: Train\n  place: Kyoto\n  vendor: JR East\n  essay: A quiet departure.\n  price_amount: 1800\n  price_currency: JPY\n  authored_fields: [title, vendor, essay, price_amount, price_currency]\n  geom: [[135.7681, 35.0116], [139.7671, 35.6812]]\n  kind_data:\n    operator: JR West\n    from: {name: Kyoto, coords: [135.7681, 35.0116]}\n    to: {name: Tokyo, coords: [139.7671, 35.6812]}\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:6105d6cc76af400325e94d588ce511be5bfdbb73b437dc51eca43917d7a43e3d\n      seq: 1\n"),
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

// TestPackagesWithTheSameMemberNameDoNotCollide is the invariant that makes the
// overwrite impossible. Two journeys can each carry `media/ticket.jpg` holding a
// different photo; while storage identity was the member name, importing the
// second replaced the first's bytes in the shared media root and the published
// site showed the wrong image with no error.
func TestPackagesWithTheSameMemberNameDoNotCollide(t *testing.T) {
	first := packageWithTicket(t, "first-journey", []byte("the first photo"))
	second := packageWithTicket(t, "second-journey", []byte("a different photo"))

	firstDocument, err := DecodePackage(first)
	if err != nil {
		t.Fatalf("decode first: %v", err)
	}
	secondDocument, err := DecodePackage(second)
	if err != nil {
		t.Fatalf("decode second: %v", err)
	}

	firstKey := firstDocument.Photos[0].ObjectKey
	secondKey := secondDocument.Photos[0].ObjectKey
	if firstKey == secondKey {
		t.Fatalf("both packages claim %q, so one overwrites the other", firstKey)
	}
	if base := path.Base(firstKey); base != "ticket.jpg" {
		t.Errorf("member name should survive for legibility, got %q", base)
	}
	if firstDocument.Photos[0].ContentHash == secondDocument.Photos[0].ContentHash {
		t.Error("different bytes must not share a content hash")
	}
}

// TestIdenticalBytesShareOneObjectKey is the other half: content addressing must
// deduplicate rather than merely disambiguate, or the same photo imported twice
// is stored twice.
func TestIdenticalBytesShareOneObjectKey(t *testing.T) {
	same := []byte("identical bytes")
	first, err := DecodePackage(packageWithTicket(t, "one", same))
	if err != nil {
		t.Fatal(err)
	}
	second, err := DecodePackage(packageWithTicket(t, "two", same))
	if err != nil {
		t.Fatal(err)
	}
	if first.Photos[0].ObjectKey != second.Photos[0].ObjectKey {
		t.Errorf("identical bytes stored twice: %q vs %q", first.Photos[0].ObjectKey, second.Photos[0].ObjectKey)
	}
}

// TestDeclaredContentHashMustMatchTheBytes keeps the declaration from choosing
// where bytes land, which is the authority the digest is meant to remove.
func TestDeclaredContentHashMustMatchTheBytes(t *testing.T) {
	pkg := packageWithTicket(t, "liar", []byte("real bytes"))
	pkg.Files["mementos.yaml"] = []byte(strings.Replace(
		string(pkg.Files["mementos.yaml"]),
		"content_hash: sha256:"+MediaDigest([]byte("real bytes")),
		"content_hash: sha256:"+MediaDigest([]byte("bytes it wishes it had")),
		1,
	))
	if _, err := DecodePackage(pkg); err == nil {
		t.Fatal("a package declaring a hash that is not its own bytes must be rejected")
	}
}

// packageWithTicket builds a minimal valid package whose single photo is always
// the member `media/ticket.jpg`, so callers vary only the bytes.
func packageWithTicket(t *testing.T, id string, image []byte) *journeypackage.Package {
	t.Helper()
	mementos := "- id: 00000000-0000-0000-0000-000000000003\n  seq: 1\n  kind: transit\n  occurred_at: 2026-04-01T09:00:00+09:00\n  occurred_tz: Asia/Tokyo\n  title: Train\n  place: Kyoto\n  geom: [[135.7681, 35.0116], [139.7671, 35.6812]]\n  kind_data:\n    operator: JR West\n    from: {name: Kyoto, coords: [135.7681, 35.0116]}\n    to: {name: Tokyo, coords: [139.7671, 35.6812]}\n  photos:\n    - id: 00000000-0000-0000-0000-000000000004\n      path: media/ticket.jpg\n      content_hash: sha256:" + MediaDigest(image) + "\n      seq: 1\n"
	return &journeypackage.Package{
		Manifest: journeypackage.Manifest{PackageID: id},
		Files: map[string][]byte{
			"journey.yaml":     []byte("id: 00000000-0000-0000-0000-000000000001\njournal_id: 00000000-0000-0000-0000-000000000002\nslug: kyoto\ntitle: Kyoto\nplace: Kyoto\ndate_start: 2026-04-01\ndate_end: 2026-04-01\n"),
			"mementos.yaml":    []byte(mementos),
			"media/ticket.jpg": image,
		},
	}
}
