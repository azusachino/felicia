package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesTOMLThenEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), "felicia.toml")
	if err := os.WriteFile(path, []byte(`
[database]
dsn = "postgres://from-file"

[server]
port = "9090"

[dawarich]
url = "https://dawarich.example"

[ingest]
rdp_epsilon = 0.002
transit_segment_length_m = 200000
`), 0o600); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{
		"FELICIA_PORT":       "9191",
		"DAWARICH_API_KEY":   "legacy-key",
		"FELICIA_IMMICH_URL": "https://immich.example",
		"FELICIA_CACHE_ADDR": "",
	}
	cfg, err := Load(path, lookup(values))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "9191" || cfg.DatabaseDSN != "postgres://from-file" || cfg.CacheAddr != "" {
		t.Errorf("unexpected base config: %+v", cfg)
	}
	if cfg.Dawarich.URL != "https://dawarich.example" || cfg.Dawarich.APIKey != "legacy-key" || cfg.Immich.URL != "https://immich.example" {
		t.Errorf("unexpected source config: %+v", cfg)
	}
	if cfg.RDPEpsilon != 0.002 || cfg.TransitSegmentLenM != 200000 {
		t.Errorf("unexpected ingest config: %+v", cfg)
	}
}

func TestLoadUsesDefaultsAndRequiresDatabaseDSN(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{"DATABASE_DSN": "postgres://dev"}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != defaultPort || cfg.CacheAddr != defaultCacheAddr || cfg.RDPEpsilon != defaultRDPEpsilon || cfg.TransitSegmentLenM != defaultTransitSegmentLenM {
		t.Errorf("unexpected defaults: %+v", cfg)
	}

	if _, err := Load("", lookup(nil)); err == nil {
		t.Error("expected missing DATABASE_DSN error")
	}
}

func TestLoadPrefersFeliciaEnvironment(t *testing.T) {
	cfg, err := Load("", lookup(map[string]string{
		"DATABASE_DSN":         "postgres://legacy",
		"FELICIA_DATABASE_DSN": "postgres://preferred",
		"DAWARICH_URL":         "https://legacy.example",
		"FELICIA_DAWARICH_URL": "https://preferred.example",
	}))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DatabaseDSN != "postgres://preferred" || cfg.Dawarich.URL != "https://preferred.example" {
		t.Errorf("FELICIA_* overrides were not preferred: %+v", cfg)
	}
}

func lookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
