package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/felicia-core/domain"
)

// CreateImportRun starts a source synchronization run.
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

// FinishImportRun records a source synchronization outcome.
func (r *Repository) FinishImportRun(ctx context.Context, id uuid.UUID, status domain.ImportRunStatus, finishedAt time.Time, errorMessage *string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE tb_import_runs SET status = ?, finished_at = ?, error_message = ? WHERE id = ?`, string(status), nullableTime(finishedAt), nullableString(errorMessage), idString(id))
	return err
}

// RecordSourceObservation persists a canonical source snapshot.
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

// MarkMissingSourceObservations marks source objects absent from a new run.
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
