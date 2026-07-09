-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE journal (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE journeys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journal_id UUID NOT NULL REFERENCES journal(id) ON DELETE CASCADE,
    slug TEXT NOT NULL UNIQUE,
    source_ref TEXT,
    title TEXT NOT NULL,
    place TEXT NOT NULL,
    country VARCHAR(3),
    region TEXT,
    date_start DATE NOT NULL,
    date_end DATE NOT NULL,
    gps_route GEOMETRY(MultiLineString, 4326),
    authored_fields TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_journal_source UNIQUE (journal_id, source_ref)
);

CREATE TABLE mementos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    journey_id UUID NOT NULL REFERENCES journeys(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    seq INT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    occurred_tz TEXT NOT NULL,
    geom GEOMETRY(Geometry, 4326) NOT NULL,
    title TEXT NOT NULL,
    place TEXT NOT NULL,
    vendor TEXT,
    essay TEXT,
    price_amount BIGINT,
    price_currency CHAR(3),
    kind_data JSONB NOT NULL DEFAULT '{}',
    source_ref TEXT,
    authored_fields TEXT[] NOT NULL DEFAULT '{}',
    orphaned_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_journey_source_memento UNIQUE (journey_id, source_ref),
    CONSTRAINT valid_currency CHECK (price_currency IS NULL OR price_currency ~ '^[A-Z]{3}$')
);

CREATE TABLE translations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_type TEXT NOT NULL CHECK (owner_type IN ('journey', 'memento', 'photo')),
    owner_id UUID NOT NULL,
    lang TEXT NOT NULL CHECK (lang IN ('en', 'zh')),
    field TEXT NOT NULL,
    value TEXT NOT NULL,
    provenance TEXT NOT NULL CHECK (provenance IN ('machine', 'authored')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_translation UNIQUE (owner_type, owner_id, lang, field)
);

CREATE TABLE memento_photos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    memento_id UUID NOT NULL REFERENCES mementos(id) ON DELETE CASCADE,
    object_key TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    caption TEXT,
    seq INT NOT NULL DEFAULT 0,
    taken_at TIMESTAMPTZ,
    source_ref TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_memento_source_photo UNIQUE (memento_id, source_ref),
    CONSTRAINT unique_memento_photo_hash UNIQUE (memento_id, content_hash)
);

CREATE INDEX idx_journeys_gps_route ON journeys USING GIST(gps_route);
CREATE INDEX idx_mementos_geom ON mementos USING GIST(geom);
CREATE INDEX idx_mementos_journey_seq ON mementos(journey_id, seq);
CREATE INDEX idx_mementos_kind ON mementos(kind);
CREATE INDEX idx_mementos_occurred ON mementos(occurred_at DESC);
CREATE INDEX idx_memento_photos_memento_seq ON memento_photos(memento_id, seq);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_memento_photos_memento_seq;
DROP INDEX IF EXISTS idx_mementos_occurred;
DROP INDEX IF EXISTS idx_mementos_kind;
DROP INDEX IF EXISTS idx_mementos_journey_seq;
DROP INDEX IF EXISTS idx_mementos_geom;
DROP INDEX IF EXISTS idx_journeys_gps_route;

DROP TABLE IF EXISTS memento_photos;
DROP TABLE IF EXISTS translations;
DROP TABLE IF EXISTS mementos;
DROP TABLE IF EXISTS journeys;
DROP TABLE IF EXISTS journal;
-- +goose StatementEnd
