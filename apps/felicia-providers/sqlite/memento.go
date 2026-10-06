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
	modsqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
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

// CreateMementoWithNextSequence inserts a new draft memento and allocates
// its display position atomically: the row and its journey-local seq
// (COALESCE(MAX(seq), -1) + 1) are written by a single INSERT, so concurrent
// creators across processes cannot share a position (docs/contracts/
// automatic-sequence-allocation.md). The caller-chosen ID makes a retry
// idempotent: when the ID already exists and the stored row still matches
// the supplied authored creation values, the stored row is returned without
// an insert, an update, a reallocation or a revision bump. A different
// journey, or stored values that no longer match (including intervening
// edits), is domain.ErrWriteConflict and never overwrites the stored row.
// The supplied Seq is ignored — positions are server-allocated on this
// path; explicit authored ordering stays on the existing patch/upsert path.
func (r *Repository) CreateMementoWithNextSequence(ctx context.Context, memento *domain.Memento) (*domain.Memento, error) {
	if memento == nil {
		return nil, errors.New("create memento: memento is required")
	}
	if memento.ID == uuid.Nil {
		return nil, errors.New("create memento: id is required")
	}
	if memento.JourneyID == uuid.Nil {
		return nil, errors.New("create memento: journey id is required")
	}
	if memento.Kind == "" {
		return nil, fmt.Errorf("create memento %s: kind is required", memento.ID)
	}
	// The new-draft form only ever creates drafts; a non-draft lifecycle
	// state must go through the edit path, never through creation.
	if memento.State != "" && memento.State != domain.MementoDraft {
		return nil, fmt.Errorf("create memento %s: state %q is not a draft", memento.ID, memento.State)
	}
	if memento.SourceIdentity != nil && !memento.SourceIdentity.Valid() {
		return nil, fmt.Errorf("create memento %s: invalid source identity", memento.ID)
	}
	values, err := mementoCreateValues(memento)
	if err != nil {
		return nil, fmt.Errorf("create memento %s: %w", memento.ID, err)
	}
	// One statement: the MAX(seq) subquery, the row insert and the returned
	// persisted row all run inside the same write transaction. The incoming
	// Seq is deliberately not part of the insert.
	query := `INSERT INTO tb_mementos(` + mementoColumns + `) SELECT ?, ?, ?, COALESCE(MAX(seq), -1) + 1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ? FROM tb_mementos WHERE journey_id = ? RETURNING ` + mementoColumns
	args := make([]any, 0, len(values)+1)
	args = append(args, values[:3]...)               // id, journey_id, kind
	args = append(args, values[4:]...)               // remaining columns in stored order
	args = append(args, idString(memento.JourneyID)) // aggregate filter
	row := r.db.QueryRowContext(ctx, query, args...)
	created, err := r.scanMemento(row)
	if err == nil {
		domain.LogMementoStateChange(ctx, nil, created, "", created.State)
		return created, nil
	}
	// Recover only a genuine identity duplicate (primary-key or unique
	// constraint): the ID already exists, so decide between a matching retry
	// and a conflict. Any other failure (IO, lock, FK) propagates unchanged.
	if !isSQLiteIdentityDuplicate(err) {
		return nil, fmt.Errorf("create memento %s: %w", memento.ID, err)
	}
	stored, fetchErr := r.GetMemento(ctx, memento.ID)
	if errors.Is(fetchErr, sql.ErrNoRows) {
		// The duplicate is on another identity (e.g. source identity), not ours.
		return nil, fmt.Errorf("create memento %s: %w", memento.ID, domain.ErrWriteConflict)
	}
	if fetchErr != nil {
		return nil, fmt.Errorf("create memento %s: fetch stored row: %w", memento.ID, fetchErr)
	}
	match, err := mementoStoredRowMatches(stored, values)
	if err != nil {
		return nil, fmt.Errorf("create memento %s: compare stored row: %w", memento.ID, err)
	}
	if !match {
		return nil, fmt.Errorf("create memento %s: %w", memento.ID, domain.ErrWriteConflict)
	}
	return stored, nil
}

