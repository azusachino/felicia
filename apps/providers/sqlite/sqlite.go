// Package sqlite implements Felicia's local SQLite provider.
package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"
	_ "modernc.org/sqlite"

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
		if err := json.Unmarshal([]byte(mustJSON(raw.Coordinates)), &coords); err != nil || len(coords) < 2 {
			return nil, fmt.Errorf("invalid point geometry")
		}
		return orb.Point{coords[0], coords[1]}, nil
	case "LineString":
		var coords [][]float64
		if err := json.Unmarshal([]byte(mustJSON(raw.Coordinates)), &coords); err != nil {
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
		if err := json.Unmarshal([]byte(mustJSON(raw.Coordinates)), &coords); err != nil {
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

func (r *Repository) GetJournal(ctx context.Context, id uuid.UUID) (*domain.Journal, error) {
	var created string
	err := r.db.QueryRowContext(ctx, "SELECT created_at FROM tb_journals WHERE id = ?", idString(id)).Scan(&created)
	if err != nil {
		return nil, fmt.Errorf("get journal %s: %w", id, err)
	}
	t, _ := time.Parse(time.RFC3339Nano, created)
	return &domain.Journal{ID: id, CreatedAt: t}, nil
}

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

func (r *Repository) ResetMockJournal(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tb_journals WHERE id = ?", idString(id))
	return err
}

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

func (r *Repository) ListJourneys(ctx context.Context) ([]*domain.Journey, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at FROM tb_journeys ORDER BY date_start DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
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
		id := parsed
		journalID, err := parseID(rawJournalID)
		if err != nil {
			return nil, err
		}
		journey, err := makeJourney(id, journalID, slug, sourceRef, title, place, country, region, start, end, route, fields, created, updated)
		if err != nil {
			return nil, err
		}
		result = append(result, journey)
	}
	return result, rows.Err()
}

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

func scanJourney(row scanner, id uuid.UUID) (*domain.Journey, error) {
	var journalID uuid.UUID
	var slug, title, place, start, end, route, fields, created, updated string
	var sourceRef, country, region sql.NullString
	if err := row.Scan(&journalID, &slug, &sourceRef, &title, &place, &country, &region, &start, &end, &route, &fields, &created, &updated); err != nil {
		return nil, err
	}
	return makeJourney(id, journalID, slug, sourceRef, title, place, country, region, start, end, route, fields, created, updated)
}

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

const mementoColumns = `id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at`

func (r *Repository) GetMemento(ctx context.Context, id uuid.UUID) (*domain.Memento, error) {
	return r.scanMemento(r.db.QueryRowContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE id = ?", idString(id)))
}

func (r *Repository) GetMementoBySourceIdentity(ctx context.Context, source domain.SourceIdentity) (*domain.Memento, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	return r.scanMemento(r.db.QueryRowContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE source_system = ? AND source_external_id = ?", source.System, source.ExternalID))
}

func (r *Repository) ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE journey_id = ? ORDER BY seq, occurred_at", idString(journeyID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Memento
	for rows.Next() {
		memento, err := r.scanMemento(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, memento)
	}
	return result, rows.Err()
}

func (r *Repository) scanMemento(row scanner) (*domain.Memento, error) {
	var rawID, rawJourneyID string
	var kind string
	var seq, revision int
	var occurredAt, occurredTZ, geom, title, place, vendor, essay, sourceSystem, sourceExternalID, sourceRef, orphanedAt sql.NullString
	var priceAmount sql.NullInt64
	var priceCurrency sql.NullString
	var kindData, authoredFields, state, createdAt, updatedAt string
	if err := row.Scan(&rawID, &rawJourneyID, &kind, &seq, &occurredAt, &occurredTZ, &geom, &title, &place, &vendor, &essay, &priceAmount, &priceCurrency, &kindData, &sourceSystem, &sourceExternalID, &sourceRef, &authoredFields, &orphanedAt, &state, &revision, &createdAt, &updatedAt); err != nil {
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
	decodedGeom, err := decodeGeometry(geom)
	if err != nil {
		return nil, err
	}
	fields, err := parseStrings(authoredFields)
	if err != nil {
		return nil, err
	}
	var data json.RawMessage
	if kindData != "" {
		data = json.RawMessage(kindData)
	}
	var source *domain.SourceIdentity
	if sourceSystem.Valid && sourceExternalID.Valid {
		sourceValue := domain.SourceIdentity{System: sourceSystem.String, ExternalID: sourceExternalID.String}
		source = &sourceValue
	}
	created, _ := time.Parse(time.RFC3339Nano, createdAt)
	updated, _ := time.Parse(time.RFC3339Nano, updatedAt)
	var amount *int64
	if priceAmount.Valid {
		amount = &priceAmount.Int64
	}
	return &domain.Memento{ID: id, JourneyID: journeyID, Kind: kind, Seq: seq, OccurredAt: readTime(occurredAt), OccurredTZ: occurredTZ.String, Geom: decodedGeom, Title: title.String, Place: place.String, Vendor: readString(vendor), Essay: readString(essay), PriceAmount: amount, PriceCurrency: readString(priceCurrency), KindData: data, SourceIdentity: source, SourceRef: readString(sourceRef), AuthoredFields: fields, OrphanedAt: timePtr(orphanedAt), State: domain.MementoState(state), Revision: int64(revision), CreatedAt: created, UpdatedAt: updated}, nil
}

func timePtr(value sql.NullString) *time.Time {
	if !value.Valid || value.String == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, value.String)
	if err != nil {
		return nil
	}
	return &t
}

func sourceValues(memento *domain.Memento) (any, any, any) {
	if memento.SourceIdentity != nil && memento.SourceIdentity.Valid() {
		ref := memento.SourceRef
		if ref == nil {
			value := memento.SourceIdentity.Ref()
			ref = &value
		}
		return memento.SourceIdentity.System, memento.SourceIdentity.ExternalID, nullableString(ref)
	}
	return nil, nil, nullableString(memento.SourceRef)
}

func (r *Repository) UpsertMemento(ctx context.Context, memento *domain.Memento) error {
	return r.upsertMemento(ctx, memento, nil)
}

func (r *Repository) upsertMemento(ctx context.Context, memento *domain.Memento, expected *int64) error {
	geom, err := encodeGeometry(memento.Geom)
	if err != nil {
		return err
	}
	fields, err := stringsJSON(memento.AuthoredFields)
	if err != nil {
		return err
	}
	data := string(memento.KindData)
	if data == "" {
		data = "{}"
	}
	system, externalID, sourceRef := sourceValues(memento)
	state := memento.State
	if state == "" {
		state = domain.MementoDraft
	}
	revision := memento.Revision
	if revision == 0 {
		revision = 1
	}
	args := []any{idString(memento.ID), idString(memento.JourneyID), memento.Kind, memento.Seq, nullableTime(memento.OccurredAt), nullableStringValue(memento.OccurredTZ), geom, nullableStringValue(memento.Title), nullableStringValue(memento.Place), nullableString(memento.Vendor), nullableString(memento.Essay), nullableInt(memento.PriceAmount), nullableString(memento.PriceCurrency), data, system, externalID, sourceRef, fields, nullableTimePtr(memento.OrphanedAt), string(state), revision, timeOrNow(memento.CreatedAt), timeOrNow(memento.UpdatedAt)}
	query := `INSERT INTO tb_mementos(` + mementoColumns + `) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET journey_id=excluded.journey_id, kind=excluded.kind, seq=excluded.seq, occurred_at=excluded.occurred_at, occurred_tz=excluded.occurred_tz, geom=excluded.geom, title=excluded.title, place=excluded.place, vendor=excluded.vendor, essay=excluded.essay, price_amount=excluded.price_amount, price_currency=excluded.price_currency, kind_data=excluded.kind_data, source_system=excluded.source_system, source_external_id=excluded.source_external_id, source_ref=excluded.source_ref, authored_fields=excluded.authored_fields, orphaned_at=excluded.orphaned_at, state=excluded.state, revision=tb_mementos.revision+1, updated_at=excluded.updated_at`
	if expected != nil {
		var current int64
		if err := r.db.QueryRowContext(ctx, "SELECT revision FROM tb_mementos WHERE id = ?", idString(memento.ID)).Scan(&current); err != nil {
			return err
		}
		if current != *expected {
			return domain.ErrWriteConflict
		}
	}
	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("upsert memento %s: %w", memento.ID, err)
	}
	return nil
}

func nullableStringValue(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableInt(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTimePtr(value *time.Time) any {
	if value == nil {
		return nil
	}
	return nullableTime(*value)
}

func timeOrNow(value time.Time) string {
	if value.IsZero() {
		return now()
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func (r *Repository) ApplyManualMementoPatch(ctx context.Context, patch *domain.ManualMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("manual memento patch is required")
	}
	current, err := r.GetMemento(ctx, patch.Memento.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID}
	}
	if patch.ExpectedRevision != nil && current.Revision != 0 && current.Revision != *patch.ExpectedRevision {
		return domain.ErrWriteConflict
	}
	mergeMemento(current, patch.Memento, patch.Fields)
	current.AuthoredFields = unionFields(current.AuthoredFields, patch.Fields)
	if patch.State != "" {
		current.State = patch.State
	}
	if current.Revision == 0 {
		current.Revision = 1
	}
	return r.upsertMemento(ctx, current, patch.ExpectedRevision)
}

func (r *Repository) ApplyIngestMementoPatch(ctx context.Context, patch *domain.IngestMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("ingest memento patch is required")
	}
	var current *domain.Memento
	var err error
	if patch.Memento.SourceIdentity != nil && patch.Memento.SourceIdentity.Valid() {
		current, err = r.GetMementoBySourceIdentity(ctx, *patch.Memento.SourceIdentity)
	}
	if errors.Is(err, sql.ErrNoRows) || current == nil {
		current, err = r.GetMemento(ctx, patch.Memento.ID)
	}
	if errors.Is(err, sql.ErrNoRows) {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID, State: domain.MementoCandidateState}
		err = nil
	}
	if err != nil {
		return err
	}
	mergeMemento(current, patch.Memento, patch.Fields)
	if patch.Memento.SourceIdentity != nil && patch.Memento.SourceIdentity.Valid() {
		current.SourceIdentity = patch.Memento.SourceIdentity
		if current.SourceRef == nil {
			ref := patch.Memento.SourceIdentity.Ref()
			current.SourceRef = &ref
		}
	}
	if current.State == "" {
		current.State = domain.MementoCandidateState
	}
	return r.UpsertMemento(ctx, current)
}

func mergeMemento(dst, src *domain.Memento, fields []string) {
	for _, field := range fields {
		switch field {
		case "journey_id":
			dst.JourneyID = src.JourneyID
		case "kind":
			dst.Kind = src.Kind
		case "seq":
			dst.Seq = src.Seq
		case "occurred_at":
			dst.OccurredAt = src.OccurredAt
		case "occurred_tz":
			dst.OccurredTZ = src.OccurredTZ
		case "geom":
			dst.Geom = src.Geom
		case "title":
			dst.Title = src.Title
		case "place":
			dst.Place = src.Place
		case "vendor":
			dst.Vendor = src.Vendor
		case "essay":
			dst.Essay = src.Essay
		case "price_amount":
			dst.PriceAmount = src.PriceAmount
		case "price_currency":
			dst.PriceCurrency = src.PriceCurrency
		case "kind_data":
			dst.KindData = src.KindData
		case "source_ref":
			dst.SourceRef = src.SourceRef
		case "orphaned_at":
			dst.OrphanedAt = src.OrphanedAt
		}
	}
}

func unionFields(existing, added []string) []string {
	result := slices.Clone(existing)
	for _, field := range added {
		if !slices.Contains(result, field) {
			result = append(result, field)
		}
	}
	return result
}

func (r *Repository) CreateTransitLeg(ctx context.Context, leg *domain.TransitLegInput) error {
	geom := orb.LineString{leg.Origin, leg.Dest}
	encoded, err := encodeGeometry(geom)
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO tb_transit_legs(id, journey_id, seq, origin_label, dest_label, geom, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, idString(leg.ID), idString(leg.JourneyID), leg.Seq, nullableString(leg.OriginLabel), nullableString(leg.DestLabel), encoded, now())
	return err
}

func (r *Repository) ListTransitLegsByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.TransitLeg, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, seq, origin_label, dest_label, geom FROM tb_transit_legs WHERE journey_id = ? ORDER BY seq", idString(journeyID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.TransitLeg
	for rows.Next() {
		var rawID string
		var seq int
		var origin, dest, geom sql.NullString
		if err := rows.Scan(&rawID, &seq, &origin, &dest, &geom); err != nil {
			return nil, err
		}
		id, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		decoded, err := decodeGeometry(geom)
		if err != nil {
			return nil, err
		}
		line, ok := decoded.(orb.LineString)
		if !ok || len(line) < 2 {
			continue
		}
		result = append(result, &domain.TransitLeg{ID: id, JourneyID: journeyID, Seq: seq, OriginLabel: readString(origin), DestLabel: readString(dest), Geom: line})
	}
	return result, rows.Err()
}

