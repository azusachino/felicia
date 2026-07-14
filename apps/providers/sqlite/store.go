// Package sqlite implements Felicia's local SQLite provider.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	_ "modernc.org/sqlite" // register the SQLite database driver

	"github.com/azusachino/felicia/apps/core/domain"
	"github.com/azusachino/felicia/apps/core/ports"
)

//go:embed schema.sql
var schemaFS embed.FS

var _ domain.Repository = (*Repository)(nil)
var _ domain.ObservationStore = (*Repository)(nil)
var _ ports.JournalStore = (*Repository)(nil)
var _ ports.JourneyStore = (*Repository)(nil)
var _ ports.MementoStore = (*Repository)(nil)
var _ ports.MediaStore = (*Repository)(nil)
var _ ports.RouteStore = (*Repository)(nil)
var _ ports.ObservationStore = (*Repository)(nil)

// Repository persists the canonical model in a local SQLite database.
type Repository struct {
	db *sql.DB
}

// Open opens a SQLite file. Use ":memory:" for isolated tests.
func Open(path string) (*Repository, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	r := &Repository{db: db}
	if _, err := db.ExecContext(context.Background(), "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
	}
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("read sqlite schema: %w", err)
	}
	if _, err := db.ExecContext(context.Background(), string(schema)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("apply sqlite schema: %w", err)
	}
	return r, nil
}

// Close releases the database handle.
func (r *Repository) Close() error { return r.db.Close() }

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

func idString(id uuid.UUID) string { return id.String() }

func parseID(value string) (uuid.UUID, error) { return uuid.Parse(value) }

func marshalJSON(value any) (string, error) {
	b, err := json.Marshal(value)
	return string(b), err
}

func unmarshalJSON(value string, dst any) error {
	if value == "" {
		return nil
	}
	return json.Unmarshal([]byte(value), dst)
}

func stringsJSON(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	return marshalJSON(values)
}

func parseStrings(value string) ([]string, error) {
	var values []string
	if err := unmarshalJSON(value, &values); err != nil {
		return nil, err
	}
	return values, nil
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func readTime(value sql.NullString) time.Time {
	if !value.Valid || value.String == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339Nano, value.String)
	return t
}

func readString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

type geometryValue struct {
	Type        string `json:"type"`
	Coordinates any    `json:"coordinates"`
}

func encodeGeometry(geom orb.Geometry) (any, error) {
	if geom == nil {
		return nil, nil
	}
	var value geometryValue
	switch g := geom.(type) {
	case orb.Point:
		value = geometryValue{Type: "Point", Coordinates: []float64{g.X(), g.Y()}}
	case orb.LineString:
		coords := make([][]float64, 0, len(g))
		for _, point := range g {
			coords = append(coords, []float64{point.X(), point.Y()})
		}
		value = geometryValue{Type: "LineString", Coordinates: coords}
	case orb.MultiLineString:
		coords := make([][][]float64, 0, len(g))
		for _, line := range g {
			lineCoords := make([][]float64, 0, len(line))
			for _, point := range line {
				lineCoords = append(lineCoords, []float64{point.X(), point.Y()})
			}
			coords = append(coords, lineCoords)
		}
		value = geometryValue{Type: "MultiLineString", Coordinates: coords}
	default:
		return nil, fmt.Errorf("unsupported geometry %T", geom)
	}
	return marshalJSON(value)
}

func decodeGeometry(value sql.NullString) (orb.Geometry, error) {
	if !value.Valid || value.String == "" || value.String == "null" {
		return nil, nil
	}
	var raw geometryValue
	if err := unmarshalJSON(value.String, &raw); err != nil {
		return nil, err
	}
	switch raw.Type {
	case "Point":
		var coords []float64
		if err := json.Unmarshal(mustJSON(raw.Coordinates), &coords); err != nil || len(coords) < 2 {
			return nil, fmt.Errorf("invalid point geometry")
		}
		return orb.Point{coords[0], coords[1]}, nil
	case "LineString":
		var coords [][]float64
		if err := json.Unmarshal(mustJSON(raw.Coordinates), &coords); err != nil {
			return nil, fmt.Errorf("invalid linestring geometry: %w", err)
		}
		line := make(orb.LineString, 0, len(coords))
		for _, coord := range coords {
			if len(coord) >= 2 {
				line = append(line, orb.Point{coord[0], coord[1]})
			}
		}
		return line, nil
	case "MultiLineString":
		var coords [][][]float64
		if err := json.Unmarshal(mustJSON(raw.Coordinates), &coords); err != nil {
			return nil, fmt.Errorf("invalid multilinestring geometry: %w", err)
		}
		mls := make(orb.MultiLineString, 0, len(coords))
		for _, lineCoords := range coords {
			line := make(orb.LineString, 0, len(lineCoords))
			for _, coord := range lineCoords {
				if len(coord) >= 2 {
					line = append(line, orb.Point{coord[0], coord[1]})
				}
			}
			mls = append(mls, line)
		}
		return mls, nil
	default:
		return nil, fmt.Errorf("unsupported geometry type %q", raw.Type)
	}
}

func mustJSON(value any) []byte {
	b, _ := json.Marshal(value)
	return b
}

// GetJournal retrieves a journal by ID.
func (r *Repository) GetJournal(ctx context.Context, id uuid.UUID) (*domain.Journal, error) {
	var created string
	err := r.db.QueryRowContext(ctx, "SELECT created_at FROM tb_journals WHERE id = ?", idString(id)).Scan(&created)
	if err != nil {
		return nil, fmt.Errorf("get journal %s: %w", id, err)
	}
	t, _ := time.Parse(time.RFC3339Nano, created)
	return &domain.Journal{ID: id, CreatedAt: t}, nil
}

