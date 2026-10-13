package sqlite_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite" // register the SQLite database driver

	"github.com/azusachino/felicia/apps/felicia-core/domain"
	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
)

// previousVersionSchema is a pre-enrichment v1 schema: the enrichment columns
// (leg_modes, weather, video_key) are absent, as they would be in a database
// created by an older build whose schema.sql predated those columns.
const previousVersionSchema = `
CREATE TABLE tb_journals (
  id TEXT PRIMARY KEY,
  created_at TEXT NOT NULL
);
CREATE TABLE tb_journeys (
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
  revision INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE tb_memento_photos (
  id TEXT PRIMARY KEY,
  memento_id TEXT NOT NULL,
  object_key TEXT NOT NULL,
  seq INTEGER NOT NULL DEFAULT 1
);
`

func TestOpenFreshDatabase(t *testing.T) {
	repo, err := sqlite.Open(":memory:")
	if err != nil {
		t.Fatalf("open fresh sqlite: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()
	journal := &domain.Journal{ID: uuid.Must(uuid.NewRandom()), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("use fresh database: %v", err)
	}
}

func tableHasColumn(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("inspect %s: %v", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull int
		var dflt any
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == column {
			return true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table_info: %v", err)
	}
	return false
}

func TestOpenUpgradesPreviousVersionDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "previous.db")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open seed database: %v", err)
	}
	if _, err := seed.Exec(previousVersionSchema); err != nil {
		t.Fatalf("seed previous-version schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed database: %v", err)
	}

	repo, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open previous-version database: %v", err)
	}
	defer repo.Close()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("reopen for inspection: %v", err)
	}
	defer db.Close()
	for _, tc := range []struct{ table, column string }{
		{"tb_journeys", "leg_modes"},
		{"tb_journeys", "weather"},
		{"tb_memento_photos", "video_key"},
	} {
		if !tableHasColumn(t, db, tc.table, tc.column) {
			t.Fatalf("column %s.%s missing after upgrade", tc.table, tc.column)
		}
	}
}

func TestOpenReportsUpgradeFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.db")
	seed, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open seed database: %v", err)
	}
	// A view named tb_journeys without the enrichment columns blocks both the
	// schema.sql CREATE (IF NOT EXISTS skips it) and the upgrade ALTER TABLE,
	// deterministically inducing an upgrade failure.
	if _, err := seed.Exec(`CREATE VIEW tb_journeys AS SELECT '' AS id, '' AS title WHERE 0`); err != nil {
		t.Fatalf("seed blocking view: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed database: %v", err)
	}

	repo, err := sqlite.Open(path)
	if err == nil {
		_ = repo.Close()
		t.Fatal("expected startup error from failed schema upgrade, got nil")
	}
	if !strings.Contains(err.Error(), "upgrade sqlite schema") {
		t.Fatalf("expected upgrade error, got: %v", err)
	}
}

func TestGetJourneyReportsMalformedStoredJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "enrichment.db")
	repo, err := sqlite.Open(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	ctx := context.Background()

	journal := &domain.Journal{ID: uuid.Must(uuid.NewRandom()), CreatedAt: time.Now().UTC()}
	if err := repo.CreateJournal(ctx, journal); err != nil {
		t.Fatalf("create journal: %v", err)
	}
	journey := &domain.Journey{
		ID: uuid.Must(uuid.NewRandom()), JournalID: journal.ID, Slug: "enrichment",
		Title: "Enrichment", Place: "Tokyo",
		DateStart: time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
		DateEnd:   time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
		LegModes:  []domain.LegMode{domain.LegModeTrain},
		Weather:   domain.Weather{"2026-03-20": {Icon: "cloudy"}},
	}
	if err := repo.UpsertJourney(ctx, journey); err != nil {
		t.Fatalf("upsert journey: %v", err)
	}

	corrupt := func(t *testing.T, column, value string) {
		t.Helper()
		db, err := sql.Open("sqlite", path)
		if err != nil {
			t.Fatalf("open raw database: %v", err)
		}
		defer db.Close()
		if _, err := db.ExecContext(ctx, "UPDATE tb_journeys SET "+column+" = ? WHERE id = ?", value, journey.ID.String()); err != nil {
			t.Fatalf("corrupt %s: %v", column, err)
		}
	}

	corrupt(t, "leg_modes", "not-json")
	if _, err := repo.GetJourney(ctx, journey.ID); err == nil {
		t.Fatal("expected error for malformed stored leg_modes, got nil")
	} else if !strings.Contains(err.Error(), "leg_modes") {
		t.Fatalf("expected error naming leg_modes, got: %v", err)
	}
	corrupt(t, "leg_modes", `["train"]`)

	corrupt(t, "weather", "{bad")
	if _, err := repo.GetJourney(ctx, journey.ID); err == nil {
		t.Fatal("expected error for malformed stored weather, got nil")
	} else if !strings.Contains(err.Error(), "weather") {
		t.Fatalf("expected error naming weather, got: %v", err)
	}
}
