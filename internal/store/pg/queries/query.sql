-- name: GetJournal :one
SELECT id, created_at FROM journal WHERE id = $1;

-- name: CreateImportRun :exec
INSERT INTO import_runs (id, source_system, started_at, status, error_message)
VALUES ($1, $2, $3, $4, $5);

-- name: FinishImportRun :exec
UPDATE import_runs
SET status = $2, finished_at = $3, error_message = $4
WHERE id = $1;

-- name: RecordSourceObservation :exec
INSERT INTO source_observations (
    id, run_id, source_system, source_external_id, kind, observed_at,
    confidence, payload, changed, orphaned_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8,
    COALESCE((
        SELECT payload IS DISTINCT FROM $8::jsonb
        FROM source_observations
        WHERE source_system = $3 AND source_external_id = $4
        ORDER BY observed_at DESC, created_at DESC
        LIMIT 1
    ), FALSE), NULL
)
ON CONFLICT (run_id, source_system, source_external_id) DO UPDATE SET
    kind = EXCLUDED.kind,
    observed_at = EXCLUDED.observed_at,
    confidence = EXCLUDED.confidence,
    payload = EXCLUDED.payload,
    changed = EXCLUDED.changed,
    orphaned_at = NULL;

-- name: MarkMissingSourceObservations :exec
UPDATE source_observations
SET orphaned_at = NOW()
WHERE source_system = $2
  AND run_id <> $1
  AND orphaned_at IS NULL
  AND NOT (source_external_id = ANY($3::text[]));

-- name: CreateJournal :exec
INSERT INTO journal (id, created_at) VALUES ($1, $2);

-- name: ResetMockJournal :exec
DELETE FROM journal WHERE id = $1;

-- name: GetJourney :one
SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, ST_AsBinary(gps_route) AS gps_route_wkb, authored_fields, created_at, updated_at
FROM journeys
WHERE id = $1;

-- name: GetJourneyBySlug :one
SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, ST_AsBinary(gps_route) AS gps_route_wkb, authored_fields, created_at, updated_at
FROM journeys
WHERE slug = $1;

-- name: ListJourneys :many
SELECT id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, ST_AsBinary(gps_route) AS gps_route_wkb, authored_fields, created_at, updated_at
FROM journeys
ORDER BY date_start DESC;

-- name: UpsertJourney :exec
INSERT INTO journeys (
    id, journal_id, slug, source_ref, title, place, country, region, date_start, date_end, gps_route, authored_fields, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, ST_GeomFromWKB($11, 4326), $12, NOW(), NOW()
) ON CONFLICT (id) DO UPDATE SET
    slug = CASE WHEN NOT (journeys.authored_fields @> ARRAY['slug']) THEN EXCLUDED.slug ELSE journeys.slug END,
    title = CASE WHEN NOT (journeys.authored_fields @> ARRAY['title']) THEN EXCLUDED.title ELSE journeys.title END,
    place = CASE WHEN NOT (journeys.authored_fields @> ARRAY['place']) THEN EXCLUDED.place ELSE journeys.place END,
    country = CASE WHEN NOT (journeys.authored_fields @> ARRAY['country']) THEN EXCLUDED.country ELSE journeys.country END,
    region = CASE WHEN NOT (journeys.authored_fields @> ARRAY['region']) THEN EXCLUDED.region ELSE journeys.region END,
    date_start = CASE WHEN NOT (journeys.authored_fields @> ARRAY['date_start']) THEN EXCLUDED.date_start ELSE journeys.date_start END,
    date_end = CASE WHEN NOT (journeys.authored_fields @> ARRAY['date_end']) THEN EXCLUDED.date_end ELSE journeys.date_end END,
    gps_route = CASE WHEN NOT (journeys.authored_fields @> ARRAY['gps_route']) THEN EXCLUDED.gps_route ELSE journeys.gps_route END,
    source_ref = EXCLUDED.source_ref,
    authored_fields = EXCLUDED.authored_fields,
    updated_at = NOW();

-- name: GetMemento :one
SELECT id, journey_id, kind, seq, occurred_at, occurred_tz, ST_AsBinary(geom) AS geom_wkb, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at
FROM mementos
WHERE id = $1;

-- name: GetMementoBySourceIdentity :one
SELECT id, journey_id, kind, seq, occurred_at, occurred_tz, ST_AsBinary(geom) AS geom_wkb, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at
FROM mementos
WHERE source_system = $1 AND source_external_id = $2;

-- name: ListMementosByJourney :many
SELECT id, journey_id, kind, seq, occurred_at, occurred_tz, ST_AsBinary(geom) AS geom_wkb, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at
FROM mementos
WHERE journey_id = $1
ORDER BY seq ASC, occurred_at ASC;

