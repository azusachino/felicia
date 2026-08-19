-- +goose Up
-- +goose StatementBegin
CREATE TABLE tb_site_settings (
    journal_id        UUID PRIMARY KEY REFERENCES tb_journal(id) ON DELETE CASCADE,
    title             TEXT NOT NULL DEFAULT '',
    description       TEXT NOT NULL DEFAULT '',
    design            TEXT NOT NULL DEFAULT 'v1' CHECK (design IN ('v1','v2','v3','v4')),
    default_language  TEXT NOT NULL DEFAULT 'ja' CHECK (default_language IN ('ja','en','zh')),
    default_theme     TEXT NOT NULL DEFAULT 'dark' CHECK (default_theme IN ('dark','light')),
    accent            TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tb_site_settings;
-- +goose StatementEnd
