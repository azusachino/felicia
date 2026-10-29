//go:build !darwin || ios

package main

import "github.com/egoist/mygo"

func applyPlatformWindowChrome(_ *mygo.WindowOptions) {}
