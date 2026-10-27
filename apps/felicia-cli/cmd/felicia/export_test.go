package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportCmd_UsageError(t *testing.T) {
	var buf bytes.Buffer
	err := execute([]string{"export"}, &buf)
	if err == nil {
		t.Fatal("expected error without arguments, got nil")
	}
}

func TestExportCmd_IzuFixture(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "izu.geojson")

	var buf bytes.Buffer
	err := execute([]string{"export", "--journey", "izu-trip-2026-08-01", "--defer-assets", "--out", outPath}, &buf)
	if err != nil {
		t.Fatalf("export failed: %v\nOutput: %s", err, buf.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read exported geojson: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("parse exported geojson: %v", err)
	}

	if root["type"] != "FeatureCollection" {
		t.Errorf("type = %v, want FeatureCollection", root["type"])
	}
	if root["schema"] != "felicia.journey.geojson.v1" {
		t.Errorf("schema = %v, want felicia.journey.geojson.v1", root["schema"])
	}

	props, ok := root["properties"].(map[string]any)
	if !ok || props["slug"] != "izu-trip-2026-08-01" {
		t.Errorf("properties.slug = %v, want izu-trip-2026-08-01", props["slug"])
	}

	features, ok := root["features"].([]any)
	if !ok || len(features) == 0 {
		t.Fatalf("expected features array, got %+v", root["features"])
	}

	// Verify stdout JSON report
	var report map[string]any
	if err := json.Unmarshal(buf.Bytes(), &report); err != nil {
		t.Fatalf("parse stdout report: %v\nOutput: %s", err, buf.String())
	}
	if report["slug"] != "izu-trip-2026-08-01" {
		t.Errorf("report.slug = %v, want izu-trip-2026-08-01", report["slug"])
	}
}

func TestExportCmd_Stdout(t *testing.T) {
	var buf bytes.Buffer
	err := execute([]string{"export", "--journey", "izu-trip-2026-08-01", "--defer-assets", "--out", "-"}, &buf)
	if err != nil {
		t.Fatalf("export to stdout failed: %v", err)
	}

	var root map[string]any
	if err := json.Unmarshal(buf.Bytes(), &root); err != nil {
		t.Fatalf("parse exported geojson from stdout: %v", err)
	}
	if root["type"] != "FeatureCollection" {
		t.Errorf("type = %v, want FeatureCollection", root["type"])
	}
}
