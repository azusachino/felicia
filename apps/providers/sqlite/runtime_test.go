package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/azusachino/felicia/apps/core/domain"
)

func TestOpenConfiguresFileDatabaseForLocalRuntime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "felicia.db")
	repo, err := Open(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if got := pragmaString(t, repo.db, "journal_mode"); got != "wal" {
		t.Fatalf("journal_mode = %q, want wal", got)
	}
	if got := pragmaInt(t, repo.db, "foreign_keys"); got != 1 {
		t.Fatalf("foreign_keys = %d, want 1", got)
	}
	if got := pragmaInt(t, repo.db, "busy_timeout"); got != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", got)
	}
	if err := repo.Close(); err != nil {
		t.Fatalf("close sqlite: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen sqlite: %v", err)
	}
	defer reopened.Close()

	journal := &domain.Journal{ID: mustRuntimeUUID(t), CreatedAt: time.Now().UTC()}
	if err := reopened.CreateJournal(context.Background(), journal); err != nil {
		t.Fatalf("create journal after reopen: %v", err)
	}
	if _, err := reopened.GetJournal(context.Background(), journal.ID); err != nil {
		t.Fatalf("get journal after reopen: %v", err)
	}
}

func TestOpenConfiguresInMemoryDatabaseWithoutWAL(t *testing.T) {
	repo, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer repo.Close()
	if got := pragmaString(t, repo.db, "journal_mode"); got != "memory" {
		t.Fatalf("journal_mode = %q, want memory", got)
	}
	if got := pragmaInt(t, repo.db, "foreign_keys"); got != 1 {
		t.Fatalf("foreign_keys = %d, want 1", got)
	}
}

func pragmaString(t *testing.T, db *sql.DB, name string) string {
	t.Helper()
	var value string
	if err := db.QueryRow("PRAGMA " + name).Scan(&value); err != nil {
		t.Fatalf("read pragma %s: %v", name, err)
	}
	return value
}

func pragmaInt(t *testing.T, db *sql.DB, name string) int {
	t.Helper()
	var value int
	if err := db.QueryRow("PRAGMA " + name).Scan(&value); err != nil {
		t.Fatalf("read pragma %s: %v", name, err)
	}
	return value
}

func mustRuntimeUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
