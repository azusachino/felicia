//go:build darwin && !ios

package main

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestPlatformWindowChromeUsesMacStudioTreatment(t *testing.T) {
	options := application.WebviewWindowOptions{}
	applyPlatformWindowChrome(&options)
	if options.Mac.TitleBar != application.MacTitleBarHiddenInset {
		t.Fatalf("titlebar = %#v, want HiddenInset", options.Mac.TitleBar)
	}
	if options.Mac.Backdrop != application.MacBackdropLiquidGlass {
		t.Fatalf("backdrop = %v, want LiquidGlass", options.Mac.Backdrop)
	}
	if options.Mac.LiquidGlass.Style != application.LiquidGlassStyleAutomatic {
		t.Fatalf("glass style = %v, want Automatic", options.Mac.LiquidGlass.Style)
	}
}
