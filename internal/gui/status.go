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

// FormatSidebar 生成侧栏纯文本。
func (s StatusSnapshot) FormatSidebar() string {
	state := "空闲"
	if s.Busy {
		state = "生成中…"
	}
	ctx := fmt.Sprintf("%d", s.CtxTokens)
	if s.CtxEstimated && s.CtxTokens > 0 {
		ctx = "≈" + ctx
	}
	return fmt.Sprintf(
		"Hi-agent\n\n状态  %s\n模型  %s\n会话  %s\n目录  %s\n\n上下文  %s / %d\n历史    %d / %d 字符\n消息    %d\n\n本轮\n  prompt %d\n  completion %d\n  requests %d\n\n累计\n  prompt %d\n  completion %d\n  requests %d\n\n工具  %d 轮 / %d 次",
		state, s.Model, s.Session, s.WorkDir,
		ctx, s.CtxWindow, s.HistChars, s.HistLimit, s.Messages,
		s.TurnPrompt, s.TurnCompletion, s.TurnRequests,
		s.CumPrompt, s.CumCompletion, s.CumRequests,
		s.ToolTurns, s.ToolCalls,
	)
}
