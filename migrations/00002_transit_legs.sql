-- +goose Up
-- +goose StatementBegin
-- Authored transit legs (flights, ferries, or GPS-gap fills). Kept SEPARATE from
-- tb_journeys.gps_route: the Dawarich track stays untouched and re-import-safe, while
-- the display route is composed at read time as gps_route ∪ ST_Collect(leg geoms)
-- (data-model.md D2, union-at-read). Each leg's geom is a great-circle arc built
-- once at insert via ST_Segmentize on geography (ADR 0009).
CREATE TABLE tb_transit_legs (
    id UUID PRIMARY KEY DEFAULT generate_uuid_v7(),
    journey_id UUID NOT NULL REFERENCES tb_journeys(id) ON DELETE CASCADE,
    seq INT NOT NULL DEFAULT 0,
    origin_label TEXT,
    dest_label TEXT,
    geom GEOMETRY(LineString, 4326) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_transit_legs_journey_seq ON tb_transit_legs(journey_id, seq);
CREATE INDEX idx_transit_legs_geom ON tb_transit_legs USING GIST(geom);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_transit_legs_geom;
DROP INDEX IF EXISTS idx_transit_legs_journey_seq;
DROP TABLE IF EXISTS tb_transit_legs;
-- +goose StatementEnd
