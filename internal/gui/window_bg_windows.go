//go:build desktop && windows

package gui

import (
	"golang.org/x/sys/windows/registry"

	"github.com/wailsapp/wails/v2/pkg/options"
)

// windowBackground 跟随系统应用主题，避免深色模式下窗口先闪白。
func windowBackground() *options.RGBA {
	light := &options.RGBA{R: 246, G: 245, B: 242, A: 255}
	dark := &options.RGBA{R: 26, G: 25, B: 23, A: 255}
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
	if err != nil {
		return light
	}
	defer k.Close()
	v, _, err := k.GetIntegerValue("AppsUseLightTheme")
	if err != nil || v != 0 {
		return light
	}
	return dark
}
