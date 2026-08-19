// Package postgres implements the PostgreSQL 18 database store repository.
package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/core/ports"
	"github.com/azusachino/felicia/providers/postgres/db"
)

// Compile-time check that pgRepository satisfies the domain contract.
var _ domain.Repository = (*pgRepository)(nil)
var _ domain.ObservationStore = (*pgRepository)(nil)
var _ ports.SiteSettingsStore = (*pgRepository)(nil)

type pgRepository struct {
	q  *db.Queries
	db db.DBTX
}

// NewRepository creates a new domain.Repository implementation backed by Postgres.
func NewRepository(d db.DBTX) domain.Repository {
	return &pgRepository{
		q: db.New(d), db: d,
	}
}

// Journal operations

func (r *pgRepository) GetJournal(ctx context.Context, id uuid.UUID) (*domain.Journal, error) {
	row, err := r.q.GetJournal(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get journal %s: %w", id, err)
	}
	return &domain.Journal{
		ID:        row.ID,
		CreatedAt: fromTimestamptz(row.CreatedAt),
	}, nil
}

func (r *pgRepository) CreateJournal(ctx context.Context, journal *domain.Journal) error {
	if err := r.q.CreateJournal(ctx, db.CreateJournalParams{
		ID:        journal.ID,
		CreatedAt: toTimestamptz(journal.CreatedAt),
	}); err != nil {
		return fmt.Errorf("create journal %s: %w", journal.ID, err)
	}
	return nil
}

func (r *pgRepository) ResetMockJournal(ctx context.Context, id uuid.UUID) error {
	if err := r.q.ResetMockJournal(ctx, id); err != nil {
		return fmt.Errorf("reset mock journal %s: %w", id, err)
	}
	return nil
}

// GetSoleJournal returns the single journal row expected in this
// single-tenant deployment.
func (r *pgRepository) GetSoleJournal(ctx context.Context) (*domain.Journal, error) {
	row, err := r.q.GetSoleJournal(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get sole journal: %w", err)
	}
	return &domain.Journal{
		ID:        row.ID,
		CreatedAt: fromTimestamptz(row.CreatedAt),
	}, nil
}

// Site settings operations (ADMIN-02 M2)

func (r *pgRepository) GetSiteSettings(ctx context.Context, journalID uuid.UUID) (*domain.SiteSettings, error) {
	row, err := r.q.GetSiteSettings(ctx, journalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("get site settings %s: %w", journalID, err)
	}
	return &domain.SiteSettings{
		JournalID:       row.JournalID,
		Title:           row.Title,
		Description:     row.Description,
		Design:          row.Design,
		DefaultLanguage: row.DefaultLanguage,
		DefaultTheme:    row.DefaultTheme,
		Accent:          row.Accent,
		CreatedAt:       fromTimestamptz(row.CreatedAt),
		UpdatedAt:       fromTimestamptz(row.UpdatedAt),
	}, nil
}

func (r *pgRepository) UpsertSiteSettings(ctx context.Context, settings *domain.SiteSettings) error {
	if err := r.q.UpsertSiteSettings(ctx, db.UpsertSiteSettingsParams{
		JournalID:       settings.JournalID,
		Title:           settings.Title,
		Description:     settings.Description,
		Design:          settings.Design,
		DefaultLanguage: settings.DefaultLanguage,
		DefaultTheme:    settings.DefaultTheme,
		Accent:          settings.Accent,
	}); err != nil {
		return fmt.Errorf("upsert site settings %s: %w", settings.JournalID, err)
	}
	return nil
}

// CreateImportRun starts a durable source synchronization boundary. UUIDv7 is
// generated here when the caller does not provide an ID so run ordering stays
// visible in the identifier as well as in started_at.
func (r *pgRepository) CreateImportRun(ctx context.Context, run *domain.ImportRun) error {
	if run == nil {
		return errors.New("import run is required")
	}
	if run.ID == uuid.Nil {
		run.ID = uuid.Must(uuid.NewV7())
	}
	if run.Status == "" {
		run.Status = domain.ImportRunRunning
	}
	if run.SourceSystem == "" {
		return errors.New("import run source system is required")
	}
	if run.StartedAt.IsZero() {
		run.StartedAt = time.Now().UTC()
	}
	if err := r.q.CreateImportRun(ctx, db.CreateImportRunParams{
		ID: run.ID, SourceSystem: run.SourceSystem, StartedAt: toTimestamptz(run.StartedAt),
		Status: string(run.Status), ErrorMessage: toText(run.ErrorMessage),
	}); err != nil {
		return fmt.Errorf("create import run %s: %w", run.ID, err)
	}
	return nil
}

