package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/azusachino/felicia/apps/felicia-runtime/intake"
	"github.com/azusachino/felicia/apps/felicia-runtime/workspace"
)

func parseOptionalTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func sourceFingerprint(filenames ...string) (string, error) {
	hash := sha256.New()
	for _, filename := range filenames {
		if filename == "" {
			continue
		}
		file, err := os.Open(filename)
		if err != nil {
			return "", fmt.Errorf("open source %s: %w", filename, err)
		}
		if _, err := io.WriteString(hash, filename+"\x00"); err != nil {
			_ = file.Close()
			return "", err
		}
		if _, err := io.Copy(hash, file); err != nil {
			_ = file.Close()
			return "", err
		}
		if err := file.Close(); err != nil {
			return "", err
		}
	}
	return "sha256:" + fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func writePlanJSONL(output io.Writer, plan intake.DraftPlan) error {
	encoder := json.NewEncoder(output)
	for _, stop := range plan.Stops {
		if err := encoder.Encode(map[string]any{"type": "stop", "stop": stop}); err != nil {
			return err
		}
	}
	return encoder.Encode(map[string]any{"type": "summary", "schema": plan.Schema, "version": plan.Version, "source_fingerprint": plan.SourceFingerprint, "stops": len(plan.Stops), "mementos": len(plan.Mementos), "issues": len(plan.Issues)})
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func resolveWorkspaceDefaults(database, mediaRoot *string) error {
	if (database != nil && *database == "") || (mediaRoot != nil && *mediaRoot == "") {
		ws, err := workspace.Resolve("")
		if err != nil {
			return fmt.Errorf("resolve workspace: %w", err)
		}
		if err := ws.EnsureDirs(); err != nil {
			return err
		}
		if database != nil && *database == "" {
			*database = ws.Database
		}
		if mediaRoot != nil && *mediaRoot == "" {
			*mediaRoot = ws.MediaRoot
		}
	}
	return nil
}
