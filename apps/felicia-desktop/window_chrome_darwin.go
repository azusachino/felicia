//go:build darwin && !ios

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// applyPlatformWindowChrome maps Felicia's compact studio chrome to macOS.
// All other platforms retain their native Wails window defaults.
func applyPlatformWindowChrome(options *application.WebviewWindowOptions) {
	options.Mac.TitleBar = application.MacTitleBarHiddenInset
	options.Mac.Backdrop = application.MacBackdropLiquidGlass
	options.Mac.LiquidGlass = application.MacLiquidGlass{
		Style: application.LiquidGlassStyleAutomatic,
	}
}
