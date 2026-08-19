package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/encoding/wkb"

	"github.com/azusachino/felicia/core/domain"
	"github.com/azusachino/felicia/core/ports"
)

var _ ports.StopCandidateStore = (*pgRepository)(nil)

const stopCandidateSelect = `SELECT id, journey_id, derivation_version, candidate_key, label, authored_fields, ST_AsBinary(geom), arrive, depart, confidence, state, merged_into, provenance, revision, created_at, updated_at FROM tb_stop_candidates`

func (r *pgRepository) GetStopCandidate(ctx context.Context, id uuid.UUID) (*domain.StopCandidate, error) {
	row := r.db.QueryRow(ctx, stopCandidateSelect+" WHERE id = $1", id)
	candidate, err := scanPostgresStopCandidate(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get stop candidate %s: %w", id, err)
	}
	if err := r.loadPostgresStopCandidateEvidence(ctx, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (r *pgRepository) ListStopCandidatesByJourney(ctx context.Context, journeyID uuid.UUID) ([]*domain.StopCandidate, error) {
	rows, err := r.db.Query(ctx, stopCandidateSelect+" WHERE journey_id = $1 ORDER BY arrive, id", journeyID)
	if err != nil {
		return nil, fmt.Errorf("list stop candidates for journey %s: %w", journeyID, err)
	}
	defer rows.Close()
	var result []*domain.StopCandidate
	for rows.Next() {
		candidate, err := scanPostgresStopCandidate(rows)
		if err != nil {
			return nil, err
		}
		if err := r.loadPostgresStopCandidateEvidence(ctx, candidate); err != nil {
			return nil, err
		}
		result = append(result, candidate)
	}
	return result, rows.Err()
}

func (r *pgRepository) UpsertStopCandidate(ctx context.Context, candidate *domain.StopCandidate) error {
	if candidate == nil {
		return errors.New("stop candidate is required")
	}
	if candidate.JourneyID == uuid.Nil || candidate.Identity.DerivationVersion == "" || candidate.Identity.Key == "" {
		return errors.New("stop candidate journey and identity are required")
	}
	if candidate.Depart.Before(candidate.Arrive) {
		return errors.New("stop candidate depart must not precede arrive")
	}
	if candidate.State == "" {
		candidate.State = domain.CandidateProposed
	}
	if !validPostgresCandidateState(candidate.State) {
		return fmt.Errorf("invalid stop candidate state %q", candidate.State)
	}
	if candidate.ID == uuid.Nil {
		candidate.ID = uuid.Must(uuid.NewV7())
	}
	geom, err := wkb.Marshal(candidate.Coord)
	if err != nil {
		return fmt.Errorf("encode stop candidate geometry: %w", err)
	}
	provenance, err := json.Marshal(candidate.Provenance)
	if err != nil {
		return fmt.Errorf("encode stop candidate provenance: %w", err)
	}
	_, err = r.db.Exec(ctx, `INSERT INTO tb_stop_candidates(id, journey_id, derivation_version, candidate_key, label, geom, arrive, depart, confidence, state, provenance)
VALUES ($1, $2, $3, $4, $5, ST_GeomFromWKB($6, 4326), $7, $8, $9, $10, $11)
ON CONFLICT (journey_id, derivation_version, candidate_key) DO UPDATE SET
 label = CASE WHEN tb_stop_candidates.authored_fields @> ARRAY['label'] THEN tb_stop_candidates.label ELSE EXCLUDED.label END,
 geom = EXCLUDED.geom, arrive = EXCLUDED.arrive, depart = EXCLUDED.depart,
	confidence = EXCLUDED.confidence, provenance = EXCLUDED.provenance, revision = tb_stop_candidates.revision + 1, updated_at = NOW()`,
		candidate.ID, candidate.JourneyID, candidate.Identity.DerivationVersion, candidate.Identity.Key, candidate.Label, geom, toTimestamptz(candidate.Arrive), toTimestamptz(candidate.Depart), candidate.Confidence, candidate.State, provenance)
	if err != nil {
		return fmt.Errorf("upsert stop candidate %s: %w", candidate.ID, err)
	}
	current, err := r.GetStopCandidate(ctx, candidate.ID)
	if err != nil {
		// Re-imports may supply a fresh ID for an existing stable identity.
		current, err = r.getPostgresStopCandidateByIdentity(ctx, candidate.JourneyID, candidate.Identity)
	}
	if err != nil {
		return err
	}
	evidence := candidate.Evidence
	*candidate = *current
	if _, err := r.db.Exec(ctx, "DELETE FROM tb_stop_candidate_evidence WHERE candidate_id = $1", candidate.ID); err != nil {
		return fmt.Errorf("replace stop candidate evidence: %w", err)
	}
	for _, evidence := range evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("stop candidate evidence: %w", err)
		}
		if _, err := r.db.Exec(ctx, `INSERT INTO tb_stop_candidate_evidence(candidate_id, kind, source_system, source_external_id, locator) VALUES ($1, $2, $3, $4, $5)`, candidate.ID, evidence.Kind, evidence.Source.System, evidence.Source.ExternalID, evidence.Locator); err != nil {
			return fmt.Errorf("insert stop candidate evidence: %w", err)
		}
	}
	return nil
}