// CreateJournal inserts a journal.
func (r *Repository) CreateJournal(ctx context.Context, journal *domain.Journal) error {
	if journal.ID == uuid.Nil {
		journal.ID = uuid.Must(uuid.NewV7())
	}
	if journal.CreatedAt.IsZero() {
		journal.CreatedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, "INSERT INTO tb_journals(id, created_at) VALUES (?, ?)", idString(journal.ID), journal.CreatedAt.Format(time.RFC3339Nano))
	return err
}

// ResetMockJournal deletes a journal and its dependent mock data.
func (r *Repository) ResetMockJournal(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tb_journals WHERE id = ?", idString(id))
	return err
}

// GetJourney retrieves a journey by ID.
func (r *Repository) GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error) {
	row := r.db.QueryRowContext(ctx, `SELECT journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at FROM tb_journeys WHERE id = ?`, idString(id))
	var rawJournalID, slug, title, place, start, end, route, fields, created, updated string
	var sourceRef, country, region sql.NullString
	if err := row.Scan(&rawJournalID, &slug, &sourceRef, &title, &place, &country, &region, &start, &end, &route, &fields, &created, &updated); err != nil {
		return nil, err
	}
	journalID, err := parseID(rawJournalID)
	if err != nil {
		return nil, err
	}
	return makeJourney(id, journalID, slug, sourceRef, title, place, country, region, start, end, route, fields, created, updated)
}

// GetJourneyBySlug retrieves a journey by its public slug.
func (r *Repository) GetJourneyBySlug(ctx context.Context, slug string) (*domain.Journey, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at FROM tb_journeys WHERE slug = ?`, slug)
	var rawID string
	var rawJourneyID string
	var sourceRef, country, region sql.NullString
	var start, end, created, updated string
	var route, fields string
	var title, place string
	if err := row.Scan(&rawID, &rawJourneyID, &slug, &sourceRef, &title, &place, &country, &region, &start, &end, &route, &fields, &created, &updated); err != nil {
		return nil, err
	}
	id, err := parseID(rawID)
	if err != nil {
		return nil, err
	}
	journeyID, err := parseID(rawJourneyID)
	if err != nil {
		return nil, err
	}
	return makeJourney(id, journeyID, slug, sourceRef, title, place, country, region, start, end, route, fields, created, updated)
}

// ListJourneys retrieves journeys ordered by start date.
func (r *Repository) ListJourneys(ctx context.Context) ([]*domain.Journey, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at FROM tb_journeys ORDER BY date_start DESC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var result []*domain.Journey
	for rows.Next() {
		var rawID string
		var rawJournalID string
		var slug, title, place, start, end, route, fields, created, updated string
		var sourceRef, country, region sql.NullString
		if err := rows.Scan(&rawID, &rawJournalID, &slug, &sourceRef, &title, &place, &country, &region, &start, &end, &route, &fields, &created, &updated); err != nil {
			return nil, err
		}
		parsed, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		journalID, err := parseID(rawJournalID)
		if err != nil {
			return nil, err
		}
		journey, err := makeJourney(parsed, journalID, slug, sourceRef, title, place, country, region, start, end, route, fields, created, updated)
		if err != nil {
			return nil, err
		}
		result = append(result, journey)
	}
	return result, rows.Err()
}

// UpsertJourney inserts or updates a journey.
func (r *Repository) UpsertJourney(ctx context.Context, journey *domain.Journey) error {
	route, err := marshalJSON(journey.GPSRoute)
	if err != nil {
		return err
	}
	fields, err := stringsJSON(journey.AuthoredFields)
	if err != nil {
		return err
	}
	if journey.CreatedAt.IsZero() {
		journey.CreatedAt = time.Now().UTC()
	}
	if journey.UpdatedAt.IsZero() {
		journey.UpdatedAt = journey.CreatedAt
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO tb_journeys(id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET slug=excluded.slug, source_ref=excluded.source_ref, title=excluded.title, place=excluded.place, country=excluded.country, region=excluded.region, date_start=excluded.date_start, date_end=excluded.date_end, gps_route=excluded.gps_route, authored_fields=excluded.authored_fields, updated_at=excluded.updated_at`, idString(journey.ID), idString(journey.JournalID), journey.Slug, nullableString(journey.SourceRef), journey.Title, journey.Place, nullableString(journey.Country), nullableString(journey.Region), journey.DateStart.Format("2006-01-02"), journey.DateEnd.Format("2006-01-02"), route, fields, journey.CreatedAt.Format(time.RFC3339Nano), journey.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

type scanner interface{ Scan(...any) error }

func makeJourney(id, journalID uuid.UUID, slug string, sourceRef sql.NullString, title, place string, country, region sql.NullString, start, end, route, fields, created, updated string) (*domain.Journey, error) {
	var gps orb.MultiLineString
	if err := unmarshalJSON(route, &gps); err != nil {
		return nil, err
	}
	authored, err := parseStrings(fields)
	if err != nil {
		return nil, err
	}
	dateStart, _ := time.Parse("2006-01-02", start)
	dateEnd, _ := time.Parse("2006-01-02", end)
	createdAt, _ := time.Parse(time.RFC3339Nano, created)
	updatedAt, _ := time.Parse(time.RFC3339Nano, updated)
	return &domain.Journey{ID: id, JournalID: journalID, Slug: slug, SourceRef: readString(sourceRef), Title: title, Place: place, Country: readString(country), Region: readString(region), DateStart: dateStart, DateEnd: dateEnd, GPSRoute: gps, AuthoredFields: authored, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}