-- name: UpsertMemento :exec
INSERT INTO mementos (
    id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_system, source_external_id, source_ref, authored_fields, orphaned_at, state, revision, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, ST_GeomFromWKB($7, 4326), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, COALESCE($21, 1), NOW(), NOW()
) ON CONFLICT (id) DO UPDATE SET
    kind = CASE WHEN NOT (mementos.authored_fields @> ARRAY['kind']) THEN EXCLUDED.kind ELSE mementos.kind END,
    seq = CASE WHEN NOT (mementos.authored_fields @> ARRAY['seq']) THEN EXCLUDED.seq ELSE mementos.seq END,
    occurred_at = CASE WHEN NOT (mementos.authored_fields @> ARRAY['occurred_at']) THEN EXCLUDED.occurred_at ELSE mementos.occurred_at END,
    occurred_tz = CASE WHEN NOT (mementos.authored_fields @> ARRAY['occurred_tz']) THEN EXCLUDED.occurred_tz ELSE mementos.occurred_tz END,
    geom = CASE WHEN NOT (mementos.authored_fields @> ARRAY['geom']) THEN EXCLUDED.geom ELSE mementos.geom END,
    title = CASE WHEN NOT (mementos.authored_fields @> ARRAY['title']) THEN EXCLUDED.title ELSE mementos.title END,
    place = CASE WHEN NOT (mementos.authored_fields @> ARRAY['place']) THEN EXCLUDED.place ELSE mementos.place END,
    vendor = CASE WHEN NOT (mementos.authored_fields @> ARRAY['vendor']) THEN EXCLUDED.vendor ELSE mementos.vendor END,
    essay = CASE WHEN NOT (mementos.authored_fields @> ARRAY['essay']) THEN EXCLUDED.essay ELSE mementos.essay END,
    price_amount = CASE WHEN NOT (mementos.authored_fields @> ARRAY['price_amount']) THEN EXCLUDED.price_amount ELSE mementos.price_amount END,
    price_currency = CASE WHEN NOT (mementos.authored_fields @> ARRAY['price_currency']) THEN EXCLUDED.price_currency ELSE mementos.price_currency END,
    kind_data = CASE WHEN NOT (mementos.authored_fields @> ARRAY['kind_data']) THEN EXCLUDED.kind_data ELSE mementos.kind_data END,
    source_system = EXCLUDED.source_system,
    source_external_id = EXCLUDED.source_external_id,
    source_ref = EXCLUDED.source_ref,
    orphaned_at = EXCLUDED.orphaned_at,
    state = EXCLUDED.state,
    revision = mementos.revision + 1,
    authored_fields = EXCLUDED.authored_fields,
    updated_at = NOW()
WHERE $22::bigint IS NULL OR mementos.revision = $22;

-- name: GetPhoto :one
SELECT id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at
FROM memento_photos
WHERE id = $1;

-- name: ListPhotosByMemento :many
SELECT id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at
FROM memento_photos
WHERE memento_id = $1
ORDER BY seq ASC;

-- name: UpsertPhoto :exec
INSERT INTO memento_photos (
    id, memento_id, object_key, content_hash, caption, seq, taken_at, source_ref, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
) ON CONFLICT (id) DO UPDATE SET
    memento_id = EXCLUDED.memento_id,
    object_key = EXCLUDED.object_key,
    content_hash = EXCLUDED.content_hash,
    caption = EXCLUDED.caption,
    seq = EXCLUDED.seq,
    taken_at = EXCLUDED.taken_at,
    source_ref = EXCLUDED.source_ref;

-- name: CreateTransitLeg :exec
-- Build the great-circle arc once, at insert: ST_MakeLine of the two endpoints,
-- densified along the geodesic by ST_Segmentize on geography ($10 = max segment
-- length in metres), cast back to a 4326 LineString for the column (ADR 0009).
INSERT INTO transit_legs (
    id, journey_id, seq, origin_label, dest_label, geom
) VALUES (
    @id, @journey_id, @seq, @origin_label, @dest_label,
    ST_Segmentize(
        ST_MakeLine(
            ST_SetSRID(ST_MakePoint(@origin_lng::float8, @origin_lat::float8), 4326),
            ST_SetSRID(ST_MakePoint(@dest_lng::float8, @dest_lat::float8), 4326)
        )::geography,
        @segment_length_m::float8
    )::geometry
);

-- name: ListTransitLegsByJourney :many
SELECT id, journey_id, seq, origin_label, dest_label, ST_AsBinary(geom) AS geom_wkb, created_at
FROM transit_legs
WHERE journey_id = $1
ORDER BY seq ASC, created_at ASC;

-- name: DeleteTransitLeg :exec
DELETE FROM transit_legs WHERE id = $1;

-- name: GetDisplayRoute :one
-- Union-at-read: the display route is the Dawarich track combined with every
-- authored transit leg, composed on the fly (data-model.md D2). gps_route itself
-- is never mutated. Returns NULL WKB when the journey has neither track nor legs.
SELECT ST_AsBinary(
    ST_Multi(ST_CollectionExtract(
        ST_Collect(
            j.gps_route,
            (SELECT ST_Collect(l.geom) FROM transit_legs l WHERE l.journey_id = j.id)
        ), 2))
) AS route_wkb
FROM journeys j
WHERE j.id = $1;

-- name: SnapToRoute :one
-- Proximity snap of an arbitrary point onto the composed route (track ∪ legs).
-- ST_ClosestPoint is MultiLineString-safe (unlike ST_LineLocatePoint, which
-- rejects multilinestrings). Returns NULL WKB when the route is empty.
SELECT ST_AsBinary(
    ST_ClosestPoint(
        ST_CollectionExtract(
            ST_Collect(
                j.gps_route,
                (SELECT ST_Collect(l.geom) FROM transit_legs l WHERE l.journey_id = j.id)
            ), 2),
        ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)
    )
) AS snapped_wkb
FROM journeys j
WHERE j.id = @journey_id;
