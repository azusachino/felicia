-- +goose Up
-- +goose StatementBegin
ALTER TABLE tb_mementos
    ADD COLUMN state TEXT NOT NULL DEFAULT 'published';

ALTER TABLE tb_mementos
    ADD CONSTRAINT mementos_valid_state
    CHECK (state IN ('candidate', 'draft', 'authored', 'published', 'archived'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tb_mementos DROP CONSTRAINT IF EXISTS mementos_valid_state;
ALTER TABLE tb_mementos DROP COLUMN IF EXISTS state;
-- +goose StatementEnd
