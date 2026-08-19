module github.com/azusachino/felicia/publication

go 1.26

require (
	github.com/azusachino/felicia/core v0.1.0
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
	golang.org/x/image v0.44.0
)

require gopkg.in/yaml.v3 v3.0.1 // indirect

replace github.com/azusachino/felicia/core => ../core
