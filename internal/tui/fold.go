package tui

import (
	"fmt"
	"os"
	"strings"
)

// FoldEnabled 是否折叠工具/思考过程（GEEKAGENT_FOLD=0 关闭）。
func FoldEnabled() bool {
	v := strings.TrimSpace(os.Getenv("GEEKAGENT_FOLD"))
	return v != "0" && !strings.EqualFold(v, "false")
}

// ProgressKind 解析进度行类别，供折叠摘要。
type ProgressKind int

const (
	ProgressOther ProgressKind = iota
	ProgressTool
	ProgressThink
	ProgressCompact
	ProgressGuard
)

// ClassifyProgress 识别进度行类型与短标签。
func ClassifyProgress(s string) (ProgressKind, string) {
	s = strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(s, "[调用工具"):
		name := "tool"
		rest := strings.TrimPrefix(s, "[调用工具")
		rest = strings.TrimSpace(rest)
		if i := strings.IndexAny(rest, " →\t"); i > 0 {
			name = strings.TrimSpace(rest[:i])
		} else if i := strings.Index(rest, "→"); i > 0 {
			name = strings.TrimSpace(rest[:i])
		}
		return ProgressTool, name
	case strings.HasPrefix(s, "[思考"):
		return ProgressThink, "思考"
	case strings.HasPrefix(s, "[历史压缩"):
		return ProgressCompact, "压缩"
	case strings.HasPrefix(s, "[工具轮次"),
		strings.HasPrefix(s, "[重复工具"),
		strings.HasPrefix(s, "[回复预算"),
		strings.HasPrefix(s, "[工具集"):
		return ProgressGuard, "护栏"
	default:
		return ProgressOther, "过程"
	}
}

// FoldSummary 生成折叠块展示行。
func FoldSummary(tools, thinks, compacts, guards int, lastLabel string, done bool) string {
	var parts []string
	if tools > 0 {
		parts = append(parts, fmt.Sprintf("工具×%d", tools))
	}
	if thinks > 0 {
		parts = append(parts, "思考")
	}
	if compacts > 0 {
		parts = append(parts, "压缩")
	}
	if guards > 0 {
		parts = append(parts, "护栏")
	}
	body := strings.Join(parts, " · ")
	if body == "" {
		body = "过程"
	}
	if !done && lastLabel != "" {
		body += " · " + lastLabel
	}
	if done {
		return "▸ " + body + "（已折叠）"
	}
	return "▸ " + body + "（进行中…）"
}

// ShortProgressLine 纯文本模式下的一行摘要（不含完整工具结果）。
func ShortProgressLine(s string) string {
	s = strings.TrimSpace(s)
	kind, label := ClassifyProgress(s)
	switch kind {
	case ProgressTool:
		return "▸ 工具 " + label
	case ProgressThink:
		return "▸ 思考（已折叠）"
	case ProgressCompact:
		return "▸ 历史压缩"
	case ProgressGuard:
		brief := s
		if len([]rune(brief)) > 60 {
			brief = string([]rune(brief)[:60]) + "…"
		}
		return "▸ " + strings.Trim(brief, "[]")
	default:
		if len([]rune(s)) > 72 {
			s = string([]rune(s)[:72]) + "…"
		}
		return "▸ " + s
	}
}
