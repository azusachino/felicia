package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/paulmach/orb"

	"github.com/azusachino/felicia/core/domain"
)

const stopCandidateColumns = `id, journey_id, derivation_version, candidate_key, label, authored_fields, geom, arrive, depart, confidence, state, merged_into, provenance, revision, created_at, updated_at`

// GetStopCandidate retrieves one private intake candidate.
func (r *Repository) GetStopCandidate(ctx context.Context, id uuid.UUID) (*domain.StopCandidate, error) {
	candidate, err := r.scanStopCandidate(r.db.QueryRowContext(ctx, "SELECT "+stopCandidateColumns+" FROM tb_stop_candidates WHERE id = ?", idString(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get stop candidate %s: %w", id, err)
	}
	if err := r.loadStopCandidateEvidence(ctx, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

// ListStopCandidatesByJourney returns candidates in chronological order.
func (r *Repository) ListStopCandidatesByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.StopCandidate, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT "+stopCandidateColumns+" FROM tb_stop_candidates WHERE journey_id = ? ORDER BY arrive, id", idString(journeyID))
	if err != nil {
		return nil, fmt.Errorf("list stop candidates for journey %s: %w", journeyID, err)
	}
	defer func() { _ = rows.Close() }()
	var result []*domain.StopCandidate
	for rows.Next() {
		candidate, err := r.scanStopCandidate(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Evidence loads only after the candidate rows are fully drained and
	// closed: the pool is capped at one connection (store.go), so a nested
	// query while `rows` is still open deadlocks until the context expires.
	_ = rows.Close()
	for _, candidate := range result {
		if err := r.loadStopCandidateEvidence(ctx, candidate); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// UpsertStopCandidate refreshes source-owned fields while preserving review state.
func (r *Repository) UpsertStopCandidate(ctx context.Context, candidate *domain.StopCandidate) error {
	if candidate == nil {
		return errors.New("stop candidate is required")
	}
	if candidate.JourneyID == uuid.Nil || candidate.Identity.DerivationVersion == "" || candidate.Identity.Key == "" {
		return errors.New("stop candidate journey and identity are required")
	}
	if candidate.Coord == (domain.StopCandidate{}).Coord {
		return errors.New("stop candidate coordinate is required")
	}
	if candidate.Depart.Before(candidate.Arrive) {
		return errors.New("stop candidate depart must not precede arrive")
	}
	if candidate.ID == uuid.Nil {
		candidate.ID = uuid.Must(uuid.NewV7())
	}
	if candidate.State == "" {
		candidate.State = domain.CandidateProposed
	}
	if !validCandidateState(candidate.State) {
		return fmt.Errorf("invalid stop candidate state %q", candidate.State)
	}
	geom, err := encodeGeometry(candidate.Coord)
	if err != nil {
		return fmt.Errorf("encode stop candidate geometry: %w", err)
	}
	provenance, err := marshalJSON(candidate.Provenance)
	if err != nil {
		return fmt.Errorf("encode stop candidate provenance: %w", err)
	}
	authoredFields, err := stringsJSON(candidate.AuthoredFields)
	if err != nil {
		return fmt.Errorf("encode stop candidate authored fields: %w", err)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stop candidate upsert: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO tb_stop_candidates(id, journey_id, derivation_version, candidate_key, label, authored_fields, geom, arrive, depart, confidence, state, provenance, revision, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, ?, ?)
ON CONFLICT(journey_id, derivation_version, candidate_key) DO UPDATE SET
  label=CASE WHEN instr(tb_stop_candidates.authored_fields, '"label"') > 0 THEN tb_stop_candidates.label ELSE excluded.label END,
  geom=excluded.geom, arrive=excluded.arrive, depart=excluded.depart,
  confidence=excluded.confidence, provenance=excluded.provenance, revision=tb_stop_candidates.revision+1,
  updated_at=excluded.updated_at`, idString(candidate.ID), idString(candidate.JourneyID), candidate.Identity.DerivationVersion, candidate.Identity.Key,
		candidate.Label, authoredFields, geom, candidate.Arrive.UTC().Format(time.RFC3339Nano), candidate.Depart.UTC().Format(time.RFC3339Nano), candidate.Confidence, candidate.State, provenance, timeOrNow(candidate.CreatedAt), timeOrNow(candidate.UpdatedAt))
	if err != nil {
		return fmt.Errorf("upsert stop candidate %s: %w", candidate.ID, err)
	}
	var rawID, created, updated string
	var rawMerged sql.NullString
	if err := tx.QueryRowContext(ctx, "SELECT id, revision, state, merged_into, created_at, updated_at FROM tb_stop_candidates WHERE journey_id = ? AND derivation_version = ? AND candidate_key = ?", idString(candidate.JourneyID), candidate.Identity.DerivationVersion, candidate.Identity.Key).Scan(&rawID, &candidate.Revision, &candidate.State, &rawMerged, &created, &updated); err != nil {
		return fmt.Errorf("read upserted stop candidate: %w", err)
	}
	parsedID, err := parseID(rawID)
	if err != nil {
		return err
	}
	candidate.ID = parsedID
	candidate.CreatedAt = readTime(sql.NullString{String: created, Valid: true})
	candidate.UpdatedAt = readTime(sql.NullString{String: updated, Valid: true})
	if rawMerged.Valid {
		parsedMerged, err := parseID(rawMerged.String)
		if err != nil {
			return err
		}
		candidate.MergedInto = &parsedMerged
	}
	// Source evidence is a complete snapshot for this candidate. Review state is
	// stored on the candidate row and is intentionally untouched above.
	if _, err := tx.ExecContext(ctx, "DELETE FROM tb_stop_candidate_evidence WHERE candidate_id = ?", idString(candidate.ID)); err != nil {
		return fmt.Errorf("replace stop candidate evidence: %w", err)
	}
	for _, evidence := range candidate.Evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("stop candidate evidence: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO tb_stop_candidate_evidence(candidate_id, kind, source_system, source_external_id, locator) VALUES (?, ?, ?, ?, ?)`, idString(candidate.ID), evidence.Kind, evidence.Source.System, evidence.Source.ExternalID, evidence.Locator); err != nil {
			return fmt.Errorf("insert stop candidate evidence: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit stop candidate upsert: %w", err)
	}
	return nil
}

// ApplyStopReview changes only review-owned fields and uses optimistic concurrency.
func (r *Repository) ApplyStopReview(ctx context.Context, patch *domain.StopReviewPatch) error {
	if patch == nil || patch.CandidateID == uuid.Nil {
		return errors.New("stop review patch is required")
	}
	if patch.State != "" && !validCandidateState(patch.State) {
		return fmt.Errorf("invalid stop candidate state %q", patch.State)
	}
	current, err := r.GetStopCandidate(ctx, patch.CandidateID)
	if err != nil {
		return err
	}
	if patch.ExpectedRevision != nil && current.Revision != *patch.ExpectedRevision {
		return domain.ErrWriteConflict
	}
	state := current.State
	if patch.State != "" {
		state = patch.State
	}
	label := current.Label
	if patch.Label != nil {
		label = *patch.Label
	}
	mergedInto := current.MergedInto
	if patch.MergedInto != nil {
		mergedInto = patch.MergedInto
	}
	if state == domain.CandidateMerged && mergedInto == nil {
		return errors.New("merged stop candidate requires a target")
	}
	authoredFields := current.AuthoredFields
	if patch.Label != nil {
		authoredFields = unionFields(authoredFields, []string{"label"})
	}
	fieldsJSON, err := stringsJSON(authoredFields)
	if err != nil {
		return err
	}
	result, err := r.db.ExecContext(ctx, `UPDATE tb_stop_candidates SET state = ?, label = ?, authored_fields = ?, merged_into = ?, revision = revision + 1, updated_at = ? WHERE id = ? AND revision = ?`, state, label, fieldsJSON, nullableUUID(mergedInto), now(), idString(patch.CandidateID), current.Revision)
	if err != nil {
		return fmt.Errorf("apply stop review %s: %w", patch.CandidateID, err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return domain.ErrWriteConflict
	}
	return nil
}

type stopCandidateScanner interface{ Scan(...any) error }

func (r *Repository) scanStopCandidate(row stopCandidateScanner) (*domain.StopCandidate, error) {
	var rawID, rawJourney, derivation, key, geom, arrive, depart, provenance, created, updated string
	var authoredFields string
	var rawMerged sql.NullString
	var label, state sql.NullString
	var confidence float64
	var revision int64
	if err := row.Scan(&rawID, &rawJourney, &derivation, &key, &label, &authoredFields, &geom, &arrive, &depart, &confidence, &state, &rawMerged, &provenance, &revision, &created, &updated); err != nil {
		return nil, err
	}
	return scanCandidateValues(rawID, rawJourney, derivation, key, label.String, authoredFields, geom, arrive, depart, confidence, state.String, rawMerged.String, provenance, revision, created, updated)
}

func (r *Repository) loadStopCandidateEvidence(ctx context.Context, candidate *domain.StopCandidate) error {
	rows, err := r.db.QueryContext(ctx, "SELECT kind, source_system, source_external_id, locator FROM tb_stop_candidate_evidence WHERE candidate_id = ? ORDER BY id", idString(candidate.ID))
	if err != nil {
		return fmt.Errorf("list stop candidate evidence: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var kind domain.EvidenceKind
		var system, external, locator string
		if err := rows.Scan(&kind, &system, &external, &locator); err != nil {
			return err
		}
		candidate.Evidence = append(candidate.Evidence, domain.EvidenceRef{Kind: kind, Source: domain.SourceIdentity{System: system, ExternalID: external}, Locator: locator})
	}
	return rows.Err()
}

func scanCandidateValues(rawID, rawJourney, derivation, key, label, authoredFields, geom, arrive, depart string, confidence float64, state, rawMerged, provenance string, revision int64, created, updated string) (*domain.StopCandidate, error) {
	id, err := parseID(rawID)
	if err != nil {
		return nil, err
	}
	journeyID, err := parseID(rawJourney)
	if err != nil {
		return nil, err
	}
	decoded, err := decodeGeometry(sql.NullString{String: geom, Valid: geom != ""})
	if err != nil {
		return nil, err
	}
	coord, ok := decoded.(orb.Point)
	if !ok {
		return nil, errors.New("stop candidate geometry is not a point")
	}
	var prov []domain.Provenance
	if err := unmarshalJSON(provenance, &prov); err != nil {
		return nil, err
	}
	fields, err := parseStrings(authoredFields)
	if err != nil {
		return nil, err
	}
	var merged *uuid.UUID
	if rawMerged != "" {
		value, err := parseID(rawMerged)
		if err != nil {
			return nil, err
		}
		merged = &value
	}
	return &domain.StopCandidate{ID: id, JourneyID: journeyID, Identity: domain.CandidateIdentity{DerivationVersion: derivation, Key: key}, Label: label, AuthoredFields: fields, Coord: coord, Arrive: readTime(sql.NullString{String: arrive, Valid: true}), Depart: readTime(sql.NullString{String: depart, Valid: true}), Confidence: confidence, State: domain.CandidateState(state), MergedInto: merged, Provenance: prov, Revision: revision, CreatedAt: readTime(sql.NullString{String: created, Valid: true}), UpdatedAt: readTime(sql.NullString{String: updated, Valid: true})}, nil
}

func validCandidateState(state domain.CandidateState) bool {
	return state == domain.CandidateProposed || state == domain.CandidateKept || state == domain.CandidateIgnored || state == domain.CandidateMerged
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return idString(*value)
}
