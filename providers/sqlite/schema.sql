PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS tb_journals (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tb_journeys (
  id TEXT PRIMARY KEY,
  journal_id TEXT NOT NULL REFERENCES tb_journals(id) ON DELETE CASCADE,
  slug TEXT NOT NULL UNIQUE,
  source_ref TEXT,
  title TEXT NOT NULL,
  place TEXT NOT NULL,
  country TEXT,
  region TEXT,
  date_start TEXT NOT NULL,
  date_end TEXT NOT NULL,
  gps_route TEXT NOT NULL DEFAULT '[]',
  authored_fields TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tb_mementos (
  id TEXT PRIMARY KEY,
  journey_id TEXT NOT NULL REFERENCES tb_journeys(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  seq INTEGER NOT NULL,
  occurred_at TEXT,
  occurred_tz TEXT,
  geom TEXT,
  title TEXT,
  place TEXT,
  vendor TEXT,
  essay TEXT,
  price_amount INTEGER,
  price_currency TEXT,
  kind_data TEXT NOT NULL DEFAULT '{}',
  source_system TEXT,
  source_external_id TEXT,
  source_ref TEXT,
  authored_fields TEXT NOT NULL DEFAULT '[]',
  orphaned_at TEXT,
  state TEXT NOT NULL DEFAULT 'published',
  revision INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (source_system, source_external_id)
);
CREATE TABLE IF NOT EXISTS tb_memento_photos (
  id TEXT PRIMARY KEY,
  memento_id TEXT NOT NULL REFERENCES tb_mementos(id) ON DELETE CASCADE,
  object_key TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  caption TEXT,
  seq INTEGER NOT NULL DEFAULT 0,
  taken_at TEXT,
  source_ref TEXT,
  created_at TEXT NOT NULL,
  UNIQUE (memento_id, content_hash)
);
CREATE TABLE IF NOT EXISTS tb_transit_legs (
  id TEXT PRIMARY KEY,
  journey_id TEXT NOT NULL REFERENCES tb_journeys(id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  origin_label TEXT,
  dest_label TEXT,
  geom TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tb_import_runs (
  id TEXT PRIMARY KEY,
  source_system TEXT NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  status TEXT NOT NULL,
  error_message TEXT
);
CREATE TABLE IF NOT EXISTS tb_source_observations (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES tb_import_runs(id) ON DELETE CASCADE,
  source_system TEXT NOT NULL,
  source_external_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  confidence REAL NOT NULL,
  payload TEXT NOT NULL,
  changed INTEGER NOT NULL DEFAULT 0,
  orphaned_at TEXT,
  created_at TEXT NOT NULL,
  UNIQUE (run_id, source_system, source_external_id)
);
CREATE INDEX IF NOT EXISTS source_observations_identity_idx
  ON tb_source_observations(source_system, source_external_id, observed_at DESC);
