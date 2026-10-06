module github.com/azusachino/felicia/apps/felicia-runtime

go 1.27

require (
	github.com/azusachino/felicia/apps/felicia-core v0.1.0
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
	github.com/ringsaturn/tzf/v2 v2.1.2
	gopkg.in/yaml.v3 v3.0.1
	github.com/azusachino/felicia/apps/felicia-providers v0.1.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	github.com/ringsaturn/tzf-dist v0.0.2026-d // indirect
)

replace github.com/azusachino/felicia/apps/felicia-core => ../felicia-core

replace github.com/azusachino/felicia/apps/felicia-providers => ../felicia-providers
