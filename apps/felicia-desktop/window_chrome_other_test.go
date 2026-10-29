//go:build !darwin || ios

package main

import (
	"testing"

	"github.com/egoist/mygo"
)

func TestPlatformWindowChromeKeepsNativeDefaults(t *testing.T) {
	options := mygo.WindowOptions{Title: "Felicia Studio", Width: 1100, Height: 760}
	applyPlatformWindowChrome(&options)
	if options.TitleBarStyle != "" || options.Title != "Felicia Studio" || options.Width != 1100 || options.Height != 760 {
		t.Fatalf("platform chrome changed window options: %#v", options)
	}
}
