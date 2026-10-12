// Package gui 提供基于 Fyne 的桌面 GUI 对话壳（非浏览器）。
package gui

import "fmt"

// StatusSnapshot 右侧状态栏数据（与 TUI 面板字段对齐，避免 gui 依赖 tui 类型）。
type StatusSnapshot struct {
	Model   string
	Session string
	WorkDir string
	Busy    bool

	CtxTokens    int
	CtxEstimated bool
	CtxWindow    int

	HistChars int
	HistLimit int
	Messages  int

	TurnPrompt, TurnCompletion, TurnRequests int
	CumPrompt, CumCompletion, CumRequests    int

	ToolTurns, ToolCalls int
}

// FormatSidebar 生成侧栏紧凑文案（Codex 向：少字段、非等宽 dump）。
func (s StatusSnapshot) FormatSidebar() string {
	state := "空闲"
	if s.Busy {
		state = "生成中…"
	}
	ctx := fmt.Sprintf("%d", s.CtxTokens)
	if s.CtxEstimated && s.CtxTokens > 0 {
		ctx = "≈" + ctx
	}
	win := s.CtxWindow
	if win <= 0 {
		win = 0
	}
	dir := s.WorkDir
	if len(dir) > 42 {
		dir = "…" + dir[len(dir)-40:]
	}
	return fmt.Sprintf(
		"%s\n\n%s\n%s\n%s\n\n上下文  %s / %d\n本轮  %d + %d\n消息 %d",
		state,
		emptyDash(s.Model), emptyDash(s.Session), emptyDash(dir),
		ctx, win,
		s.TurnPrompt, s.TurnCompletion,
		s.Messages,
	)
}

func emptyDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// ModelShort 顶栏用模型短名。
func (s StatusSnapshot) ModelShort() string {
	if s.Model == "" {
		return ""
	}
	return s.Model
}
