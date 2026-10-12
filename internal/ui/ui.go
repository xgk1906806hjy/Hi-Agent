// Package ui 定义可插拔的用户界面抽象与工厂。
//
// 业务层（REPL）只依赖 UI 接口；具体实现（plain / tui / 未来 gui、web）
// 通过 Register 注册，由 Create / MustCreate 按 Kind 创建。
package ui

import "bufio"

// Kind 标识一种 UI 实现。
type Kind string

const (
	KindPlain Kind = "plain" // 纯文本终端
	KindTUI   Kind = "tui"   // 轻量全屏 TUI（已实现）
	KindGUI   Kind = "gui"   // 桌面 GUI（Fyne，已实现）
	KindWeb   Kind = "web"   // Web UI（预留）
)

// SlashCmd 斜杠命令提示项（各 UI 可选用）。
type SlashCmd struct {
	Name string
	Desc string
}

// UI 是面向 REPL / Agent 的统一交互面。
// 后续 GUI、Web 只需实现本接口并 Register 即可接入。
type UI interface {
	Info(s string)
	Notice(s string)
	Error(s string)
	UserEcho(s string)
	Progress(s string)
	Model(s string)
	BeginReply()
	EndReply()
	Refresh()
	ReadLine(prompt string) (string, error)
	Confirm(prompt string) bool
	Select(title string, items []string, selected int) (int, error)
	Close()
}

// Options 创建 UI 时的通用参数。
type Options struct {
	In            *bufio.Reader
	SlashCommands []SlashCmd
	// PanelFn 仅 TUI 使用，类型应为 func() tui.Panel；其它 Kind 忽略。
	PanelFn any
	// StatusFn 仅 GUI 使用，类型应为 func() gui.StatusSnapshot。
	StatusFn any
	// NavFn 仅 GUI 使用，类型应为 func() gui.NavSnapshot（左侧项目与会话）。
	NavFn any
}

// DesktopRunner 桌面 GUI 需在主线程阻塞 Run；REPL 应放到 goroutine。
type DesktopRunner interface {
	UI
	Run() // 阻塞直到窗口关闭
}

// Canceler 可选：界面自带「停止」按钮时实现，回复期间由 REPL 注入取消函数，结束后传 nil。
type Canceler interface {
	SetCancel(cancel func())
}
