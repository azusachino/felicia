-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE OR REPLACE FUNCTION generate_uuid_v7() RETURNS uuid AS $$
DECLARE
    timestamp    timestamp with time zone;
    timestamp_ms bigint;
    uuid_bytes   bytea;
BEGIN
    -- Get current time and convert to milliseconds since epoch
    timestamp := clock_timestamp();
    timestamp_ms := floor(extract(epoch from timestamp) * 1000)::bigint;

    -- Construct UUIDv7 byte array (16 bytes):
    -- 48 bits (6 bytes) timestamp
    -- 4 bits version (0111 = 7)
    -- 12 bits sequence/random
    -- 2 bits variant (10 = RFC 4122)
    -- 62 bits random
    
    -- Generate 16 random bytes
    uuid_bytes := gen_random_bytes(16);
    
    -- Overwrite first 6 bytes with timestamp_ms (48 bits)
    uuid_bytes := set_byte(uuid_bytes, 0, (timestamp_ms >> 40 & 255)::int);
    uuid_bytes := set_byte(uuid_bytes, 1, (timestamp_ms >> 32 & 255)::int);
    uuid_bytes := set_byte(uuid_bytes, 2, (timestamp_ms >> 24 & 255)::int);
    uuid_bytes := set_byte(uuid_bytes, 3, (timestamp_ms >> 16 & 255)::int);
    uuid_bytes := set_byte(uuid_bytes, 4, (timestamp_ms >> 8 & 255)::int);
    uuid_bytes := set_byte(uuid_bytes, 5, (timestamp_ms & 255)::int);
    
    -- Overwrite version bits (set 4 MSB of byte 6 to 7 => 0111xxxx)
    uuid_bytes := set_byte(uuid_bytes, 6, (get_byte(uuid_bytes, 6) & 15 | 112)::int);
    
    -- Overwrite variant bits (set 2 MSB of byte 8 to 2 => 10xxxxxx)
    uuid_bytes := set_byte(uuid_bytes, 8, (get_byte(uuid_bytes, 8) & 63 | 128)::int);
    
    RETURN encode(uuid_bytes, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;

CREATE TABLE journal (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE journeys (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
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
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
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

CREATE TABLE memento_photos (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
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
DROP TABLE IF EXISTS mementos;
DROP TABLE IF EXISTS journeys;
DROP TABLE IF EXISTS journal;

DROP FUNCTION IF EXISTS generate_uuid_v7();
-- +goose StatementEnd
