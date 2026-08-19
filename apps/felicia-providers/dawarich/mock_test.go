package dawarich

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestFetchAgainstMockUpstream drives the real client against the Python mock
// (scripts/mock_upstream.py). Gated on MOCK_UPSTREAM_URL, like the DSN-gated DB
// tests. Run: make mock-up, then MOCK_UPSTREAM_URL=http://127.0.0.1:8099 go test.
func TestFetchAgainstMockUpstream(t *testing.T) {
	base := os.Getenv("MOCK_UPSTREAM_URL")
	if base == "" {
		t.Skip("MOCK_UPSTREAM_URL not set; start `make mock-up` and set it")
	}
	key := os.Getenv("MOCK_API_KEY")
	if key == "" {
		key = "mock-key"
	}

	c := New(base, key, nil)
	ctx := context.Background()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	routes, err := c.FetchRoutes(ctx, from, to)
	if err != nil {
		t.Fatalf("FetchRoutes: %v", err)
	}
	if len(routes) != 2 {
		t.Errorf("expected 2 routes across the paginated mock, got %d", len(routes))
	}

	visits, err := c.FetchVisits(ctx, from, to)
	if err != nil {
		t.Fatalf("FetchVisits: %v", err)
	}
	if len(visits) != 2 {
		t.Errorf("expected 2 visits (declined dropped), got %d", len(visits))
	}
}
