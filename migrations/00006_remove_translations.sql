-- +goose Up
DROP TABLE IF EXISTS translations;

-- +goose Down
-- Translation storage was intentionally removed from the product model.
-- Restoring it requires a new explicit schema decision and migration.