func (r *pgRepository) ApplyStopReview(ctx context.Context, patch *domain.StopReviewPatch) error {
	if patch == nil || patch.CandidateID == uuid.Nil {
		return errors.New("stop review patch is required")
	}
	if patch.State != "" && !validPostgresCandidateState(patch.State) {
		return fmt.Errorf("invalid stop candidate state %q", patch.State)
	}
	current, err := r.GetStopCandidate(ctx, patch.CandidateID)
	if err != nil {
		return err
	}
	if patch.ExpectedRevision != nil && current.Revision != *patch.ExpectedRevision {
		return domain.ErrWriteConflict
	}
	state, label, merged := current.State, current.Label, current.MergedInto
	if patch.State != "" {
		state = patch.State
	}
	if patch.Label != nil {
		label = *patch.Label
	}
	if patch.MergedInto != nil {
		merged = patch.MergedInto
	}
	if state == domain.CandidateMerged && merged == nil {
		return errors.New("merged stop candidate requires a target")
	}
	authoredFields := current.AuthoredFields
	if patch.Label != nil {
		authoredFields = appendUniqueString(authoredFields, "label")
	}
	result, err := r.db.Exec(ctx, `UPDATE tb_stop_candidates SET state = $1, label = $2, authored_fields = $3, merged_into = $4, revision = revision + 1, updated_at = NOW() WHERE id = $5 AND revision = $6`, state, label, authoredFields, merged, patch.CandidateID, current.Revision)
	if err != nil {
		return fmt.Errorf("apply stop review %s: %w", patch.CandidateID, err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrWriteConflict
	}
	return nil
}

type postgresRow interface{ Scan(...any) error }

func scanPostgresStopCandidate(row postgresRow) (*domain.StopCandidate, error) {
	var id, journeyID uuid.UUID
	var derivation, key, label, state string
	var authoredFields []string
	var geom []byte
	var arrive, depart pgtype.Timestamptz
	var confidence float64
	var merged *uuid.UUID
	var provenance []byte
	var revision int64
	var created, updated pgtype.Timestamptz
	if err := row.Scan(&id, &journeyID, &derivation, &key, &label, &authoredFields, &geom, &arrive, &depart, &confidence, &state, &merged, &provenance, &revision, &created, &updated); err != nil {
		return nil, err
	}
	decoded, err := wkb.Unmarshal(geom)
	if err != nil {
		return nil, err
	}
	coord, ok := decoded.(orb.Point)
	if !ok {
		return nil, errors.New("stop candidate geometry is not a point")
	}
	var prov []domain.Provenance
	if err := json.Unmarshal(provenance, &prov); err != nil {
		return nil, err
	}
	return &domain.StopCandidate{ID: id, JourneyID: journeyID, Identity: domain.CandidateIdentity{DerivationVersion: derivation, Key: key}, Label: label, AuthoredFields: authoredFields, Coord: coord, Arrive: fromTimestamptz(arrive), Depart: fromTimestamptz(depart), Confidence: confidence, State: domain.CandidateState(state), MergedInto: merged, Provenance: prov, Revision: revision, CreatedAt: fromTimestamptz(created), UpdatedAt: fromTimestamptz(updated)}, nil
}

func (r *pgRepository) getPostgresStopCandidateByIdentity(ctx context.Context, journeyID uuid.UUID, identity domain.CandidateIdentity) (*domain.StopCandidate, error) {
	candidate, err := scanPostgresStopCandidate(r.db.QueryRow(ctx, stopCandidateSelect+" WHERE journey_id = $1 AND derivation_version = $2 AND candidate_key = $3", journeyID, identity.DerivationVersion, identity.Key))
	if err != nil {
		return nil, err
	}
	if err := r.loadPostgresStopCandidateEvidence(ctx, candidate); err != nil {
		return nil, err
	}
	return candidate, nil
}

func (r *pgRepository) loadPostgresStopCandidateEvidence(ctx context.Context, candidate *domain.StopCandidate) error {
	rows, err := r.db.Query(ctx, "SELECT kind, source_system, source_external_id, locator FROM tb_stop_candidate_evidence WHERE candidate_id = $1 ORDER BY id", candidate.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, system, external, locator string
		if err := rows.Scan(&kind, &system, &external, &locator); err != nil {
			return err
		}
		candidate.Evidence = append(candidate.Evidence, domain.EvidenceRef{Kind: domain.EvidenceKind(kind), Source: domain.SourceIdentity{System: system, ExternalID: external}, Locator: locator})
	}
	return rows.Err()
}

func validPostgresCandidateState(state domain.CandidateState) bool {
	return state == domain.CandidateProposed || state == domain.CandidateKept || state == domain.CandidateIgnored || state == domain.CandidateMerged
}

func appendUniqueString(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
