package config

import (
	"os"
	"path/filepath"
	"strings"
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

// A PostgreSQL config file used to load successfully. Under ADR-0032 it is a
// startup error: PostgreSQL is deferred to v1.1/v1.2, and a supported
// executable refuses rather than running an unsupported provider.
func TestLoadRejectsPostgresConfigFixture(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "postgres.toml"), lookup(nil))
	if err == nil {
		t.Fatal("a postgres config file must be refused, not loaded")
	}
	if !strings.Contains(err.Error(), "ADR-0032") {
		t.Errorf("the refusal should say why it is refused, got %v", err)
	}
}

// The precedence this asserts is unchanged; only the values are, since the
// provider it used to override to is no longer a supported one.
func TestLoadEnvironmentOverridesConfigFile(t *testing.T) {
	path := filepath.Join("testdata", "sqlite.toml")
	cfg, err := Load(path, lookup(map[string]string{
		"FELICIA_DATABASE_PATH": "overridden.db",
		"FELICIA_PORT":          "9999",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabasePath != "overridden.db" || cfg.Port != "9999" {
		t.Errorf("environment did not override config file: %+v", cfg)
	}
}

// Selecting PostgreSQL through the environment is refused for the same reason
// as through a file: the precedence chain does not create an exception.
func TestLoadRejectsPostgresFromEnvironment(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "sqlite.toml"), lookup(map[string]string{
		"FELICIA_DATABASE_DRIVER": "postgres",
		"FELICIA_DATABASE_DSN":    "postgres://from-env",
	}))
	if err == nil {
		t.Fatal("selecting postgres via the environment must be refused")
	}
}

// A DSN with no driver selected used to start SQLite silently, which is what
// ops/compose.yaml did while PostgreSQL and every migration sat unused.
func TestLoadRejectsADSNForAnUnselectedProvider(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "sqlite.toml"), lookup(map[string]string{
		"DATABASE_DSN": "postgres://nobody-selected-this",
	}))
	if err == nil {
		t.Fatal("a DSN for an unselected provider must be a configuration error, not a default")
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

// TestLoadAcceptsTheComposeShape pins the deployment configuration ops/compose.yaml
// now ships: an explicit sqlite driver, an explicit path, and no DSN. The
// previous shape (a PostgreSQL DSN with no driver) is what this refuses.
func TestLoadAcceptsTheComposeShape(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"DATABASE_DRIVER": "sqlite",
		"DATABASE_PATH":   "/data/felicia.sqlite",
		"CACHE_ADDR":      "cache:6379",
	}))
	if err != nil {
		t.Fatalf("the shipped compose configuration must start: %v", err)
	}
	if cfg.DatabaseDriver != "sqlite" || cfg.DatabasePath != "/data/felicia.sqlite" {
		t.Errorf("unexpected compose config: %+v", cfg)
	}
}
