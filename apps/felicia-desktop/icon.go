package main

import _ "embed"

// Supply the same default icon to the running application as the app bundle.
// This also covers direct executable launches without LaunchServices metadata.
//
//go:embed build/appicon.png
var defaultAppIcon []byte
