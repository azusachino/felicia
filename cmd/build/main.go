// Package main implements the command-line entry point for the Static Site Compiler.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/internal/publication"
	"github.com/azusachino/felicia/internal/store/pg"
)

type geoJSONGeometry struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

type staticJourney struct {
	ID             string           `json:"id"`
	JournalID      string           `json:"journal_id"`
	Slug           string           `json:"slug"`
	SourceRef      *string          `json:"source_ref,omitempty"`
	Title          string           `json:"title"`
	Place          string           `json:"place"`
	Country        *string          `json:"country,omitempty"`
	Region         *string          `json:"region,omitempty"`
	DateStart      string           `json:"date_start"`
	DateEnd        string           `json:"date_end"`
	GPSRoute       *geoJSONGeometry `json:"gps_route,omitempty"`
	AuthoredFields []string         `json:"authored_fields"`
	Translations   translationMap   `json:"translations,omitempty"`
}

type staticMemento struct {
	ID            string           `json:"id"`
	JourneyID     string           `json:"journey_id"`
	Kind          string           `json:"kind"`
	Seq           int              `json:"seq"`
	OccurredAt    string           `json:"occurred_at"`
	OccurredTZ    string           `json:"occurred_tz"`
	Geom          *geoJSONGeometry `json:"geom,omitempty"`
	Title         string           `json:"title"`
	Place         string           `json:"place"`
	Vendor        *string          `json:"vendor,omitempty"`
	Essay         *string          `json:"essay,omitempty"`
	PriceAmount   *int64           `json:"price_amount,omitempty"`
	PriceCurrency *string          `json:"price_currency,omitempty"`
	KindData      map[string]any   `json:"kind_data,omitempty"`
	SourceRef     *string          `json:"source_ref,omitempty"`
	Photos        []staticPhoto    `json:"photos,omitempty"`
	Translations  translationMap   `json:"translations,omitempty"`
}

type staticPhoto struct {
	ID          string  `json:"id"`
	MementoID   string  `json:"memento_id"`
	ObjectKey   string  `json:"object_key"`
	ContentHash string  `json:"content_hash"`
	Caption     *string `json:"caption,omitempty"`
	Seq         int     `json:"seq"`
	TakenAt     *string `json:"taken_at,omitempty"`
	SourceRef   *string `json:"source_ref,omitempty"`
}

type translationMap map[string]map[string]any

func buildTranslationMap(translations []*domain.Translation) translationMap {
	m := make(translationMap)
	for _, t := range translations {
		if _, ok := m[t.Lang]; !ok {
			m[t.Lang] = make(map[string]any)
		}
		if strings.HasPrefix(t.Field, "kind_data.") {
			parts := strings.Split(t.Field, ".")
			if len(parts) == 2 {
				kindDataMap, ok := m[t.Lang]["kind_data"].(map[string]any)
				if !ok {
					kindDataMap = make(map[string]any)
					m[t.Lang]["kind_data"] = kindDataMap
				}
				kindDataMap[parts[1]] = t.Value
			} else {
				m[t.Lang][t.Field] = t.Value
			}
		} else {
			m[t.Lang][t.Field] = t.Value
		}
	}
	return m
}

func main() {
	// The build tool writes files to the output dir, so logs go to stderr.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	if err := run(); err != nil {
		slog.Error("static site compilation failed", "err", err)
		os.Exit(1)
	}
}

