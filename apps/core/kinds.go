// Package core hosts the canonical domain model and embedded kind templates.
package core

import "embed"

// KindsFS contains the embedded declarative kind template YAML files.
//
//go:embed kinds/*.yaml
var KindsFS embed.FS
