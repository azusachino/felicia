package main

import (
	"bytes"
	"image/png"
	"testing"
)

func TestDefaultApplicationIconIsDecodable(t *testing.T) {
	icon, err := png.DecodeConfig(bytes.NewReader(defaultAppIcon))
	if err != nil {
		t.Fatalf("default application icon: %v", err)
	}
	if icon.Width < 256 || icon.Width != icon.Height {
		t.Fatalf("invalid application icon dimensions: %dx%d", icon.Width, icon.Height)
	}
}
