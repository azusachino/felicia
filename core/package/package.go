// Package journeypackage reads and validates Felicia's portable journey ZIP.
package journeypackage

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// CurrentSchemaVersion is the manifest schema version accepted by this importer.
const CurrentSchemaVersion = "1"

// Manifest describes the files and identities inside one portable package.
type Manifest struct {
	SchemaVersion string      `yaml:"schema_version"`
	PackageID     string      `yaml:"package_id"`
	CreatedAt     string      `yaml:"created_at"`
	Source        string      `yaml:"source,omitempty"`
	Timezone      string      `yaml:"timezone,omitempty"`
	Files         []FileEntry `yaml:"files"`
}

// FileEntry is the integrity record for one package member.
type FileEntry struct {
	Path   string `yaml:"path"`
	Kind   string `yaml:"kind"`
	Bytes  int64  `yaml:"bytes"`
	SHA256 string `yaml:"sha256"`
}

// Package is a validated, in-memory package envelope. Importers consume the
// named members but do not receive arbitrary archive paths.
type Package struct {
	Manifest Manifest
	Files    map[string][]byte
}

// Read opens and validates a package ZIP before returning any importable data.
func Read(filename string) (*Package, error) {
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return nil, fmt.Errorf("open package %s: %w", filename, err)
	}
	defer func() { _ = archive.Close() }()

	entries := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		if err := validateMemberPath(file.Name); err != nil {
			return nil, fmt.Errorf("package member %q: %w", file.Name, err)
		}
		if _, exists := entries[file.Name]; exists {
			return nil, fmt.Errorf("duplicate package member %q", file.Name)
		}
		if file.Mode()&0111 != 0 || file.Mode()&fs.ModeSymlink != 0 {
			return nil, fmt.Errorf("unsafe package member %q", file.Name)
		}
		entries[file.Name] = file
	}

	manifestFile, ok := entries["manifest.yaml"]
	if !ok {
		return nil, errors.New("manifest.yaml is required")
	}
	manifestBytes, err := readMember(manifestFile)
	if err != nil {
		return nil, fmt.Errorf("read manifest.yaml: %w", err)
	}
	var manifest Manifest
	if err := yaml.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest.yaml: %w", err)
	}
	if err := validateManifest(manifest, entries); err != nil {
		return nil, err
	}

	files := make(map[string][]byte, len(manifest.Files))
	for _, expected := range manifest.Files {
		member := entries[expected.Path]
		data, err := readMember(member)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", expected.Path, err)
		}
		files[expected.Path] = data
	}
	return &Package{Manifest: manifest, Files: files}, nil
}

func validateManifest(manifest Manifest, entries map[string]*zip.File) error {
	if manifest.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema_version %q", manifest.SchemaVersion)
	}
	if manifest.PackageID == "" {
		return errors.New("manifest package_id is required")
	}
	if len(manifest.Files) == 0 {
		return errors.New("manifest files cannot be empty")
	}
	seen := make(map[string]bool, len(manifest.Files))
	for _, expected := range manifest.Files {
		if err := validateMemberPath(expected.Path); err != nil {
			return fmt.Errorf("manifest file %q: %w", expected.Path, err)
		}
		if seen[expected.Path] {
			return fmt.Errorf("duplicate manifest file %q", expected.Path)
		}
		seen[expected.Path] = true
		member, ok := entries[expected.Path]
		if !ok {
			return fmt.Errorf("manifest file %q is missing from ZIP", expected.Path)
		}
		if expected.Bytes != int64(member.UncompressedSize64) {
			return fmt.Errorf("size mismatch for %q", expected.Path)
		}
		data, err := readMember(member)
		if err != nil {
			return fmt.Errorf("read %q for checksum: %w", expected.Path, err)
		}
		digest := sha256.Sum256(data)
		if !strings.EqualFold(expected.SHA256, hex.EncodeToString(digest[:])) {
			return fmt.Errorf("checksum mismatch for %q", expected.Path)
		}
	}
	return nil
}

func validateMemberPath(name string) error {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") {
		return errors.New("path must be relative and use slash separators")
	}
	clean := path.Clean(name)
	if clean != name || clean == "." || strings.HasPrefix(clean, "../") {
		return errors.New("path traversal is not allowed")
	}
	return nil
}

func readMember(file *zip.File) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = reader.Close() }()
	return io.ReadAll(reader)
}
