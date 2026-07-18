// Package config loads the API's local configuration from TOML and environment
// variables. Environment values take precedence so secrets stay out of files.
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

const (
	defaultPort               = "8080"
	defaultDatabaseDriver     = "sqlite"
	defaultDatabasePath       = "felicia.db"
	defaultCacheAddr          = "localhost:6379"
	defaultRDPEpsilon         = 0.0001
	defaultTransitSegmentLenM = 100000
	defaultConfigPath         = "felicia.toml"
	defaultMediaRoot          = ".felicia/media"
)

// SourceConfig describes one authenticated upstream source.
type SourceConfig struct {
	URL    string `koanf:"url"`
	APIKey string `koanf:"api_key"`
}

// Config is the complete runtime configuration for the local API server.
type Config struct {
	DatabaseDriver     string       `koanf:"database.driver"`
	DatabaseDSN        string       `koanf:"database.dsn"`
	DatabasePath       string       `koanf:"database.path"`
	Port               string       `koanf:"server.port"`
	CacheAddr          string       `koanf:"cache.addr"`
	Dawarich           SourceConfig `koanf:"dawarich"`
	Immich             SourceConfig `koanf:"immich"`
	RDPEpsilon         float64      `koanf:"ingest.rdp_epsilon"`
	TransitSegmentLenM float64      `koanf:"ingest.transit_segment_length_m"`
	// MediaRoot is the private local media root read when compiling the public
	// static artifact (server/api's /api/admin/compile), the same root the CLI
	// resolves via its --media-root flag.
	MediaRoot string `koanf:"media.root"`
}

// LoadFromEnv loads felicia.toml, or the path selected by FELICIA_CONFIG, then
// applies environment overrides.
func LoadFromEnv() (Config, error) {
	path := defaultConfigPath
	required := false
	if configured, ok := os.LookupEnv("FELICIA_CONFIG"); ok {
		path = configured
		required = configured != ""
	}
	return load(path, os.LookupEnv, required)
}

// Load reads an optional TOML file and applies environment overrides from
// lookup. FELICIA_* variables take precedence over their legacy unprefixed
// equivalents so existing local scripts continue to work.
func Load(path string, lookup func(string) (string, bool)) (Config, error) {
	return load(path, lookup, false)
}

func load(path string, lookup func(string) (string, bool), required bool) (Config, error) {
	k := koanf.New(".")
	if err := k.Load(confmap.Provider(defaults(), "."), nil); err != nil {
		return Config{}, fmt.Errorf("load defaults: %w", err)
	}
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
				return Config{}, fmt.Errorf("load config %s: %w", path, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("stat config %s: %w", path, err)
		} else if required {
			return Config{}, fmt.Errorf("config %s: %w", path, err)
		}
	}
	if err := k.Load(confmap.Provider(environmentOverrides(lookup), "."), nil); err != nil {
		return Config{}, fmt.Errorf("load environment: %w", err)
	}

	cfg := Config{
		DatabaseDriver:     k.String("database.driver"),
		DatabaseDSN:        k.String("database.dsn"),
		DatabasePath:       k.String("database.path"),
		Port:               k.String("server.port"),
		CacheAddr:          k.String("cache.addr"),
		Dawarich:           SourceConfig{URL: k.String("dawarich.url"), APIKey: k.String("dawarich.api_key")},
		Immich:             SourceConfig{URL: k.String("immich.url"), APIKey: k.String("immich.api_key")},
		RDPEpsilon:         k.Float64("ingest.rdp_epsilon"),
		TransitSegmentLenM: k.Float64("ingest.transit_segment_length_m"),
		MediaRoot:          k.String("media.root"),
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks the values that would make the API unsafe or inoperable.
func (c Config) Validate() error {
	if c.DatabaseDriver != "sqlite" && c.DatabaseDriver != "postgres" {
		return fmt.Errorf("database driver must be sqlite or postgres, got %q", c.DatabaseDriver)
	}
	if c.DatabaseDriver == "sqlite" && c.DatabasePath == "" {
		return errors.New("database path is required for sqlite")
	}
	if c.DatabaseDriver == "postgres" && c.DatabaseDSN == "" {
		return errors.New("DATABASE_DSN environment variable is required for postgres")
	}
	if c.Port == "" {
		return errors.New("server port is required")
	}
	if c.RDPEpsilon <= 0 {
		return errors.New("ingest rdp_epsilon must be positive")
	}
	if c.TransitSegmentLenM <= 0 {
		return errors.New("ingest transit_segment_length_m must be positive")
	}
	if c.MediaRoot == "" {
		return errors.New("media root is required")
	}
	return nil
}

func defaults() map[string]any {
	return map[string]any{
		"database": map[string]any{"driver": defaultDatabaseDriver, "path": defaultDatabasePath},
		"server":   map[string]any{"port": defaultPort},
		"cache":    map[string]any{"addr": defaultCacheAddr},
		"ingest": map[string]any{
			"rdp_epsilon":              defaultRDPEpsilon,
			"transit_segment_length_m": defaultTransitSegmentLenM,
		},
		"media": map[string]any{"root": defaultMediaRoot},
	}
}

func environmentOverrides(lookup func(string) (string, bool)) map[string]any {
	values := make(map[string]any)
	set := func(key string, names ...string) {
		for _, name := range names {
			if value, ok := lookup(name); ok {
				values[key] = value
				return
			}
		}
	}
	set("database.dsn", "FELICIA_DATABASE_DSN", "DATABASE_DSN")
	set("database.driver", "FELICIA_DATABASE_DRIVER", "DATABASE_DRIVER")
	set("database.path", "FELICIA_DATABASE_PATH", "DATABASE_PATH")
	set("server.port", "FELICIA_PORT", "PORT")
	set("cache.addr", "FELICIA_CACHE_ADDR", "CACHE_ADDR")
	set("dawarich.url", "FELICIA_DAWARICH_URL", "DAWARICH_URL")
	set("dawarich.api_key", "FELICIA_DAWARICH_API_KEY", "DAWARICH_API_KEY")
	set("immich.url", "FELICIA_IMMICH_URL", "IMMICH_URL")
	set("immich.api_key", "FELICIA_IMMICH_API_KEY", "IMMICH_API_KEY")
	set("ingest.rdp_epsilon", "FELICIA_RDP_EPSILON", "RDP_EPSILON")
	set("ingest.transit_segment_length_m", "FELICIA_TRANSIT_SEGMENT_LENGTH_M", "TRANSIT_SEGMENT_LENGTH_M")
	set("media.root", "FELICIA_MEDIA_ROOT", "MEDIA_ROOT")
	return values
}
