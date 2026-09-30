//go:build !windows

package tui

func enableVT() (func(), error) {
	return func() {}, nil
}
