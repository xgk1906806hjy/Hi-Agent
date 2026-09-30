package color

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// 颜色键，与业务约定一致。
const (
	User  = "user"
	Model = "model"
	Tool  = "tool"
	Sys   = "sys"
)

var ansi = map[string]string{
	"reset": "\x1b[0m",
	"user":  "\x1b[36m", // 青
	"model": "\x1b[32m", // 绿
	"tool":  "\x1b[33m", // 黄
	"sys":   "\x1b[90m", // 灰
}

// useColor：stdout 为 TTY 时启用着色。
var useColor = term.IsTerminal(int(os.Stdout.Fd()))

// Paint 给字符串包上颜色并复位。
func Paint(name, s string) string {
	if !useColor {
		return s
	}
	code, ok := ansi[name]
	if !ok {
		return s
	}
	return code + s + ansi["reset"]
}

// Out 写到 stdout；nl 为 true 时补换行。
func Out(name, s string, nl bool) {
	text := Paint(name, s)
	if nl {
		fmt.Fprintln(os.Stdout, text)
		return
	}
	fmt.Fprint(os.Stdout, text)
}

// Err 写到 stderr 并换行。
func Err(name, s string) {
	fmt.Fprintln(os.Stderr, Paint(name, s))
}
