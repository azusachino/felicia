package domain

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Registry is the set of registered kind templates — loaded from data, so a new
// kind is a new file, not new code (backend-stack.md D9). The legal set of
// mementos.kind values *is* the registry (D8 soft enum).
type Registry struct {
	templates map[string]Template
}

// Template returns the template for a kind and whether it is registered.
// Validating against an unregistered kind is itself an error, surfaced here.
func (r *Registry) Template(kind string) (Template, bool) {
	t, ok := r.templates[kind]
	return t, ok
}

// Kinds returns the registered kinds (unordered).
func (r *Registry) Kinds() []string {
	ks := make([]string, 0, len(r.templates))
	for k := range r.templates {
		ks = append(ks, k)
	}
	return ks
}

// LoadRegistry parses every *.yaml template under fsys. Parsing is the only I/O
// seam; Validate stays pure over the parsed Template. Duplicate kinds are an
// error, as is a file whose declared kind is empty.
func LoadRegistry(fsys fs.FS) (*Registry, error) {
	reg := &Registry{templates: map[string]Template{}}
	entries, err := fs.Glob(fsys, "*.yaml")
	if err != nil {
		return nil, fmt.Errorf("globbing templates: %w", err)
	}
	for _, name := range entries {
		b, err := fs.ReadFile(fsys, name)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", name, err)
		}
		var tpl Template
		if err := yaml.Unmarshal(b, &tpl); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		if tpl.Kind == "" {
			return nil, fmt.Errorf("%s: template has no kind", filepath.Base(name))
		}
		if _, dup := reg.templates[tpl.Kind]; dup {
			return nil, fmt.Errorf("duplicate template kind %q (in %s)", tpl.Kind, name)
		}
		reg.templates[tpl.Kind] = tpl
	}
	return reg, nil
}
