-- +goose Up
-- +goose StatementBegin
CREATE TABLE tb_stop_candidates (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    journey_id UUID NOT NULL REFERENCES tb_journeys(id) ON DELETE CASCADE,
    derivation_version TEXT NOT NULL,
    candidate_key TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    authored_fields TEXT[] NOT NULL DEFAULT '{}',
    geom GEOMETRY(Point, 4326) NOT NULL,
    arrive TIMESTAMPTZ NOT NULL,
    depart TIMESTAMPTZ NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    state TEXT NOT NULL DEFAULT 'proposed',
    merged_into UUID REFERENCES tb_stop_candidates(id),
    provenance JSONB NOT NULL DEFAULT '[]',
    revision BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_stop_candidate_identity UNIQUE (journey_id, derivation_version, candidate_key),
    CONSTRAINT valid_stop_candidate_state CHECK (state IN ('proposed', 'kept', 'ignored', 'merged')),
    CONSTRAINT valid_stop_candidate_interval CHECK (depart >= arrive)
);

CREATE TABLE tb_stop_candidate_evidence (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    candidate_id UUID NOT NULL REFERENCES tb_stop_candidates(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    source_system TEXT NOT NULL,
    source_external_id TEXT NOT NULL,
    locator TEXT NOT NULL,
    CONSTRAINT unique_stop_candidate_evidence UNIQUE (candidate_id, kind, source_system, source_external_id, locator)
);
CREATE INDEX idx_stop_candidates_journey_arrive ON tb_stop_candidates(journey_id, arrive);
CREATE INDEX idx_stop_candidate_evidence_candidate ON tb_stop_candidate_evidence(candidate_id);
CREATE INDEX idx_stop_candidates_geom ON tb_stop_candidates USING GIST(geom);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_stop_candidates_geom;
DROP INDEX IF EXISTS idx_stop_candidate_evidence_candidate;
DROP INDEX IF EXISTS idx_stop_candidates_journey_arrive;
DROP TABLE IF EXISTS tb_stop_candidate_evidence;
DROP TABLE IF EXISTS tb_stop_candidates;
-- +goose StatementEnd
