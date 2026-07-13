-- +goose Up
-- +goose StatementBegin
ALTER TABLE mementos
    ADD COLUMN source_system TEXT,
    ADD COLUMN source_external_id TEXT;

-- Existing adapter refs use the stable `system:external-id` shape. Keep the
-- complete legacy ref while backfilling the normalized identity columns.
UPDATE mementos
SET source_system = split_part(source_ref, ':', 1),
    source_external_id = substring(source_ref FROM position(':' IN source_ref) + 1)
WHERE source_ref IS NOT NULL
  AND position(':' IN source_ref) > 1;

ALTER TABLE mementos
    ADD CONSTRAINT mementos_source_identity_pair
    CHECK ((source_system IS NULL AND source_external_id IS NULL)
        OR (source_system IS NOT NULL AND source_external_id IS NOT NULL
            AND source_system <> '' AND source_external_id <> ''));

CREATE UNIQUE INDEX mementos_source_identity_unique
    ON mementos (source_system, source_external_id)
    WHERE source_system IS NOT NULL AND source_external_id IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS mementos_source_identity_unique;
ALTER TABLE mementos DROP CONSTRAINT IF EXISTS mementos_source_identity_pair;
ALTER TABLE mementos
    DROP COLUMN IF EXISTS source_system,
    DROP COLUMN IF EXISTS source_external_id;
-- +goose StatementEnd
