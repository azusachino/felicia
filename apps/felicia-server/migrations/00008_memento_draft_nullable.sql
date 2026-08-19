-- +goose Up
-- +goose StatementBegin
-- Draft mementos may be saved before occurrence, location, or authored prose
-- is complete. Publication validation remains lifecycle-aware in the runtime.
ALTER TABLE tb_mementos
    ALTER COLUMN occurred_at DROP NOT NULL,
    ALTER COLUMN occurred_tz DROP NOT NULL,
    ALTER COLUMN geom DROP NOT NULL,
    ALTER COLUMN title DROP NOT NULL,
    ALTER COLUMN place DROP NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverting requires all existing drafts to be completed first.
ALTER TABLE tb_mementos
    ALTER COLUMN occurred_at SET NOT NULL,
    ALTER COLUMN occurred_tz SET NOT NULL,
    ALTER COLUMN geom SET NOT NULL,
    ALTER COLUMN title SET NOT NULL,
    ALTER COLUMN place SET NOT NULL;
-- +goose StatementEnd