func (r *Repository) DeleteTransitLeg(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM tb_transit_legs WHERE id = ?", idString(id))
	return err
}

func (r *Repository) GetDisplayRoute(ctx context.Context, journeyID uuid.UUID) (orb.MultiLineString, error) {
	journey, err := r.GetJourney(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	route := slices.Clone(journey.GPSRoute)
	legs, err := r.ListTransitLegsByJourney(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	for _, leg := range legs {
		route = append(route, leg.Geom)
	}
	return route, nil
}

func (r *Repository) SnapToRoute(ctx context.Context, journeyID uuid.UUID, point orb.Point) (*orb.Point, error) {
	route, err := r.GetDisplayRoute(ctx, journeyID)
	if err != nil {
		return nil, err
	}
	if len(route) == 0 {
		return nil, nil
	}
	return &point, nil
}

func (r *Repository) GetPhoto(ctx context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	row := r.db.QueryRowContext(ctx, "SELECT memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at FROM tb_memento_photos WHERE id = ?", idString(id))
	return scanPhoto(row, id)
}

func (r *Repository) ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at FROM tb_memento_photos WHERE memento_id = ? ORDER BY seq", idString(mementoID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.MementoPhoto
	for rows.Next() {
		var rawID string
		var objectKey, hash string
		var caption, takenAt, sourceRef, created sql.NullString
		var seq int
		if err := rows.Scan(&rawID, &objectKey, &hash, &caption, &seq, &takenAt, &sourceRef, &created); err != nil {
			return nil, err
		}
		id, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		result = append(result, photoFromValues(id, mementoID, objectKey, hash, caption, seq, takenAt, sourceRef, created))
	}
	return result, rows.Err()
}

func (r *Repository) UpsertPhoto(ctx context.Context, photo *domain.MementoPhoto) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tb_memento_photos(id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO UPDATE SET object_key=excluded.object_key, content_hash=excluded.content_hash, caption=excluded.caption, seq=excluded.seq, taken_at=excluded.taken_at, source_ref=excluded.source_ref`, idString(photo.ID), idString(photo.MementoID), photo.ObjectKey, photo.ContentHash, nullableString(photo.Caption), photo.Seq, nullableTimePtr(photo.TakenAt), nullableString(photo.SourceRef), timeOrNow(photo.CreatedAt))
	return err
}

func scanPhoto(row scanner, id uuid.UUID) (*domain.MementoPhoto, error) {
	var rawMemento, objectKey, hash string
	var caption, takenAt, sourceRef, created sql.NullString
	var seq int
	if err := row.Scan(&rawMemento, &objectKey, &hash, &caption, &seq, &takenAt, &sourceRef, &created); err != nil {
		return nil, err
	}
	mementoID, err := parseID(rawMemento)
	if err != nil {
		return nil, err
	}
	return photoFromValues(id, mementoID, objectKey, hash, caption, seq, takenAt, sourceRef, created), nil
}

func photoFromValues(id, mementoID uuid.UUID, objectKey, hash string, caption sql.NullString, seq int, takenAt, sourceRef, created sql.NullString) *domain.MementoPhoto {
	return &domain.MementoPhoto{ID: id, MementoID: mementoID, ObjectKey: objectKey, ContentHash: hash, Caption: readString(caption), Seq: seq, TakenAt: timePtr(takenAt), SourceRef: readString(sourceRef), CreatedAt: readTime(created)}
}

func (r *Repository) ListTranslations(ctx context.Context, ownerType string, ownerID uuid.UUID) ([]*domain.Translation, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id, lang, field, value, provenance, updated_at FROM tb_translations WHERE owner_type = ? AND owner_id = ? ORDER BY lang, field", ownerType, idString(ownerID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*domain.Translation
	for rows.Next() {
		var rawID, lang, field, value, provenance, updated string
		if err := rows.Scan(&rawID, &lang, &field, &value, &provenance, &updated); err != nil {
			return nil, err
		}
		id, err := parseID(rawID)
		if err != nil {
			return nil, err
		}
		updatedAt, _ := time.Parse(time.RFC3339Nano, updated)
		result = append(result, &domain.Translation{ID: id, OwnerType: ownerType, OwnerID: ownerID, Lang: lang, Field: field, Value: value, Provenance: provenance, UpdatedAt: updatedAt})
	}
	return result, rows.Err()
}

func (r *Repository) UpsertTranslation(ctx context.Context, translation *domain.Translation) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO tb_translations(id, owner_type, owner_id, lang, field, value, provenance, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(owner_type, owner_id, lang, field) DO UPDATE SET value=excluded.value, provenance=excluded.provenance, updated_at=excluded.updated_at`, idString(translation.ID), translation.OwnerType, idString(translation.OwnerID), translation.Lang, translation.Field, translation.Value, translation.Provenance, timeOrNow(translation.UpdatedAt))
	return err
}