func (r *pgRepository) FinishImportRun(ctx context.Context, id uuid.UUID, status domain.ImportRunStatus, finishedAt time.Time, errorMessage *string) error {
	if !validImportRunStatus(status) || finishedAt.IsZero() {
		return errors.New("invalid import run completion")
	}
	if err := r.q.FinishImportRun(ctx, db.FinishImportRunParams{
		ID: id, Status: string(status), FinishedAt: toTimestamptz(finishedAt), ErrorMessage: toText(errorMessage),
	}); err != nil {
		return fmt.Errorf("finish import run %s: %w", id, err)
	}
	return nil
}

func (r *pgRepository) RecordSourceObservation(ctx context.Context, observation *domain.SourceObservation) error {
	if observation == nil || !observation.Source.Valid() || observation.RunID == uuid.Nil {
		return errors.New("source observation, run, and source identity are required")
	}
	if observation.Kind == "" || observation.ObservedAt.IsZero() || observation.Confidence < 0 || observation.Confidence > 1 || !json.Valid(observation.Payload) {
		return errors.New("invalid source observation")
	}
	if observation.ID == uuid.Nil {
		observation.ID = uuid.Must(uuid.NewV7())
	}
	if err := r.q.RecordSourceObservation(ctx, db.RecordSourceObservationParams{
		ID: observation.ID, RunID: observation.RunID, SourceSystem: observation.Source.System,
		SourceExternalID: observation.Source.ExternalID, Kind: string(observation.Kind),
		ObservedAt: toTimestamptz(observation.ObservedAt), Confidence: observation.Confidence,
		Payload: observation.Payload,
	}); err != nil {
		return fmt.Errorf("record source observation %s: %w", observation.ID, err)
	}
	return nil
}

func (r *pgRepository) MarkMissingSourceObservations(ctx context.Context, runID uuid.UUID, sourceSystem string, seenExternalIDs []string) error {
	if runID == uuid.Nil || sourceSystem == "" {
		return errors.New("run ID and source system are required")
	}
	if err := r.q.MarkMissingSourceObservations(ctx, db.MarkMissingSourceObservationsParams{
		RunID: runID, SourceSystem: sourceSystem, SeenExternalIds: seenExternalIDs,
	}); err != nil {
		return fmt.Errorf("mark missing source observations for run %s: %w", runID, err)
	}
	return nil
}

func validImportRunStatus(status domain.ImportRunStatus) bool {
	switch status {
	case domain.ImportRunRunning, domain.ImportRunSucceeded, domain.ImportRunFailed:
		return true
	default:
		return false
	}
}

// Journey operations

func (r *pgRepository) GetJourney(ctx context.Context, id uuid.UUID) (*domain.Journey, error) {
	row, err := r.q.GetJourney(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get journey %s: %w", id, err)
	}
	return toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) GetJourneyBySlug(ctx context.Context, slug string) (*domain.Journey, error) {
	row, err := r.q.GetJourneyBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("get journey by slug %q: %w", slug, err)
	}
	return toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) ListJourneys(ctx context.Context) ([]*domain.Journey, error) {
	rows, err := r.q.ListJourneys(ctx)
	if err != nil {
		return nil, fmt.Errorf("list journeys: %w", err)
	}
	var res []*domain.Journey
	for _, row := range rows {
		j, err := toDomainJourney(row.ID, row.JournalID, row.Slug, row.SourceRef, row.Title, row.Place, row.Country, row.Region, row.DateStart, row.DateEnd, row.GpsRouteWkb, row.AuthoredFields, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("list journeys: decode %s: %w", row.ID, err)
		}
		res = append(res, j)
	}
	return res, nil
}

