package importer

import (
	"strings"
	"testing"

	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
)

func TestPackageTimezoneDefaults(t *testing.T) {
	for _, test := range []struct {
		name, manifest, geom, explicit, want string
	}{
		{"GPS beats journey fallback", "America/New_York", "[139.6917, 35.6895]", "", "Asia/Tokyo"},
		{"no GPS uses manifest", "America/New_York", "null", "", "America/New_York"},
		{"no GPS uses route", "", "null", "", "Asia/Tokyo"},
		{"explicit UTC is preserved", "", "[139.6917, 35.6895]", "UTC", "UTC"},
		{"edge uses departure", "", "[[-74.006, 40.7128], [139.6917, 35.6895]]", "", "America/New_York"},
	} {
		t.Run(test.name, func(t *testing.T) {
			pkg := timezonePackage(test.geom, test.explicit)
			pkg.Manifest.Timezone = test.manifest
			doc, err := DecodePackage(pkg)
			if err != nil {
				t.Fatal(err)
			}
			if doc.Mementos[0].OccurredTZ != test.want {
				t.Fatalf("zone = %q, want %q", doc.Mementos[0].OccurredTZ, test.want)
			}
			if doc.Mementos[0].OccurredAt.Format("2006-01-02T15:04:05Z07:00") != "2026-07-01T23:30:00Z" {
				t.Fatal("changed occurrence instant")
			}
		})
	}
	pkg := timezonePackage("null", "")
	delete(pkg.Files, "route.gpx")
	doc, err := DecodePackage(pkg)
	if err != nil || doc.Mementos[0].OccurredTZ != "UTC" {
		t.Fatalf("empty journey fallback: doc=%+v err=%v", doc, err)
	}
	pkg.Manifest.Timezone = "not/a-zone"
	if _, err := DecodePackage(pkg); err == nil {
		t.Fatal("accepted invalid manifest timezone")
	}
}

func timezonePackage(geom, explicit string) *journeypackage.Package {
	memento := `[{"id":"00000000-0000-0000-0000-000000000003","kind":"goods","seq":1,"occurred_at":"2026-07-01T23:30:00Z","geom":GEOM,"occurred_tz":"ZONE"}]`
	memento = strings.ReplaceAll(strings.ReplaceAll(memento, "GEOM", geom), "ZONE", explicit)
	return &journeypackage.Package{Manifest: journeypackage.Manifest{PackageID: "timezone-test"}, Files: map[string][]byte{
		"journey.yaml":  []byte(`{"id":"00000000-0000-0000-0000-000000000001","journal_id":"00000000-0000-0000-0000-000000000002","slug":"timezone-test","title":"Test","date_start":"2026-07-01","date_end":"2026-07-02"}`),
		"mementos.yaml": []byte(memento),
		"route.gpx":     []byte(`<gpx><trk><trkseg><trkpt lon="139.6917" lat="35.6895"/><trkpt lon="139.7" lat="35.7"/></trkseg></trk></gpx>`),
	}}
}
