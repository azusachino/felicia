package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSQLiteConfigFixture(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "sqlite.toml"), lookup(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseDriver != "sqlite" || cfg.DatabasePath != "journal.db" || cfg.Port != "8181" {
		t.Errorf("unexpected SQLite config: %+v", cfg)
	}
}

func TestLoadPostgresConfigFixture(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "postgres.toml"), lookup(nil))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseDriver != "postgres" || cfg.DatabaseDSN != "postgres://from-file" {
		t.Errorf("unexpected PostgreSQL config: %+v", cfg)
	}
}

func TestLoadEnvironmentOverridesConfigFile(t *testing.T) {
	path := filepath.Join("testdata", "sqlite.toml")
	cfg, err := Load(path, lookup(map[string]string{
		"FELICIA_DATABASE_DRIVER": "postgres",
		"FELICIA_DATABASE_DSN":    "postgres://from-env",
		"FELICIA_DATABASE_PATH":   "overridden.db",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseDriver != "postgres" || cfg.DatabaseDSN != "postgres://from-env" || cfg.DatabasePath != "overridden.db" {
		t.Errorf("environment did not override config file: %+v", cfg)
	}
}

func TestLoadRejectsMalformedConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "broken.toml")
	if err := os.WriteFile(path, []byte("[database\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path, lookup(nil)); err == nil {
		t.Fatal("expected malformed config to fail")
	}
}

func TestLoadFromEnvRejectsMissingExplicitConfig(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.toml")
	t.Setenv("FELICIA_CONFIG", missing)
	if _, err := LoadFromEnv(); err == nil {
		t.Fatal("expected missing explicit config to fail")
	}
}
