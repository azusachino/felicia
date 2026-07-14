-- +goose Up
-- +goose StatementBegin
ALTER TABLE tb_mementos
    ADD COLUMN revision BIGINT NOT NULL DEFAULT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tb_mementos DROP COLUMN IF EXISTS revision;
-- +goose StatementEnd
