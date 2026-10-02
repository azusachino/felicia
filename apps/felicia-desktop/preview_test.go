package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestPreviewServerOverlay(t *testing.T) {
	outDir := t.TempDir()

	// 1. Create a compiled artifact file in outDir
	apiDir := filepath.Join(outDir, "api", "v1")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(apiDir, "site.json"), []byte(`{"title":"Artifact Site"}`), 0o644); err != nil {
		t.Fatalf("write site.json: %v", err)
	}

	// 2. Create mock reader filesystem with index.html
	readerFS := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(`<!doctype html><html><body>Reader SPA</body></html>`)},
	}

	ps, err := StartPreviewServer("127.0.0.1:0", func() string { return outDir }, readerFS)
	if err != nil {
		t.Fatalf("start preview server: %v", err)
	}
	defer func() { _ = ps.Close() }()

	baseURL := "http://127.0.0.1:" + ps.Port()

	// Test 1: Root returns index.html from readerFS
	res, err := http.Get(baseURL + "/")
	if err != nil {
		t.Fatalf("get root: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(body), "Reader SPA") {
		t.Fatalf("expected Reader SPA at root, got %s", string(body))
	}

	// Test 2: /api/v1/site.json returns compiled artifact from outDir
	res, err = http.Get(baseURL + "/api/v1/site.json")
	if err != nil {
		t.Fatalf("get site.json: %v", err)
	}
	body, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(body), "Artifact Site") {
		t.Fatalf("expected Artifact Site, got %s", string(body))
	}

	// Test 3: Method not allowed on POST
	res, err = http.Post(baseURL+"/api/v1/site.json", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("post site.json: %v", err)
	}
	_ = res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 Method Not Allowed, got %d", res.StatusCode)
	}
}
