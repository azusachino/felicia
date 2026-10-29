//go:build darwin && !ios

package main

import "github.com/egoist/mygo"

func applyPlatformWindowChrome(options *mygo.WindowOptions) {
	options.TitleBarStyle = mygo.TitleBarHiddenInset
}
