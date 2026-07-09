-- name: GetJournal :one
SELECT id, created_at FROM journal WHERE id = $1;

-- name: CreateJournal :exec
INSERT INTO journal (id, created_at) VALUES ($1, $2);

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
SELECT id, journey_id, kind, seq, occurred_at, occurred_tz, ST_AsBinary(geom) AS geom_wkb, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields, orphaned_at, created_at, updated_at
FROM mementos
WHERE id = $1;

-- name: ListMementosByJourney :many
SELECT id, journey_id, kind, seq, occurred_at, occurred_tz, ST_AsBinary(geom) AS geom_wkb, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields, orphaned_at, created_at, updated_at
FROM mementos
WHERE journey_id = $1
ORDER BY seq ASC, occurred_at ASC;

-- name: UpsertMemento :exec
INSERT INTO mementos (
    id, journey_id, kind, seq, occurred_at, occurred_tz, geom, title, place, vendor, essay, price_amount, price_currency, kind_data, source_ref, authored_fields, orphaned_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, ST_GeomFromWKB($7, 4326), $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, NOW(), NOW()
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
    source_ref = EXCLUDED.source_ref,
    orphaned_at = EXCLUDED.orphaned_at,
    authored_fields = EXCLUDED.authored_fields,
    updated_at = NOW();

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

-- name: ListTranslations :many
SELECT id, owner_type, owner_id, lang, field, value, provenance, updated_at
FROM translations
WHERE owner_type = $1 AND owner_id = $2;

-- name: UpsertTranslation :exec
INSERT INTO translations (
    id, owner_type, owner_id, lang, field, value, provenance, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW()
) ON CONFLICT (owner_type, owner_id, lang, field) DO UPDATE SET
    value = CASE WHEN translations.provenance = 'machine' OR EXCLUDED.provenance = 'authored' THEN EXCLUDED.value ELSE translations.value END,
    provenance = CASE WHEN translations.provenance = 'machine' OR EXCLUDED.provenance = 'authored' THEN EXCLUDED.provenance ELSE translations.provenance END,
    updated_at = NOW();
