//go:build desktop && !windows

package gui

import "github.com/wailsapp/wails/v2/pkg/options"

func windowBackground() *options.RGBA {
	return &options.RGBA{R: 246, G: 245, B: 242, A: 255}
}
