-- +goose Up
-- +goose StatementBegin
ALTER TABLE mementos
    ADD COLUMN revision BIGINT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE mementos DROP COLUMN IF EXISTS revision;
-- +goose StatementEnd
