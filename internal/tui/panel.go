package tui

import (
	"fmt"
	"strings"
)

// Panel 右侧面板数据，由调用方在每次渲染时提供。
type Panel struct {
	Model   string
	Session string
	Busy    bool // 由 TUI 自己根据 BeginReply/EndReply 覆盖

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

// styledLine 一行带样式标签的纯文本（颜色在渲染阶段套用）。
type styledLine struct {
	style string
	text  string
}

// panelLines 生成面板内容；每行文本不含颜色码，宽度由渲染阶段 fit。
func panelLines(p Panel, width int) []styledLine {
	inner := width - 1
	if inner < 4 {
		inner = 4
	}
	state := "空闲"
	stateStyle := "sys"
	if p.Busy {
		state = "生成中…"
		stateStyle = "model"
	}

	ctx := fmtTokens(p.CtxTokens)
	if p.CtxEstimated && p.CtxTokens > 0 {
		ctx = "≈" + ctx
	}
	pct := 0
	if p.CtxWindow > 0 {
		pct = p.CtxTokens * 100 / p.CtxWindow
	}

	lines := []styledLine{
		{"user", "Hi-agent"},
		{"sys", strings.Repeat("─", inner)},
		{"sys", "模型 " + p.Model},
		{"sys", "会话 " + p.Session},
		{stateStyle, "状态 " + state},
		{"", ""},
		{"user", "上下文"},
		{ctxStyle(pct), bar(pct, inner-7) + fmt.Sprintf(" %3d%%", pct)},
		{"sys", fmt.Sprintf("%s / %s tokens", ctx, fmtTokens(p.CtxWindow))},
		{"sys", fmt.Sprintf("历史 %s / %s 字符", fmtTokens(p.HistChars), fmtTokens(p.HistLimit))},
		{"sys", fmt.Sprintf("消息 %d 条", p.Messages)},
		{"", ""},
		{"user", "本轮 tokens"},
		{"sys", fmt.Sprintf("输入 %s  输出 %s", fmtTokens(p.TurnPrompt), fmtTokens(p.TurnCompletion))},
		{"sys", fmt.Sprintf("请求 %d 次", p.TurnRequests)},
		{"sys", fmt.Sprintf("工具 %d 轮 / %d 次", p.ToolTurns, p.ToolCalls)},
		{"", ""},
		{"user", "累计 tokens"},
		{"sys", fmt.Sprintf("输入 %s  输出 %s", fmtTokens(p.CumPrompt), fmtTokens(p.CumCompletion))},
		{"sys", fmt.Sprintf("合计 %s（%d 次）", fmtTokens(p.CumPrompt+p.CumCompletion), p.CumRequests)},
		{"", ""},
		{"sys", "/help 命令  Ctrl+C 中断"},
	}
	return lines
}

// ctxStyle 按占用百分比选样式：≥80% 用警示色。
func ctxStyle(pct int) string {
	switch {
	case pct >= 80:
		return "tool"
	default:
		return "model"
	}
}

// bar 用 ASCII 画进度条，避免 East Asian 歧义宽度字符导致错位。
func bar(pct, width int) string {
	if width < 4 {
		width = 4
	}
	inner := width - 2
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := pct * inner / 100
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", inner-filled) + "]"
}

// fmtTokens 把 token/字符数格式化为短读法（k / M）。
func fmtTokens(n int) string {
	switch {
	case n < 1000:
		return fmt.Sprintf("%d", n)
	case n < 1_000_000:
		return fmt.Sprintf("%.1fk", float64(n)/1000)
	default:
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	}
}
