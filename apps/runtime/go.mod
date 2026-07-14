module github.com/azusachino/felicia/apps/runtime

go 1.26

require github.com/azusachino/felicia/apps/core v0.1.0

require (
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/azusachino/felicia/apps/core => ../core
