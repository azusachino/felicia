-- +goose Up
-- +goose StatementBegin
ALTER TABLE mementos
    ADD COLUMN state TEXT NOT NULL DEFAULT 'published';

ALTER TABLE mementos
    ADD CONSTRAINT mementos_valid_state
    CHECK (state IN ('candidate', 'draft', 'authored', 'published', 'archived'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mementos DROP CONSTRAINT IF EXISTS mementos_valid_state;
ALTER TABLE mementos DROP COLUMN IF EXISTS state;
-- +goose StatementEnd
