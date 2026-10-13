//go:build desktop && !windows

package gui

import "github.com/wailsapp/wails/v2/pkg/options"

func windowBackground() *options.RGBA {
	return &options.RGBA{R: 250, G: 250, B: 250, A: 255}
}
