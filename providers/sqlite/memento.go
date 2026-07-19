package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/core/domain"
)

const mementoColumns = `id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at`

// GetMemento retrieves a memento by ID.
func (r *Repository) GetMemento(ctx context.Context, id uuid.UUID) (*domain.Memento, error) {
	return r.scanMemento(r.db.QueryRowContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE id = ?", idString(id)))
}

// GetMementoBySourceIdentity retrieves the memento owned by a source identity.
func (r *Repository) GetMementoBySourceIdentity(ctx context.Context, source domain.SourceIdentity) (*domain.Memento, error) {
	if err := source.Validate(); err != nil {
		return nil, err
	}
	return r.scanMemento(r.db.QueryRowContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE source_system = ? AND source_external_id = ?", source.System, source.ExternalID))
}

// ListMementosByJourney retrieves a journey's mementos in display order.
func (r *Repository) ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+mementoColumns+" FROM tb_mementos WHERE journey_id = ? ORDER BY seq, occurred_at", idString(journeyID))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
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

// UpsertMemento inserts or updates a memento.
func (r *Repository) UpsertMemento(ctx context.Context, memento *domain.Memento) error {
	return r.upsertMemento(ctx, memento, nil)
}

// DeleteMemento removes a memento; its photos cascade via the FK
// (PRAGMA foreign_keys is enabled at open).
func (r *Repository) DeleteMemento(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM tb_mementos WHERE id = ?", idString(id))
	if err != nil {
		return fmt.Errorf("delete memento %s: %w", id, err)
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.ErrNotFound
	}
	return nil
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

// ApplyManualMementoPatch applies authored fields with revision protection.
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

// ApplyIngestMementoPatch applies source fields without taking authorship.
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
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID, State: patch.Memento.State}
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
		if patch.Memento.State != "" {
			current.State = patch.Memento.State
		} else {
			current.State = domain.MementoCandidateState
		}
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
