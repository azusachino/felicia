module github.com/azusachino/felicia/apps/felicia-publication

go 1.26

require (
	github.com/azusachino/felicia/apps/felicia-core v0.1.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
	golang.org/x/image v0.44.0
)

require (
	github.com/kr/pretty v0.3.0 // indirect
	github.com/rogpeppe/go-internal v1.15.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/azusachino/felicia/apps/felicia-core => ../felicia-core
