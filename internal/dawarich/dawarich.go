// Package dawarich is the Dawarich adapter implementing domain.TrackSource:
// routes (GET /api/v1/tracks) and visits (GET /api/v1/visits) for a time range.
package dawarich

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/felicia/internal/domain"
)

// Doer is the subset of *http.Client the client needs; injectable for tests.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client talks to a Dawarich instance.
type Client struct {
	baseURL string
	apiKey  string
	http    Doer
}

var _ domain.TrackSource = (*Client)(nil)

// New builds a Client. A nil doer defaults to http.DefaultClient.
func New(baseURL, apiKey string, doer Doer) *Client {
	if doer == nil {
		doer = http.DefaultClient
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    doer,
	}
}

// FetchRoutes returns the tracks overlapping [from, to] as normalized routes.
func (c *Client) FetchRoutes(ctx context.Context, from, to time.Time) ([]domain.Route, error) {
	var all []domain.Route
	err := c.paginate(ctx, "/api/v1/tracks", timeRange(from, to), func(body []byte) error {
		routes, err := parseTracks(body)
		if err != nil {
			return err
		}
		all = append(all, routes...)
		return nil
	})
	return all, err
}

// FetchVisits returns the visits overlapping [from, to] as derived places.
func (c *Client) FetchVisits(ctx context.Context, from, to time.Time) ([]domain.Visit, error) {
	var all []domain.Visit
	err := c.paginate(ctx, "/api/v1/visits", timeRange(from, to), func(body []byte) error {
		visits, err := parseVisits(body)
		if err != nil {
			return err
		}
		all = append(all, visits...)
		return nil
	})
	return all, err
}

func timeRange(from, to time.Time) url.Values {
	return url.Values{
		"start_at": {from.UTC().Format(time.RFC3339)},
		"end_at":   {to.UTC().Format(time.RFC3339)},
		"per_page": {"500"},
	}
}

// paginate walks every page (X-Total-Pages) and hands each body to handle.
func (c *Client) paginate(ctx context.Context, path string, q url.Values, handle func([]byte) error) error {
	for page := 1; ; page++ {
		q.Set("page", strconv.Itoa(page))
		body, totalPages, err := c.get(ctx, path, q)
		if err != nil {
			return err
		}
		if err := handle(body); err != nil {
			return err
		}
		if page >= totalPages {
			return nil
		}
	}
}

// get issues one authenticated GET and returns the body plus X-Total-Pages.
func (c *Client) get(ctx context.Context, path string, q url.Values) ([]byte, int, error) {
	q.Set("api_key", c.apiKey) // Dawarich accepts api_key as a query param
	u := c.baseURL + path + "?" + q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("dawarich %s: build request: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("dawarich %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("dawarich %s: read body: %w", path, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("dawarich %s: status %d: %s", path, resp.StatusCode, body)
	}

	totalPages := 1
	if n, err := strconv.Atoi(resp.Header.Get("X-Total-Pages")); err == nil && n > 0 {
		totalPages = n
	}
	return body, totalPages, nil
}
