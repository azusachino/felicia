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

CREATE TABLE IF NOT EXISTS tb_stop_candidates (
  id TEXT PRIMARY KEY,
  journey_id TEXT NOT NULL REFERENCES tb_journeys(id) ON DELETE CASCADE,
  derivation_version TEXT NOT NULL,
  candidate_key TEXT NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  authored_fields TEXT NOT NULL DEFAULT '[]',
  geom TEXT NOT NULL,
  arrive TEXT NOT NULL,
  depart TEXT NOT NULL,
  confidence REAL NOT NULL,
  state TEXT NOT NULL DEFAULT 'proposed' CHECK (state IN ('proposed', 'kept', 'ignored', 'merged')),
  merged_into TEXT REFERENCES tb_stop_candidates(id),
  provenance TEXT NOT NULL DEFAULT '[]',
  revision INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (journey_id, derivation_version, candidate_key)
);
CREATE TABLE IF NOT EXISTS tb_stop_candidate_evidence (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  candidate_id TEXT NOT NULL REFERENCES tb_stop_candidates(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  source_system TEXT NOT NULL,
  source_external_id TEXT NOT NULL,
  locator TEXT NOT NULL,
  UNIQUE (candidate_id, kind, source_system, source_external_id, locator)
);
CREATE INDEX IF NOT EXISTS stop_candidates_journey_idx
  ON tb_stop_candidates(journey_id, arrive);
CREATE INDEX IF NOT EXISTS stop_candidate_evidence_candidate_idx
  ON tb_stop_candidate_evidence(candidate_id);

CREATE TABLE IF NOT EXISTS tb_site_settings (
  journal_id TEXT PRIMARY KEY REFERENCES tb_journals(id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  design TEXT NOT NULL DEFAULT 'atlas' CHECK (design IN ('cartography', 'cabinet', 'techo', 'atlas')),
  default_language TEXT NOT NULL DEFAULT 'ja' CHECK (default_language IN ('ja', 'en', 'zh')),
  default_theme TEXT NOT NULL DEFAULT 'dark' CHECK (default_theme IN ('dark', 'light')),
  accent TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
