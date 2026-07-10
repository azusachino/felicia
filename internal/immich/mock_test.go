package immich

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestFetchAssetsAgainstMockUpstream drives the real client against the Python
// mock (scripts/mock_upstream.py). Gated on MOCK_UPSTREAM_URL.
func TestFetchAssetsAgainstMockUpstream(t *testing.T) {
	base := os.Getenv("MOCK_UPSTREAM_URL")
	if base == "" {
		t.Skip("MOCK_UPSTREAM_URL not set; start `make mock-up` and set it")
	}
	key := os.Getenv("MOCK_API_KEY")
	if key == "" {
		key = "mock-key"
	}

	c := New(base, key, nil)
	assets, err := c.FetchAssets(context.Background(),
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("FetchAssets: %v", err)
	}
	if len(assets) != 2 {
		t.Fatalf("expected 2 assets across the paginated mock, got %d", len(assets))
	}

	var noGPS bool
	for _, a := range assets {
		if a.ID == "asset-2" && a.Coord == nil {
			noGPS = true
		}
	}
	if !noGPS {
		t.Error("expected asset-2 to have no GPS coord")
	}
}
