//go:build !darwin || ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// applyPlatformWindowChrome leaves non-macOS windows on Wails' native defaults.
func applyPlatformWindowChrome(_ *application.WebviewWindowOptions) {}