// ApplyIngestJourneyPatch applies source-owned journey fields without taking
// authorship. The authored-field decision is made in Go (shared with the SQLite
// provider) rather than in the upsert SQL, because a single shared upsert cannot
// tell an import from an authoring edit (ADR-0033).
func (r *pgRepository) ApplyIngestJourneyPatch(ctx context.Context, patch *domain.IngestJourneyPatch) error {
	if patch == nil || patch.Journey == nil {
		return errors.New("ingest journey patch is required")
	}
	current, err := r.GetJourney(ctx, patch.Journey.ID)
	switch {
	case errors.Is(err, pgx.ErrNoRows), errors.Is(err, domain.ErrNotFound):
		current = &domain.Journey{ID: patch.Journey.ID, JournalID: patch.Journey.JournalID}
	case err != nil:
		return fmt.Errorf("load journey ingest target %s: %w", patch.Journey.ID, err)
	}
	if current.JournalID == uuid.Nil {
		current.JournalID = patch.Journey.JournalID
	}
	domain.MergeIngestJourney(current, patch)
	return r.UpsertJourney(ctx, current)
}

// UpsertJourney is the *authoring* write: every column and the authored mask
// come from the caller. Source imports must use ApplyIngestJourneyPatch.
func (r *pgRepository) UpsertJourney(ctx context.Context, journey *domain.Journey) error {
	var gpsRouteBytes []byte
	var err error
	if len(journey.GPSRoute) > 0 {
		gpsRouteBytes, err = wkb.Marshal(journey.GPSRoute)
		if err != nil {
			return fmt.Errorf("upsert journey %s: marshal gps route: %w", journey.ID, err)
		}
	}
	authoredFields := nonNilStrings(journey.AuthoredFields)

	if err := r.q.UpsertJourney(ctx, db.UpsertJourneyParams{
		ID:             journey.ID,
		JournalID:      journey.JournalID,
		Slug:           journey.Slug,
		SourceRef:      toText(journey.SourceRef),
		Title:          journey.Title,
		Place:          journey.Place,
		Country:        toText(journey.Country),
		Region:         toText(journey.Region),
		DateStart:      toDate(journey.DateStart),
		DateEnd:        toDate(journey.DateEnd),
		StGeomfromwkb:  gpsRouteBytes,
		AuthoredFields: authoredFields,
	}); err != nil {
		return fmt.Errorf("upsert journey %s: %w", journey.ID, err)
	}
	return nil
}

// Memento operations

