module github.com/azusachino/felicia/apps/felicia-desktop

go 1.27

require (
	github.com/azusachino/felicia/apps/felicia-core v0.1.0
	github.com/azusachino/felicia/apps/felicia-providers v0.1.0
	github.com/azusachino/felicia/apps/felicia-publication v0.1.0
	github.com/azusachino/felicia/apps/felicia-runtime v0.1.0
	github.com/google/uuid v1.6.0
	github.com/paulmach/orb v0.13.0
	github.com/wailsapp/wails/v3 v3.0.0-beta.24
)

replace (
	github.com/azusachino/felicia/apps/felicia-core => ../felicia-core
	github.com/azusachino/felicia/apps/felicia-providers => ../felicia-providers
	github.com/azusachino/felicia/apps/felicia-publication => ../felicia-publication
	github.com/azusachino/felicia/apps/felicia-runtime => ../felicia-runtime
)
