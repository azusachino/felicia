package domain_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// station / venue value with a resolved [lon, lat] pair.
func coordVal(name string, lon, lat float64) map[string]any {
	return map[string]any{"name": name, "coords": []any{lon, lat}}
}

func TestValidateForStateAllowsIncompleteDrafts(t *testing.T) {
	reg := loadTestRegistry(t)
	tpl, _ := reg.Template("live")
	if got := domain.ValidateForState(tpl, map[string]any{}, domain.MementoDraft); len(got) != 0 {
		t.Fatalf("draft issues = %+v, want none for missing editable fields", got)
	}
	if got := domain.ValidateForState(tpl, map[string]any{}, domain.MementoPublished); len(got) == 0 {
		t.Fatal("published memento should require complete template data")
	}
	if got := domain.ValidateForState(tpl, map[string]any{}, domain.MementoState("bogus")); len(got) != 1 || got[0].Code != domain.CodeInvalidState {
		t.Fatalf("invalid state issues = %+v", got)
	}
}

func TestValidateMementoGeometry(t *testing.T) {
	tests := []struct {
		name   string
		anchor domain.Anchor
		geom   orb.Geometry
		code   string
	}{
		{name: "point", anchor: domain.AnchorPoint, geom: orb.Point{139.7, 35.6}},
		{name: "edge", anchor: domain.AnchorEdge, geom: orb.LineString{{139.7, 35.6}, {135.5, 34.7}}},
		{name: "out of range", anchor: domain.AnchorPoint, geom: orb.Point{181, 35.6}, code: domain.CodeInvalidCoordinate},
		{name: "wrong anchor", anchor: domain.AnchorPoint, geom: orb.LineString{{139.7, 35.6}, {135.5, 34.7}}, code: domain.CodeAnchorMismatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			issues := domain.ValidateMementoGeometry(tc.anchor, tc.geom)
			if tc.code == "" && len(issues) != 0 {
				t.Fatalf("issues = %+v, want none", issues)
			}
			if tc.code != "" && (len(issues) != 1 || issues[0].Code != tc.code) {
				t.Fatalf("issues = %+v, want %s", issues, tc.code)
			}
		})
	}
}

func TestValidateOccurredTimezone(t *testing.T) {
	if got := domain.ValidateOccurredTimezone("Asia/Tokyo"); len(got) != 0 {
		t.Fatalf("valid timezone issues = %+v", got)
	}
	if got := domain.ValidateOccurredTimezone("not/a-timezone"); len(got) != 1 || got[0].Code != domain.CodeInvalidTimezone {
		t.Fatalf("invalid timezone issues = %+v", got)
	}
}

func money(amount int, currency string) map[string]any {
	return map[string]any{"amount": amount, "currency": currency}
}

func TestValidate(t *testing.T) {
	reg := loadTestRegistry(t)

	tokyo := coordVal("東京", 139.767, 35.681)
	shinOsaka := coordVal("新大阪", 135.500, 34.733)
	budokan := coordVal("日本武道館", 139.7495, 35.6933)

	cases := []struct {
		name string
		kind string
		data map[string]any
		want []domain.Issue
	}{
		{
			name: "transit valid",
			kind: "transit",
			data: map[string]any{
				"operator": "JR東海", "line": "東海道新幹線",
				"from": tokyo, "to": shinOsaka, "fare": money(13870, "JPY"),
			},
			want: nil,
		},
		{
			name: "transit missing operator and endpoints",
			kind: "transit",
			data: map[string]any{"line": "山手線"},
			want: []domain.Issue{
				{Code: domain.CodeAnchorMismatch}, // 0 coord fields, edge needs 2
				{Field: "from", Code: domain.CodeRequiredMissing},
				{Field: "operator", Code: domain.CodeRequiredMissing},
				{Field: "to", Code: domain.CodeRequiredMissing},
			},
		},
		{
			name: "transit only one endpoint resolves",
			kind: "transit",
			data: map[string]any{"operator": "JR東日本", "from": tokyo,
				"to": map[string]any{"name": "水戸"}}, // no coords
			want: []domain.Issue{
				{Code: domain.CodeAnchorMismatch},            // only 1 coord field
				{Field: "to", Code: domain.CodeTypeMismatch}, // station without coords
			},
		},
		{
			name: "transit unknown field",
			kind: "transit",
			data: map[string]any{"operator": "JR", "from": tokyo, "to": shinOsaka, "platform": "14"},
			want: []domain.Issue{{Field: "platform", Code: domain.CodeUnknownField}},
		},
		{
			name: "transit fare bad currency",
			kind: "transit",
			data: map[string]any{"operator": "JR", "from": tokyo, "to": shinOsaka, "fare": money(500, "yen")},
			want: []domain.Issue{{Field: "fare", Code: domain.CodeBadCurrency}},
		},
		{
			name: "transit fare amount not integer",
			kind: "transit",
			data: map[string]any{"operator": "JR", "from": tokyo, "to": shinOsaka,
				"fare": map[string]any{"amount": "500", "currency": "JPY"}},
			want: []domain.Issue{{Field: "fare", Code: domain.CodeTypeMismatch}},
		},
		{
			name: "live valid",
			kind: "live",
			data: map[string]any{
				"artist": "羊文学", "venue": budokan,
				"date": "2026-05-12T19:00:00+09:00", "seat": "アリーナ A-12",
				"setlist_url": "https://www.setlist.fm/x",
			},
			want: nil,
		},
		{
			name: "live missing venue and artist",
			kind: "live",
			data: map[string]any{"date": "2026-05-12T19:00:00+09:00"},
			want: []domain.Issue{
				{Code: domain.CodeAnchorMismatch}, // point needs exactly 1 coord field, has 0
				{Field: "artist", Code: domain.CodeRequiredMissing},
				{Field: "venue", Code: domain.CodeRequiredMissing},
			},
		},
		{
			name: "live bad datetime and url",
			kind: "live",
			data: map[string]any{"artist": "X", "venue": budokan,
				"date": "2026-05-12", "setlist_url": "not-a-url"},
			want: []domain.Issue{
				{Field: "date", Code: domain.CodeTypeMismatch},
				{Field: "setlist_url", Code: domain.CodeTypeMismatch},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tpl, ok := reg.Template(tc.kind)
			if !ok {
				t.Fatalf("kind %q not registered", tc.kind)
			}
			got := domain.Validate(tpl, tc.data)
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Validate() issues mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