func (r *pgRepository) GetMemento(ctx context.Context, id uuid.UUID) (*domain.Memento, error) {
	row, err := r.q.GetMemento(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get memento %s: %w", id, err)
	}
	return toDomainMemento(row.ID, row.JourneyID, row.Kind, row.Seq, row.OccurredAt, row.OccurredTz, row.GeomWkb, row.Title, row.Place, row.Vendor, row.Essay, row.PriceAmount, row.PriceCurrency, row.KindData, row.SourceSystem, row.SourceExternalID, row.SourceRef, row.AuthoredFields, row.OrphanedAt, row.State, row.Revision, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) GetMementoBySourceIdentity(ctx context.Context, source domain.SourceIdentity) (*domain.Memento, error) {
	if err := source.Validate(); err != nil {
		return nil, fmt.Errorf("get memento by source identity: %w", err)
	}
	row, err := r.q.GetMementoBySourceIdentity(ctx, db.GetMementoBySourceIdentityParams{SourceSystem: toText(&source.System), SourceExternalID: toText(&source.ExternalID)})
	if err != nil {
		return nil, fmt.Errorf("get memento by source identity %s: %w", source.Ref(), err)
	}
	return toDomainMemento(row.ID, row.JourneyID, row.Kind, row.Seq, row.OccurredAt, row.OccurredTz, row.GeomWkb, row.Title, row.Place, row.Vendor, row.Essay, row.PriceAmount, row.PriceCurrency, row.KindData, row.SourceSystem, row.SourceExternalID, row.SourceRef, row.AuthoredFields, row.OrphanedAt, row.State, row.Revision, row.CreatedAt, row.UpdatedAt)
}

func (r *pgRepository) ListMementosByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.Memento, error) {
	rows, err := r.q.ListMementosByJourney(ctx, journeyID)
	if err != nil {
		return nil, fmt.Errorf("list mementos for journey %s: %w", journeyID, err)
	}
	var res []*domain.Memento
	for _, row := range rows {
		m, err := toDomainMemento(row.ID, row.JourneyID, row.Kind, row.Seq, row.OccurredAt, row.OccurredTz, row.GeomWkb, row.Title, row.Place, row.Vendor, row.Essay, row.PriceAmount, row.PriceCurrency, row.KindData, row.SourceSystem, row.SourceExternalID, row.SourceRef, row.AuthoredFields, row.OrphanedAt, row.State, row.Revision, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("list mementos for journey %s: decode %s: %w", journeyID, row.ID, err)
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *pgRepository) UpsertMemento(ctx context.Context, memento *domain.Memento) error {
	return r.upsertMemento(ctx, memento, nil)
}

func (r *pgRepository) upsertMemento(ctx context.Context, memento *domain.Memento, expectedRevision *int64) error {
	return r.upsertMementoWith(ctx, memento, expectedRevision, false)
}

func (r *pgRepository) upsertManualMemento(ctx context.Context, memento *domain.Memento, expectedRevision *int64) error {
	return r.upsertMementoWith(ctx, memento, expectedRevision, true)
}

func (r *pgRepository) upsertMementoWith(ctx context.Context, memento *domain.Memento, expectedRevision *int64, manual bool) error {
	var geomBytes []byte
	var err error
	if memento.Geom != nil {
		geomBytes, err = wkb.Marshal(memento.Geom)
		if err != nil {
			return fmt.Errorf("upsert memento %s: marshal geom: %w", memento.ID, err)
		}
	}
	state := mementoStateOrDefault(memento.State)
	sourceSystem, sourceExternalID, sourceRef := sourceColumns(memento)
	authoredFields := nonNilStrings(memento.AuthoredFields)
	var revision *int64
	if memento.Revision > 0 {
		revision = &memento.Revision
	}

	params := db.UpsertMementoParams{
		ID:               memento.ID,
		JourneyID:        memento.JourneyID,
		Kind:             memento.Kind,
		Seq:              int32(memento.Seq),
		OccurredAt:       toTimestamptz(memento.OccurredAt),
		OccurredTz:       pgtype.Text{String: memento.OccurredTZ, Valid: true},
		StGeomfromwkb:    geomBytes,
		Title:            pgtype.Text{String: memento.Title, Valid: true},
		Place:            pgtype.Text{String: memento.Place, Valid: true},
		Vendor:           toText(memento.Vendor),
		Essay:            toText(memento.Essay),
		PriceAmount:      toInt8(memento.PriceAmount),
		PriceCurrency:    toText(memento.PriceCurrency),
		KindData:         memento.KindData,
		SourceSystem:     toText(sourceSystem),
		SourceExternalID: toText(sourceExternalID),
		SourceRef:        toText(sourceRef),
		AuthoredFields:   authoredFields,
		OrphanedAt:       toTimestamptzPtr(memento.OrphanedAt),
		State:            string(state),
		Revision:         toInt8(revision),
		ExpectedRevision: toInt8(expectedRevision),
	}
	if manual {
		// The two upsert queries share an identical parameter shape; Go's
		// struct conversion keeps that in one place.
		err = r.q.UpsertManualMemento(ctx, db.UpsertManualMementoParams(params))
	} else {
		err = r.q.UpsertMemento(ctx, params)
	}
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrWriteConflict
		}
		return fmt.Errorf("upsert memento %s: %w", memento.ID, err)
	}
	return nil
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func mementoStateOrDefault(state domain.MementoState) domain.MementoState {
	if state == "" {
		return domain.MementoDraft
	}
	return state
}

// ApplyManualMementoPatch merges an explicit authoring patch with the current
// row. The field mask, not the incoming authored_fields array, determines
// ownership; this prevents stale clients from clearing authorship.
func (r *pgRepository) ApplyManualMementoPatch(ctx context.Context, patch *domain.ManualMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("manual memento patch is required")
	}
	current, err := r.GetMemento(ctx, patch.Memento.ID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load memento patch target %s: %w", patch.Memento.ID, err)
	}
	exists := err == nil
	var fromState domain.MementoState
	if exists {
		fromState = current.State
	} else {
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID}
	}
	// Optimistic-concurrency pre-check, mirroring the sqlite provider: the
	// UpsertManualMemento query is sqlc `:exec`, so a stale-revision UPDATE
	// silently affects zero rows without surfacing an error. Detect the
	// conflict here off the already-loaded row instead. Creation (revision 0)
	// is exempt.
	if patch.ExpectedRevision != nil && current.Revision != 0 && current.Revision != *patch.ExpectedRevision {
		return domain.ErrWriteConflict
	}
	// Lifecycle guard: an existing row may only change state along a legal
	// transition (docs/contracts/memento-lifecycle.md §3). Creation is
	// unconstrained.
	if exists && patch.State != "" && !domain.CanTransitionMementoState(fromState, patch.State) {
		return &domain.InvalidTransitionError{From: fromState, To: patch.State}
	}
	prevRevision := current.Revision
	mergeMementoFields(current, patch.Memento, patch.Fields)
	current.AuthoredFields = unionFields(current.AuthoredFields, patch.Fields)
	if patch.State != "" {
		current.State = patch.State
	}
	toState := current.State
	if err := r.upsertManualMemento(ctx, current, patch.ExpectedRevision); err != nil {
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

// ApplyMementoAggregate persists the authored memento and child content in a
// single transaction. The transaction is deliberately exposed through a
// separate interface so lightweight repositories and tests need not implement
// PostgreSQL-specific aggregate mechanics.
func (r *pgRepository) ApplyMementoAggregate(ctx context.Context, aggregate *domain.MementoAggregate) error {
	if aggregate == nil || aggregate.Patch == nil || aggregate.Patch.Memento == nil {
		return errors.New("memento aggregate is required")
	}
	beg, ok := r.db.(interface {
		Begin(context.Context) (pgx.Tx, error)
	})
	if !ok {
		return errors.New("repository database does not support transactions")
	}
	tx, err := beg.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin memento aggregate: %w", err)
	}
	txRepo := &pgRepository{q: db.New(tx), db: tx}
	if err := txRepo.ApplyManualMementoPatch(ctx, aggregate.Patch); err != nil {
		_ = tx.Rollback(ctx)
		return fmt.Errorf("apply memento aggregate patch: %w", err)
	}
	for _, photo := range aggregate.Photos {
		if err := txRepo.UpsertPhoto(ctx, photo); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("apply memento aggregate photo: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit memento aggregate: %w", err)
	}
	return nil
}

