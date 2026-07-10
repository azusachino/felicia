// Package immich is the Immich adapter implementing domain.PhotoSource:
// assets for a time range via POST /api/search/metadata. felicia queries Immich
// directly; Dawarich's own Immich integration is only a how-to reference.
package immich

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/felicia/internal/domain"
)

// Doer is the subset of *http.Client the client needs; injectable for tests.
type Doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client talks to an Immich instance.
type Client struct {
	baseURL string
	apiKey  string
	http    Doer
}

var _ domain.PhotoSource = (*Client)(nil)

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

// FetchAssets returns every image taken within [from, to], following the
// nextPage cursor across pages.
func (c *Client) FetchAssets(ctx context.Context, from, to time.Time) ([]domain.PhotoAsset, error) {
	var all []domain.PhotoAsset
	for page := 1; ; {
		body, err := c.searchPage(ctx, from, to, page)
		if err != nil {
			return nil, err
		}
		assets, next, err := parseSearchPage(body)
		if err != nil {
			return nil, err
		}
		all = append(all, assets...)
		if next == "" {
			return all, nil
		}
		n, err := strconv.Atoi(next)
		if err != nil {
			return nil, fmt.Errorf("immich nextPage %q: %w", next, err)
		}
		page = n
	}
}

func (c *Client) searchPage(ctx context.Context, from, to time.Time, page int) ([]byte, error) {
	reqBody, err := json.Marshal(map[string]any{
		"takenAfter":  from.UTC().Format(time.RFC3339),
		"takenBefore": to.UTC().Format(time.RFC3339),
		"type":        "IMAGE",
		"visibility":  "timeline",
		"withExif":    true,
		"order":       "asc",
		"size":        1000,
		"page":        page,
	})
	if err != nil {
		return nil, fmt.Errorf("immich search: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/search/metadata", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("immich search: build request: %w", err)
	}
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("immich search: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("immich search: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("immich search: status %d: %s", resp.StatusCode, body)
	}
	return body, nil
}