// mementoCreateValues encodes the authored creation values of a memento in
// stored form: id, journey_id, kind, seq, the remaining authored columns,
// then revision 1 and the created/updated timestamps. Indices 20..22 are
// server-assigned defaults and are excluded from retry comparison; index 3
// (seq) is likewise excluded by the create statement and comparison.
func mementoCreateValues(memento *domain.Memento) ([]any, error) {
	geom, err := encodeGeometry(memento.Geom)
	if err != nil {
		return nil, err
	}
	fields, err := stringsJSON(memento.AuthoredFields)
	if err != nil {
		return nil, err
	}
	data := string(memento.KindData)
	if data == "" {
		data = "{}"
	}
	system, externalID, sourceRef := sourceValues(memento)
	state := string(memento.State)
	if state == "" {
		state = string(domain.MementoDraft)
	}
	return []any{idString(memento.ID), idString(memento.JourneyID), memento.Kind, memento.Seq, nullableTime(memento.OccurredAt), nullableStringValue(memento.OccurredTZ), geom, nullableStringValue(memento.Title), nullableStringValue(memento.Place), nullableString(memento.Vendor), nullableString(memento.Essay), nullableInt(memento.PriceAmount), nullableString(memento.PriceCurrency), data, system, externalID, sourceRef, fields, nullableTimePtr(memento.OrphanedAt), state, 1, timeOrNow(memento.CreatedAt), timeOrNow(memento.UpdatedAt)}, nil
}

// mementoStoredRowMatches reports whether the stored row still matches the
// normalized authored creation values, ignoring server-assigned fields
// (seq, revision, created_at, updated_at).
func mementoStoredRowMatches(stored *domain.Memento, values []any) (bool, error) {
	storedValues, err := mementoCreateValues(stored)
	if err != nil {
		return false, err
	}
	for i := range values {
		if i == 3 || i >= 20 { // seq, revision and timestamps are server-assigned
			continue
		}
		if !mementoValueMatches(values[i], storedValues[i]) {
			return false, nil
		}
	}
	return true, nil
}

// mementoValueMatches compares two stored-form values. Both sides run
// through the same encoding, so driver types are flattened before compare.
func mementoValueMatches(incoming, stored any) bool {
	incomingText, incomingOK := mementoValueText(incoming)
	storedText, storedOK := mementoValueText(stored)
	if incomingOK != storedOK {
		return false
	}
	return !incomingOK || incomingText == storedText
}

// mementoValueText renders one stored-form value as (text, present).
func mementoValueText(value any) (string, bool) {
	switch v := value.(type) {
	case nil:
		return "", false
	case string:
		return v, true
	case int:
		return fmt.Sprint(v), true
	case int64:
		return fmt.Sprint(v), true
	default:
		text, err := marshalJSON(value)
		if err != nil {
			return fmt.Sprint(value), true
		}
		return text, true
	}
}

// isSQLiteIdentityDuplicate reports whether the error is a SQLite UNIQUE
// constraint failure (primary key or unique index), as opposed to IO,
// locking, cancellation, NOT NULL or foreign-key failures. A rowid
// constraint (SQLITE_CONSTRAINT_ROWID) cannot fire here: the identity
// column is a TEXT primary key, not an integer alias.
func isSQLiteIdentityDuplicate(err error) bool {
	var sqliteErr *modsqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
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
	exists := err == nil
	var fromState domain.MementoState
	if exists {
		fromState = current.State
	} else {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID}
	}
	if patch.ExpectedRevision != nil && current.Revision != 0 && current.Revision != *patch.ExpectedRevision {
		return domain.ErrWriteConflict
	}
	// Lifecycle guard: an existing row may only change state along a legal
	// transition (docs/contracts/memento-lifecycle.md §3). Creation is
	// unconstrained (no prior state to transition from).
	if exists && patch.State != "" && !domain.CanTransitionMementoState(fromState, patch.State) {
		return &domain.InvalidTransitionError{From: fromState, To: patch.State}
	}
	prevRevision := current.Revision
	mergeMemento(current, patch.Memento, patch.Fields)
	current.AuthoredFields = unionFields(current.AuthoredFields, patch.Fields)
	if patch.State != "" {
		current.State = patch.State
	}
	if current.Revision == 0 {
		current.Revision = 1
	}
	toState := current.State
	if err := r.upsertMemento(ctx, current, patch.ExpectedRevision); err != nil {
		return err
	}
	if !exists || fromState != toState {
		logFrom := fromState
		if !exists {
			logFrom = ""
			current.Revision = 1
		} else {
			current.Revision = prevRevision + 1
		}
		domain.LogMementoStateChange(ctx, nil, current, logFrom, toState)
	}
	return nil
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
	created := false
	if errors.Is(err, sql.ErrNoRows) {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID, State: patch.Memento.State}
		created = true
		err = nil
	}
	if err != nil {
		return err
	}
	// An ingest write may only touch fields the author has not claimed, and it
	// leaves current.AuthoredFields untouched so the mask can never shrink
	// (ADR-0033).
	mergeMemento(current, patch.Memento, domain.IngestableFields(patch.Fields, current.AuthoredFields))
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
	if err := r.UpsertMemento(ctx, current); err != nil {
		return err
	}
	// Ingest never changes an existing row's state; only creation is a
	// lifecycle event worth logging.
	if created {
		current.Revision = 1
		domain.LogMementoStateChange(ctx, nil, current, "", current.State)
	}
	return nil
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