// ApplyIngestMementoPatch merges source-owned fields and preserves all
// authored ownership. Importers cannot supply or clear authored_fields.
func (r *pgRepository) ApplyIngestMementoPatch(ctx context.Context, patch *domain.IngestMementoPatch) error {
	if patch == nil || patch.Memento == nil {
		return errors.New("ingest memento patch is required")
	}
	identity, hasIdentity := sourceIdentity(patch.Memento)
	var current *domain.Memento
	var err error
	if hasIdentity {
		current, err = r.GetMementoBySourceIdentity(ctx, identity)
	}
	if !hasIdentity || errors.Is(err, pgx.ErrNoRows) {
		current, err = r.GetMemento(ctx, patch.Memento.ID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("load memento ingest target %s: %w", patch.Memento.ID, err)
	}
	created := false
	if err != nil {
		// Honor the package-supplied creation state (matches the sqlite
		// provider); fall back to candidate only when unset. Without this,
		// importing a published fixture row would create it as candidate and
		// then require an illegal candidate→published jump on the next patch.
		current = &domain.Memento{ID: patch.Memento.ID, JourneyID: patch.Memento.JourneyID, State: patch.Memento.State}
		created = true
	}
	// An ingest write may only touch fields the author has not claimed, and it
	// leaves current.AuthoredFields untouched so the mask can never shrink
	// (ADR-0033).
	mergeMementoFields(current, patch.Memento, domain.IngestableFields(patch.Fields, current.AuthoredFields))
	if hasIdentity {
		current.SourceIdentity = &identity
		if current.SourceRef == nil {
			ref := identity.Ref()
			current.SourceRef = &ref
		}
	}
	if current.State == "" {
		current.State = domain.MementoCandidateState
	}
	if err := r.UpsertMemento(ctx, current); err != nil {
		return err
	}
	if created {
		current.Revision = 1
		domain.LogMementoStateChange(ctx, nil, current, "", current.State)
	}
	return nil
}

func sourceIdentity(memento *domain.Memento) (domain.SourceIdentity, bool) {
	if memento.SourceIdentity != nil && memento.SourceIdentity.Valid() {
		return *memento.SourceIdentity, true
	}
	if memento.SourceRef == nil {
		return domain.SourceIdentity{}, false
	}
	identity, ok := sourceIdentityFromRef(*memento.SourceRef)
	return identity, ok
}

func sourceIdentityFromRef(ref string) (domain.SourceIdentity, bool) {
	system, externalID, ok := strings.Cut(ref, ":")
	if !ok || system == "" || externalID == "" {
		return domain.SourceIdentity{}, false
	}
	return domain.SourceIdentity{System: system, ExternalID: externalID}, true
}

func sourceColumns(memento *domain.Memento) (system, externalID, ref *string) {
	identity, ok := sourceIdentity(memento)
	if !ok {
		return nil, nil, memento.SourceRef
	}
	systemValue, externalValue, refValue := identity.System, identity.ExternalID, memento.SourceRef
	if refValue == nil {
		legacyRef := identity.Ref()
		refValue = &legacyRef
	}
	return &systemValue, &externalValue, refValue
}

func mergeMementoFields(dst, src *domain.Memento, fields []string) {
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

// TransitLeg operations

func (r *pgRepository) CreateTransitLeg(ctx context.Context, leg *domain.TransitLegInput) error {
	if err := r.q.CreateTransitLeg(ctx, db.CreateTransitLegParams{
		ID:             leg.ID,
		JourneyID:      leg.JourneyID,
		Seq:            int32(leg.Seq),
		OriginLabel:    toText(leg.OriginLabel),
		DestLabel:      toText(leg.DestLabel),
		OriginLng:      leg.Origin.X(),
		OriginLat:      leg.Origin.Y(),
		DestLng:        leg.Dest.X(),
		DestLat:        leg.Dest.Y(),
		SegmentLengthM: leg.SegmentLenM,
	}); err != nil {
		return fmt.Errorf("create transit leg for journey %s: %w", leg.JourneyID, err)
	}
	return nil
}

func (r *pgRepository) ListTransitLegsByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.TransitLeg, error) {
	rows, err := r.q.ListTransitLegsByJourney(ctx, journeyID)
	if err != nil {
		return nil, fmt.Errorf("list transit legs for journey %s: %w", journeyID, err)
	}
	var res []*domain.TransitLeg
	for _, row := range rows {
		geom, err := wkbToGeom(row.GeomWkb)
		if err != nil {
			return nil, fmt.Errorf("list transit legs for journey %s: decode %s: %w", journeyID, row.ID, err)
		}
		ls, ok := geom.(orb.LineString)
		if !ok {
			return nil, fmt.Errorf("transit leg %s: expected LineString geom, got %T", row.ID, geom)
		}
		res = append(res, &domain.TransitLeg{
			ID:          row.ID,
			JourneyID:   row.JourneyID,
			Seq:         int(row.Seq),
			OriginLabel: fromText(row.OriginLabel),
			DestLabel:   fromText(row.DestLabel),
			Geom:        ls,
			CreatedAt:   fromTimestamptz(row.CreatedAt),
		})
	}
	return res, nil
}

func (r *pgRepository) DeleteTransitLeg(ctx context.Context, id uuid.UUID) error {
	if err := r.q.DeleteTransitLeg(ctx, id); err != nil {
		return fmt.Errorf("delete transit leg %s: %w", id, err)
	}
	return nil
}

// DeleteMemento removes a memento; its photos cascade via the FK.
func (r *pgRepository) DeleteMemento(ctx context.Context, id uuid.UUID) error {
	affected, err := r.q.DeleteMemento(ctx, id)
	if err != nil {
		return fmt.Errorf("delete memento %s: %w", id, err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *pgRepository) GetDisplayRoute(ctx context.Context, journeyID uuid.UUID) (orb.MultiLineString, error) {
	raw, err := r.q.GetDisplayRoute(ctx, journeyID)
	if err != nil {
		return nil, fmt.Errorf("get display route for journey %s: %w", journeyID, err)
	}
	geom, err := wkbToGeom(raw)
	if err != nil {
		return nil, fmt.Errorf("get display route for journey %s: decode: %w", journeyID, err)
	}
	if geom == nil {
		return nil, nil // journey has neither a track nor legs
	}
	mls, ok := geom.(orb.MultiLineString)
	if !ok {
		return nil, fmt.Errorf("display route for journey %s: expected MultiLineString, got %T", journeyID, geom)
	}
	return mls, nil
}

func (r *pgRepository) SnapToRoute(ctx context.Context, journeyID uuid.UUID, pt orb.Point) (*orb.Point, error) {
	raw, err := r.q.SnapToRoute(ctx, db.SnapToRouteParams{
		JourneyID: journeyID,
		Lng:       pt.X(),
		Lat:       pt.Y(),
	})
	if err != nil {
		return nil, fmt.Errorf("snap to route for journey %s: %w", journeyID, err)
	}
	geom, err := wkbToGeom(raw)
	if err != nil {
		return nil, fmt.Errorf("snap to route for journey %s: decode: %w", journeyID, err)
	}
	if geom == nil {
		return nil, nil // empty route: nothing to snap to
	}
	p, ok := geom.(orb.Point)
	if !ok {
		return nil, fmt.Errorf("snap for journey %s: expected Point, got %T", journeyID, geom)
	}
	return &p, nil
}

// Photo operations

func (r *pgRepository) GetPhoto(ctx context.Context, id uuid.UUID) (*domain.MementoPhoto, error) {
	row, err := r.q.GetPhoto(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get photo %s: %w", id, err)
	}
	return &domain.MementoPhoto{
		ID:          row.ID,
		MementoID:   row.MementoID,
		ObjectKey:   row.ObjectKey,
		ContentHash: row.ContentHash,
		Caption:     fromText(row.Caption),
		Seq:         int(row.Seq),
		TakenAt:     fromTimestamptzPtr(row.TakenAt),
		SourceRef:   fromText(row.SourceRef),
		CreatedAt:   fromTimestamptz(row.CreatedAt),
	}, nil
}

func (r *pgRepository) ListPhotosByMemento(ctx context.Context, mementoID uuid.UUID) ([]*domain.MementoPhoto, error) {
	rows, err := r.q.ListPhotosByMemento(ctx, mementoID)
	if err != nil {
		return nil, fmt.Errorf("list photos for memento %s: %w", mementoID, err)
	}
	var res []*domain.MementoPhoto
	for _, row := range rows {
		res = append(res, &domain.MementoPhoto{
			ID:          row.ID,
			MementoID:   row.MementoID,
			ObjectKey:   row.ObjectKey,
			ContentHash: row.ContentHash,
			Caption:     fromText(row.Caption),
			Seq:         int(row.Seq),
			TakenAt:     fromTimestamptzPtr(row.TakenAt),
			SourceRef:   fromText(row.SourceRef),
			CreatedAt:   fromTimestamptz(row.CreatedAt),
		})
	}
	return res, nil
}

func (r *pgRepository) UpsertPhoto(ctx context.Context, photo *domain.MementoPhoto) error {
	if err := r.q.UpsertPhoto(ctx, db.UpsertPhotoParams{
		ID:          photo.ID,
		MementoID:   photo.MementoID,
		ObjectKey:   photo.ObjectKey,
		ContentHash: photo.ContentHash,
		Caption:     toText(photo.Caption),
		Seq:         int32(photo.Seq),
		TakenAt:     toTimestamptzPtr(photo.TakenAt),
		SourceRef:   toText(photo.SourceRef),
	}); err != nil {
		return fmt.Errorf("upsert photo %s: %w", photo.ID, err)
	}
	return nil
}

// wkbToGeom decodes a WKB byte slice (as returned by ST_AsBinary and scanned
// into an interface{}) into an orb geometry. A nil/empty value yields (nil, nil),
// which callers treat as "no geometry" (e.g. an empty route or a NULL snap).
func wkbToGeom(v interface{}) (orb.Geometry, error) {
	b, ok := v.([]byte)
	if !ok || len(b) == 0 {
		return nil, nil
	}
	return wkb.Unmarshal(b)
}

// Helper converters

func toDomainJourney(
	id uuid.UUID, journalID uuid.UUID, slug string, sourceRef pgtype.Text,
	title string, place string, country pgtype.Text, region pgtype.Text,
	dateStart pgtype.Date, dateEnd pgtype.Date, gpsRouteWkb interface{},
	authoredFields []string, createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz,
) (*domain.Journey, error) {
	var gpsRoute orb.MultiLineString
	if gpsRouteWkb != nil {
		if bytes, ok := gpsRouteWkb.([]byte); ok && len(bytes) > 0 {
			geom, err := wkb.Unmarshal(bytes)
			if err != nil {
				return nil, err
			}
			if mls, ok := geom.(orb.MultiLineString); ok {
				gpsRoute = mls
			}
		}
	}

	return &domain.Journey{
		ID:             id,
		JournalID:      journalID,
		Slug:           slug,
		SourceRef:      fromText(sourceRef),
		Title:          title,
		Place:          place,
		Country:        fromText(country),
		Region:         fromText(region),
		DateStart:      fromDate(dateStart),
		DateEnd:        fromDate(dateEnd),
		GPSRoute:       gpsRoute,
		AuthoredFields: authoredFields,
		CreatedAt:      fromTimestamptz(createdAt),
		UpdatedAt:      fromTimestamptz(updatedAt),
	}, nil
}

func toDomainMemento(
	id uuid.UUID, journeyID uuid.UUID, kind string, seq int32,
	occurredAt pgtype.Timestamptz, occurredTz pgtype.Text, geomWkb interface{},
	title pgtype.Text, place pgtype.Text, vendor pgtype.Text, essay pgtype.Text,
	priceAmount pgtype.Int8, priceCurrency pgtype.Text, kindData []byte,
	sourceSystem pgtype.Text, sourceExternalID pgtype.Text, sourceRef pgtype.Text,
	authoredFields []string, orphanedAt pgtype.Timestamptz, state string, revision int64,
	createdAt pgtype.Timestamptz, updatedAt pgtype.Timestamptz,
) (*domain.Memento, error) {
	var geom orb.Geometry
	if geomWkb != nil {
		if bytes, ok := geomWkb.([]byte); ok && len(bytes) > 0 {
			g, err := wkb.Unmarshal(bytes)
			if err != nil {
				return nil, err
			}
			geom = g
		}
	}

	var source *domain.SourceIdentity
	if sourceSystem.Valid && sourceExternalID.Valid {
		sourceValue := domain.SourceIdentity{System: sourceSystem.String, ExternalID: sourceExternalID.String}
		source = &sourceValue
	}

	return &domain.Memento{
		ID:             id,
		JourneyID:      journeyID,
		Kind:           kind,
		Seq:            int(seq),
		OccurredAt:     fromTimestamptz(occurredAt),
		OccurredTZ:     occurredTz.String,
		Geom:           geom,
		Title:          title.String,
		Place:          place.String,
		Vendor:         fromText(vendor),
		Essay:          fromText(essay),
		PriceAmount:    fromInt8(priceAmount),
		PriceCurrency:  fromText(priceCurrency),
		KindData:       kindData,
		SourceIdentity: source,
		SourceRef:      fromText(sourceRef),
		AuthoredFields: authoredFields,
		OrphanedAt:     fromTimestamptzPtr(orphanedAt),
		State:          domain.MementoState(state),
		Revision:       revision,
		CreatedAt:      fromTimestamptz(createdAt),
		UpdatedAt:      fromTimestamptz(updatedAt),
	}, nil
}

func toText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	return &t.String
}

func toInt8(i *int64) pgtype.Int8 {
	if i == nil {
		return pgtype.Int8{Valid: false}
	}
	return pgtype.Int8{Int64: *i, Valid: true}
}

func fromInt8(i pgtype.Int8) *int64 {
	if !i.Valid {
		return nil
	}
	return &i.Int64
}

func toTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func toTimestamptzPtr(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromTimestamptz(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func fromTimestamptzPtr(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func toDate(t time.Time) pgtype.Date {
	if t.IsZero() {
		return pgtype.Date{Valid: false}
	}
	return pgtype.Date{Time: t, Valid: true}
}

func fromDate(d pgtype.Date) time.Time {
	if !d.Valid {
		return time.Time{}
	}
	return d.Time
}
