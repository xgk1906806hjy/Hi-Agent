//go:build windows

package tui

import (
	"os"

	"golang.org/x/sys/windows"
)

// enableVT 打开 Windows 控制台的 ANSI 转义处理；返回恢复原模式的函数。
func enableVT() (func(), error) {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return func() {}, err
	}
	if err := windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING); err != nil {
		return func() {}, err
	}
	return func() { _ = windows.SetConsoleMode(h, mode) }, nil
}
