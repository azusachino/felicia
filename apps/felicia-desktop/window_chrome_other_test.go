//go:build !darwin || ios

package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestPlatformWindowChromeKeepsNativeDefaults(t *testing.T) {
	options := application.WebviewWindowOptions{Title: "Felicia Studio", Width: 1100, Height: 760}
	applyPlatformWindowChrome(&options)
	if options.Mac.Backdrop != application.MacBackdropNormal || options.Mac.TitleBar.Hide || options.Mac.TitleBar.FullSizeContent || options.Mac.LiquidGlass.Style != application.LiquidGlassStyleAutomatic {
		t.Fatalf("non-macOS chrome unexpectedly set macOS options: %#v", options.Mac)
	}
	if options.Title != "Felicia Studio" || options.Width != 1100 || options.Height != 760 {
		t.Fatalf("platform chrome changed shared window options: %#v", options)
	}
}
