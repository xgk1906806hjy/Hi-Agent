//go:build !windows

package tui

// enableVT 非 Windows 终端通常已支持 ANSI，直接返回空恢复函数。
func enableVT() (func(), error) {
	return func() {}, nil
}
