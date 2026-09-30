// Package color 为 REPL / TUI 提供统一的 ANSI 终端着色。
//
// 仅在 stdout 为 TTY 时启用颜色；管道或重定向时自动退化为纯文本。
// 颜色键在 init 时检测一次，运行中切换重定向不会更新。
package color

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// 颜色键，与业务角色约定一致（user / model / tool / sys）。
const (
	User  = "user"  // 用户提示符等，青色
	Model = "model" // 模型正文，绿色
	Tool  = "tool"  // 工具进度 / 确认 / 警告，黄色
	Sys   = "sys"   // 系统信息 / Banner，灰色
)

// ansi 映射颜色键到 SGR 转义序列；reset 用于复位。
var ansi = map[string]string{
	"reset": "\x1b[0m",
	"user":  "\x1b[36m", // 青
	"model": "\x1b[32m", // 绿
	"tool":  "\x1b[33m", // 黄
	"sys":   "\x1b[90m", // 灰
}

// useColor：包加载时检测 stdout 是否为交互终端。
var useColor = term.IsTerminal(int(os.Stdout.Fd()))

// Paint 给字符串包上颜色并复位。
// 非 TTY、或 name 不在表中时返回原文（不报错）。
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

// Out 写到 stdout；nl 为 true 时补换行（流式片段常用 nl=false）。
func Out(name, s string, nl bool) {
	text := Paint(name, s)
	if nl {
		fmt.Fprintln(os.Stdout, text)
		return
	}
	fmt.Fprint(os.Stdout, text)
}

// Err 写到 stderr 并换行；着色开关与 Out 相同。
func Err(name, s string) {
	fmt.Fprintln(os.Stderr, Paint(name, s))
}
