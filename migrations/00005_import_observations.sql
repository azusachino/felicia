-- +goose Up
-- +goose StatementBegin
CREATE TABLE import_runs (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    source_system TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed')),
    error_message TEXT
);

CREATE TABLE source_observations (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    run_id UUID NOT NULL REFERENCES import_runs(id) ON DELETE CASCADE,
    source_system TEXT NOT NULL,
    source_external_id TEXT NOT NULL,
    kind TEXT NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL,
    confidence DOUBLE PRECISION NOT NULL CHECK (confidence >= 0 AND confidence <= 1),
    payload JSONB NOT NULL,
    changed BOOLEAN NOT NULL DEFAULT FALSE,
    orphaned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT source_observation_identity_nonempty CHECK (source_system <> '' AND source_external_id <> ''),
    CONSTRAINT unique_run_source_observation UNIQUE (run_id, source_system, source_external_id)
);

CREATE INDEX source_observations_identity_idx
    ON source_observations (source_system, source_external_id, observed_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS source_observations;
DROP TABLE IF EXISTS import_runs;
-- +goose StatementEnd
