//go:build darwin && !ios

package main

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestPlatformWindowChromeUsesMacStudioTreatment(t *testing.T) {
	options := mygo.WindowOptions{}
	applyPlatformWindowChrome(&options)
	if options.TitleBarStyle != mygo.TitleBarHiddenInset {
		t.Fatalf("titlebar = %q, want hidden inset", options.TitleBarStyle)
	}
}
