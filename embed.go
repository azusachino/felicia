// Package felicia is the root package hosting embedded templates.
package felicia

import "embed"

// KindsFS contains the embedded declarative kind template YAML files.
//
//go:embed kinds/*.yaml
var KindsFS embed.FS
