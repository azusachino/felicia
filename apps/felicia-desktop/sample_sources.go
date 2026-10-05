package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// Only composed for an isolated sample repository: no remote source or credential lookup.
type sampleSources struct{ repo domain.Repository }

func (s sampleSources) FetchVisits(_ context.Context, from, to time.Time) ([]domain.Visit, error) {
	day := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	coords := []orb.Point{{135.7671, 34.9858}, {135.7727, 35.0030}, {135.7780, 35.0120}}
	out := []domain.Visit{}
	for i, coord := range coords {
		arrive := day.Add(time.Duration(9+i*2) * time.Hour)
		depart := arrive.Add(time.Hour)
		if arrive.After(to) || depart.Before(from) {
			continue
		}
		out = append(out, domain.Visit{Coord: coord, Label: fmt.Sprintf("Fictional Kyoto stop %d", i+1), Arrive: arrive, Depart: depart, Confidence: 1, SourceRef: fmt.Sprintf("sample:visit:%d", i)})
	}
	return out, nil
}
func (s sampleSources) FetchRoutes(ctx context.Context, from, to time.Time) ([]domain.Route, error) {
	visits, err := s.FetchVisits(ctx, from, to)
	if err != nil || len(visits) == 0 {
		return []domain.Route{}, err
	}
	route := domain.Route{From: visits[0].Arrive, To: visits[len(visits)-1].Depart, Mode: "walking", SourceRef: "sample:route:kyoto"}
	for _, v := range visits {
		route.Line = append(route.Line, v.Coord)
		route.Points = append(route.Points, domain.TrackPoint{Coord: v.Coord, At: v.Arrive})
	}
	return []domain.Route{route}, nil
}
func (s sampleSources) FetchAssets(ctx context.Context, from, to time.Time) ([]domain.PhotoAsset, error) {
	photos, err := s.repo.ListPhotosByMemento(ctx, uuid.MustParse("0190cbde-f300-7000-8000-000000000010"))
	if err != nil {
		return nil, err
	}
	out := []domain.PhotoAsset{}
	for _, p := range photos {
		at := p.CreatedAt
		if p.TakenAt != nil {
			at = *p.TakenAt
		}
		if at.Before(from) || at.After(to) {
			continue
		}
		coord := orb.Point{135.7671, 34.9858}
		out = append(out, domain.PhotoAsset{ID: p.ID.String(), Kind: domain.MediaImage, At: at, Coord: &coord, Checksum: p.ContentHash, SourceRef: "sample:photo:" + p.ID.String(), URI: p.ObjectKey, MIME: "image/png"})
	}
	return out, nil
}
