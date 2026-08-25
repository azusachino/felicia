module github.com/azusachino/felicia/apps/felicia-publication

go 1.27

require (
	github.com/azusachino/felicia/apps/felicia-core v0.1.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
	golang.org/x/image v0.45.0
)

require (
	github.com/kr/text v0.2.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/azusachino/felicia/apps/felicia-core => ../felicia-core