func run() error {
	dsn := flag.String("dsn", os.Getenv("DATABASE_DSN"), "PostgreSQL 18 connection DSN")
	outDir := flag.String("out", "dist", "Output directory for static files")
	flag.Parse()

	if *dsn == "" {
		return fmt.Errorf("database DSN is required. Set DATABASE_DSN env var or pass -dsn flag")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, *dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	repo := pg.NewRepository(pool)

	slog.Info("starting static site compilation", "out_dir", *outDir)

	// Create directories
	apiDir := filepath.Join(*outDir, "api", "v1")
	if err := os.MkdirAll(apiDir, 0755); err != nil {
		return fmt.Errorf("failed to create api directory: %w", err)
	}

	// 1. Fetch journeys
	journeys, err := repo.ListJourneys(ctx)
	if err != nil {
		return fmt.Errorf("failed to list journeys: %w", err)
	}

	var staticJourneyList []publication.JourneyListItem
	for _, j := range journeys {
		trans, err := repo.ListTranslations(ctx, "journey", j.ID)
		if err != nil {
			return fmt.Errorf("failed to list translations for journey %s: %w", j.ID, err)
		}

		gpsGeom := toGeoJSONGeometry(j.GPSRoute)
		sj := staticJourney{
			ID:             j.ID.String(),
			JournalID:      j.JournalID.String(),
			Slug:           j.Slug,
			SourceRef:      j.SourceRef,
			Title:          j.Title,
			Place:          j.Place,
			Country:        j.Country,
			Region:         j.Region,
			DateStart:      j.DateStart.Format("2006-01-02"),
			DateEnd:        j.DateEnd.Format("2006-01-02"),
			GPSRoute:       gpsGeom,
			AuthoredFields: j.AuthoredFields,
			Translations:   buildTranslationMap(trans),
		}
		// 2. Fetch mementos for this journey
		mementos, err := repo.ListMementosByJourney(ctx, j.ID)
		if err != nil {
			return fmt.Errorf("failed to list mementos for journey %s: %w", j.ID, err)
		}

		var staticMementos []staticMemento
		for _, m := range mementos {
			mTrans, err := repo.ListTranslations(ctx, "memento", m.ID)
			if err != nil {
				return fmt.Errorf("failed to list translations for memento %s: %w", m.ID, err)
			}

			photos, err := repo.ListPhotosByMemento(ctx, m.ID)
			if err != nil {
				return fmt.Errorf("failed to list photos for memento %s: %w", m.ID, err)
			}

			var sPhotos []staticPhoto
			for _, ph := range photos {
				var takenAtStr *string
				if ph.TakenAt != nil {
					tStr := ph.TakenAt.Format(time.RFC3339)
					takenAtStr = &tStr
				}
				sPhotos = append(sPhotos, staticPhoto{
					ID:          ph.ID.String(),
					MementoID:   ph.MementoID.String(),
					ObjectKey:   ph.ObjectKey,
					ContentHash: ph.ContentHash,
					Caption:     ph.Caption,
					Seq:         ph.Seq,
					TakenAt:     takenAtStr,
					SourceRef:   ph.SourceRef,
				})
			}

			var kindData map[string]any
			if len(m.KindData) > 0 {
				if err := json.Unmarshal(m.KindData, &kindData); err != nil {
					slog.Warn("failed to parse kind_data", "memento", m.ID, "err", err)
				}
			}

			mGeom := toGeoJSONGeometry(m.Geom)
			sm := staticMemento{
				ID:            m.ID.String(),
				JourneyID:     m.JourneyID.String(),
				Kind:          m.Kind,
				Seq:           m.Seq,
				OccurredAt:    m.OccurredAt.Format(time.RFC3339),
				OccurredTZ:    m.OccurredTZ,
				Geom:          mGeom,
				Title:         m.Title,
				Place:         m.Place,
				Vendor:        m.Vendor,
				Essay:         m.Essay,
				PriceAmount:   m.PriceAmount,
				PriceCurrency: m.PriceCurrency,
				KindData:      kindData,
				SourceRef:     m.SourceRef,
				Photos:        sPhotos,
				Translations:  buildTranslationMap(mTrans),
			}
			staticMementos = append(staticMementos, sm)
		}
		staticJourneyList = append(staticJourneyList, publication.NewJourneyListItem(j, mementos))

		// Write journey-specific mementos: /api/v1/journeys/<id>/mementos.json
		journeyIDDir := filepath.Join(apiDir, "journeys", j.ID.String())
		if err := os.MkdirAll(journeyIDDir, 0755); err != nil {
			return fmt.Errorf("failed to create journey ID directory: %w", err)
		}

		mementosFilePath := filepath.Join(journeyIDDir, "mementos.json")
		if err := writeJSONFile(mementosFilePath, staticMementos); err != nil {
			return fmt.Errorf("failed to write mementos file: %w", err)
		}

		// Write journey details: /api/v1/journeys/<id>.json
		journeyFilePath := filepath.Join(apiDir, "journeys", fmt.Sprintf("%s.json", j.ID))
		if err := writeJSONFile(journeyFilePath, sj); err != nil {
			return fmt.Errorf("failed to write journey file: %w", err)
		}
	}

	// Write global journeys list: /api/v1/journeys.json
	journeysFilePath := filepath.Join(apiDir, "journeys.json")
	if err := writeJSONFile(journeysFilePath, staticJourneyList); err != nil {
		return fmt.Errorf("failed to write global journeys file: %w", err)
	}

	slog.Info("static site compilation complete", "out_dir", *outDir)
	return nil
}

func toGeoJSONGeometry(geom orb.Geometry) *geoJSONGeometry {
	if geom == nil {
		return nil
	}
	switch g := geom.(type) {
	case orb.Point:
		return &geoJSONGeometry{
			Type:        "Point",
			Coordinates: []float64{g.X(), g.Y()},
		}
	case orb.LineString:
		var coords [][]float64
		for _, pt := range g {
			coords = append(coords, []float64{pt.X(), pt.Y()})
		}
		return &geoJSONGeometry{
			Type:        "LineString",
			Coordinates: coords,
		}
	case orb.MultiLineString:
		var coords [][][]float64
		for _, ls := range g {
			var lsCoords [][]float64
			for _, pt := range ls {
				lsCoords = append(lsCoords, []float64{pt.X(), pt.Y()})
			}
			coords = append(coords, lsCoords)
		}
		return &geoJSONGeometry{
			Type:        "MultiLineString",
			Coordinates: coords,
		}
	default:
		return nil
	}
}

func writeJSONFile(path string, data any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}