func (r *Repository) CreateImportRun(ctx context.Context, run *domain.ImportRun) error {
	if run == nil {
		return errors.New("import run is required")
	}
	if run.ID == uuid.Nil {
		run.ID = uuid.Must(uuid.NewV7())
	}
	if run.Status == "" {
		run.Status = domain.ImportRunRunning
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO tb_import_runs(id, source_system, started_at, status, error_message) VALUES (?, ?, ?, ?, ?)`, idString(run.ID), run.SourceSystem, run.StartedAt.Format(time.RFC3339Nano), string(run.Status), nullableString(run.ErrorMessage))
	return err
}

func (r *Repository) FinishImportRun(ctx context.Context, id uuid.UUID, status domain.ImportRunStatus, finishedAt time.Time, errorMessage *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tb_import_runs SET status = ?, finished_at = ?, error_message = ? WHERE id = ?`, string(status), nullableTime(finishedAt), nullableString(errorMessage), idString(id))
	return err
}

func (r *Repository) RecordSourceObservation(ctx context.Context, observation *domain.SourceObservation) error {
	if observation == nil || !observation.Source.Valid() || observation.RunID == uuid.Nil || observation.ObservedAt.IsZero() || !json.Valid(observation.Payload) {
		return errors.New("invalid source observation")
	}
	if observation.ID == uuid.Nil {
		observation.ID = uuid.Must(uuid.NewV7())
	}
	var previous string
	changed := 0
	err := r.db.QueryRowContext(ctx, "SELECT payload FROM tb_source_observations WHERE source_system = ? AND source_external_id = ? ORDER BY observed_at DESC LIMIT 1", observation.Source.System, observation.Source.ExternalID).Scan(&previous)
	if err == nil && previous != string(observation.Payload) {
		changed = 1
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO tb_source_observations(id, run_id, source_system, source_external_id, kind, observed_at, confidence, payload, changed, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(run_id, source_system, source_external_id) DO UPDATE SET payload=excluded.payload, observed_at=excluded.observed_at, confidence=excluded.confidence, changed=excluded.changed, orphaned_at=NULL`, idString(observation.ID), idString(observation.RunID), observation.Source.System, observation.Source.ExternalID, string(observation.Kind), observation.ObservedAt.Format(time.RFC3339Nano), observation.Confidence, string(observation.Payload), changed, now())
	return err
}

func (r *Repository) MarkMissingSourceObservations(ctx context.Context, runID uuid.UUID, sourceSystem string, seenExternalIDs []string) error {
	query := "UPDATE tb_source_observations SET orphaned_at = ? WHERE source_system = ? AND run_id <> ?"
	args := []any{now(), sourceSystem, idString(runID)}
	if len(seenExternalIDs) > 0 {
		placeholders := make([]string, len(seenExternalIDs))
		for i, id := range seenExternalIDs {
			placeholders[i] = "?"
			args = append(args, id)
		}
		query += " AND source_external_id NOT IN (" + strings.Join(placeholders, ",") + ")"
	}
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}
