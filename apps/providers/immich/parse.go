package immich

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
)

// searchResponse is the subset of Immich SearchResponseDto we consume.
type searchResponse struct {
	Assets struct {
		Items    []assetDTO `json:"items"`
		NextPage *string    `json:"nextPage"` // page-number cursor, null when done
	} `json:"assets"`
}

type assetDTO struct {
	ID            string   `json:"id"`
	Checksum      string   `json:"checksum"`
	Type          string   `json:"type"`
	FileCreatedAt string   `json:"fileCreatedAt"`
	LocalDateTime string   `json:"localDateTime"`
	ExifInfo      *exifDTO `json:"exifInfo"`
}

type exifDTO struct {
	Latitude         *float64 `json:"latitude"`
	Longitude        *float64 `json:"longitude"`
	DateTimeOriginal *string  `json:"dateTimeOriginal"`
}

// parseSearchPage maps one /search/metadata response into normalized photo
// assets and returns the next-page cursor ("" when there are no more pages).
func parseSearchPage(body []byte) ([]domain.PhotoAsset, string, error) {
	var resp searchResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, "", fmt.Errorf("decode search response: %w", err)
	}
	assets := make([]domain.PhotoAsset, 0, len(resp.Assets.Items))
	for _, d := range resp.Assets.Items {
		assets = append(assets, domain.PhotoAsset{
			ID:        d.ID,
			At:        assetTime(d),
			Coord:     assetCoord(d),
			Checksum:  d.Checksum,
			SourceRef: "immich:asset:" + d.ID,
		})
	}
	next := ""
	if resp.Assets.NextPage != nil {
		next = *resp.Assets.NextPage
	}
	return assets, next, nil
}

// assetTime prefers the EXIF capture time, then the local time, then the file
// creation time — the first that parses as RFC3339 (Immich emits fractional Zulu).
func assetTime(d assetDTO) time.Time {
	var candidates []string
	if d.ExifInfo != nil && d.ExifInfo.DateTimeOriginal != nil {
		candidates = append(candidates, *d.ExifInfo.DateTimeOriginal)
	}
	candidates = append(candidates, d.LocalDateTime, d.FileCreatedAt)
	for _, s := range candidates {
		if s == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// assetCoord is the EXIF GPS point, or nil when the photo carries no location.
func assetCoord(d assetDTO) *orb.Point {
	if d.ExifInfo == nil || d.ExifInfo.Latitude == nil || d.ExifInfo.Longitude == nil {
		return nil
	}
	p := orb.Point{*d.ExifInfo.Longitude, *d.ExifInfo.Latitude} // [lng, lat]
	return &p
}
