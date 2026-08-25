module github.com/azusachino/felicia/apps/felicia-cli

go 1.27

require (
	github.com/azusachino/felicia/apps/felicia-core v0.1.0
	github.com/azusachino/felicia/apps/felicia-providers v0.1.0
	github.com/azusachino/felicia/apps/felicia-publication v0.1.0
	github.com/azusachino/felicia/apps/felicia-runtime v0.1.0
	github.com/google/uuid v1.6.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/mattn/go-isatty v0.0.16 // indirect
	github.com/paulmach/orb v0.13.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd // indirect
	golang.org/x/image v0.45.0 // indirect
	golang.org/x/mod v0.40.0 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/tools v0.49.0 // indirect
	lukechampine.com/uint128 v1.2.0 // indirect
	modernc.org/cc/v3 v3.40.0 // indirect
	modernc.org/ccgo/v3 v3.16.13 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/opt v0.1.3 // indirect
	modernc.org/sqlite v1.23.1 // indirect
	modernc.org/strutil v1.1.3 // indirect
	modernc.org/token v1.0.1 // indirect
)

replace (
	github.com/azusachino/felicia/apps/felicia-core => ../felicia-core
	github.com/azusachino/felicia/apps/felicia-providers => ../felicia-providers
	github.com/azusachino/felicia/apps/felicia-publication => ../felicia-publication
	github.com/azusachino/felicia/apps/felicia-runtime => ../felicia-runtime
)
