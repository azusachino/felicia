module github.com/azusachino/felicia/apps/apiserver

go 1.26

require (
	github.com/azusachino/felicia/apps/core v0.1.0
	github.com/azusachino/felicia/apps/providers v0.1.0
	github.com/azusachino/felicia/apps/runtime v0.1.0
)

replace (
	github.com/azusachino/felicia/apps/core => ../core
	github.com/azusachino/felicia/apps/providers => ../providers
	github.com/azusachino/felicia/apps/runtime => ../runtime
)
