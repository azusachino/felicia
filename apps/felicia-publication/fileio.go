package publication

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// ManifestPath is the artifact-owned file inventory. Finalize uses the
// previous compile's manifest to delete artifacts that this compile did not
// produce, so unpublished or deleted content cannot linger in a reused
// output directory (which the SPA build may share — only manifest-listed
// files are ever removed).
const ManifestPath = "api/v1/manifest.json"

type artifactManifest struct {
	Files []string `json:"files"`
}

// SafeJoin resolves a slash-separated relative path under root, rejecting
// absolute paths, backslashes, and traversal outside the root.
func SafeJoin(root, relative string) (string, error) {
	if relative == "" || strings.Contains(relative, "\\") || strings.HasPrefix(relative, "/") {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	clean := path.Clean(relative)
	if clean == "." || clean != relative || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("unsafe relative path %q", relative)
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

// FileMediaSource opens source media objects from a local directory root.
type FileMediaSource struct{ Root string }

// Open opens the object key under the source root.
func (source FileMediaSource) Open(_ context.Context, objectKey string) (io.ReadCloser, error) {
	filename, err := SafeJoin(source.Root, objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(filename)
}

// FileArtifactWriter writes compiled public artifacts under Root. Every
// written path is recorded so Finalize can reconcile the output directory
// against the previous compile's manifest.
type FileArtifactWriter struct {
	Root    string
	written []string
}

// WriteJSON writes a JSON artifact atomically (temp file + rename).
func (writer *FileArtifactWriter) WriteJSON(filename string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := writer.writeAtomic(filename, append(data, '\n')); err != nil {
		return err
	}
	writer.written = append(writer.written, filename)
	return nil
}

// WriteMedia writes a media artifact atomically (temp file + rename).
func (writer *FileArtifactWriter) WriteMedia(filename string, source io.Reader) error {
	data, err := io.ReadAll(source)
	if err != nil {
		return err
	}
	if err := writer.writeAtomic(filename, data); err != nil {
		return err
	}
	writer.written = append(writer.written, filename)
	return nil
}

func (writer *FileArtifactWriter) writeAtomic(filename string, data []byte) error {
	destination, err := SafeJoin(writer.Root, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(destination), ".felicia-*")
	if err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		_ = os.Remove(temp.Name())
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(temp.Name())
		return err
	}
	if err := os.Chmod(temp.Name(), 0o644); err != nil {
		_ = os.Remove(temp.Name())
		return err
	}
	return os.Rename(temp.Name(), destination)
}

// Finalize writes the artifact manifest and removes every file the previous
// manifest listed that this compile did not produce, pruning directories
// that become empty. It returns the removed artifact paths. Files never
// listed in a manifest (for example a co-located SPA build) are untouched;
// a missing or unreadable previous manifest skips cleanup rather than
// guessing.
func (writer *FileArtifactWriter) Finalize() ([]string, error) {
	keep := make(map[string]bool, len(writer.written)+1)
	for _, name := range writer.written {
		keep[name] = true
	}
	keep[ManifestPath] = true

	previous := writer.readPreviousManifest()
	var removed []string
	for _, name := range previous {
		if keep[name] {
			continue
		}
		destination, err := SafeJoin(writer.Root, name)
		if err != nil {
			// A tampered manifest entry must never trigger a delete
			// outside the artifact root.
			continue
		}
		if err := os.Remove(destination); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return removed, fmt.Errorf("remove stale artifact %s: %w", name, err)
		}
		removed = append(removed, name)
		writer.pruneEmptyParents(destination)
	}
	sort.Strings(removed)

	inventory := append([]string(nil), writer.written...)
	sort.Strings(inventory)
	if err := writer.WriteJSON(ManifestPath, artifactManifest{Files: inventory}); err != nil {
		return removed, fmt.Errorf("write manifest: %w", err)
	}
	return removed, nil
}

func (writer *FileArtifactWriter) readPreviousManifest() []string {
	location, err := SafeJoin(writer.Root, ManifestPath)
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(location)
	if err != nil {
		return nil
	}
	var manifest artifactManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil
	}
	return manifest.Files
}

func (writer *FileArtifactWriter) pruneEmptyParents(removedFile string) {
	root := filepath.Clean(writer.Root)
	for dir := filepath.Dir(removedFile); dir != root && strings.HasPrefix(dir, root+string(filepath.Separator)); dir = filepath.Dir(dir) {
		if err := os.Remove(dir); err != nil {
			return // not empty (or gone) — stop pruning
		}
	}
}
