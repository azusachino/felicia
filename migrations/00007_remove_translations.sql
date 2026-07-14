-- +goose Up
-- Prefix legacy application tables for consistent provider naming. The
-- IF EXISTS guards make this safe for fresh databases whose initial schema
-- already uses the tb_ names.
ALTER TABLE IF EXISTS journal RENAME TO tb_journal;
ALTER TABLE IF EXISTS journeys RENAME TO tb_journeys;
ALTER TABLE IF EXISTS mementos RENAME TO tb_mementos;
ALTER TABLE IF EXISTS memento_photos RENAME TO tb_memento_photos;
ALTER TABLE IF EXISTS transit_legs RENAME TO tb_transit_legs;
ALTER TABLE IF EXISTS import_runs RENAME TO tb_import_runs;
ALTER TABLE IF EXISTS source_observations RENAME TO tb_source_observations;

DROP TABLE IF EXISTS translations;

-- +goose Down
-- Translation storage was intentionally removed from the product model.
-- Restoring it requires a new explicit schema decision and migration.
