package publication

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileMediaSource opens media by object key relative to a local root
// directory. It implements MediaSource for any compilation (CLI or server)
// that reads from a filesystem media root.
type FileMediaSource struct{ Root string }

// Open implements MediaSource.
func (source FileMediaSource) Open(_ context.Context, objectKey string) (io.ReadCloser, error) {
	filename, err := SafeJoin(source.Root, objectKey)
	if err != nil {
		return nil, err
	}
	return os.Open(filename)
}

// FileArtifactWriter writes the compiled public JSON tree and referenced
// media files under a local output directory. It implements ArtifactWriter.
type FileArtifactWriter struct{ Root string }

// WriteJSON implements ArtifactWriter.
func (writer *FileArtifactWriter) WriteJSON(filename string, value any) error {
	destination, err := SafeJoin(writer.Root, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0o644)
}

// WriteMedia implements ArtifactWriter.
func (writer *FileArtifactWriter) WriteMedia(filename string, source io.Reader) error {
	destination, err := SafeJoin(writer.Root, filename)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	_, err = io.Copy(file, source)
	return err
}

// SafeJoin joins a relative path onto root, rejecting any relative path that
// could escape root (a leading slash, a backslash, or a ".." segment).
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
