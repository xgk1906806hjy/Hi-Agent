package tui

import (
	"strings"
	"unicode/utf8"
)

// runeWidth 返回字符在终端中的显示列宽（CJK / 全角 / 常见 emoji 为 2，控制与组合字符为 0）。
func runeWidth(r rune) int {
	switch {
	case r == 0:
		return 0
	case r < 0x20 || (r >= 0x7f && r < 0xa0):
		return 0
	case r >= 0x300 && r <= 0x36f, r >= 0x200b && r <= 0x200f, r == 0xfe0f:
		return 0
	case r >= 0x1100 && r <= 0x115f,
		r >= 0x2e80 && r <= 0x303e,
		r >= 0x3041 && r <= 0x33ff,
		r >= 0x3400 && r <= 0x4dbf,
		r >= 0x4e00 && r <= 0x9fff,
		r >= 0xa000 && r <= 0xa4cf,
		r >= 0xac00 && r <= 0xd7a3,
		r >= 0xf900 && r <= 0xfaff,
		r >= 0xfe30 && r <= 0xfe4f,
		r >= 0xff00 && r <= 0xff60,
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f300 && r <= 0x1f64f,
		r >= 0x1f900 && r <= 0x1f9ff,
		r >= 0x20000 && r <= 0x3fffd:
		return 2
	}
	return 1
}

// textWidth 返回字符串显示宽度（按 runeWidth 累加）。
func textWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}
	return w
}

// sanitize 去掉会破坏布局的控制字符：\t 展开为 4 空格，\r 与 ESC 等丢弃，保留 \n。
func sanitize(s string) string {
	if s == "" {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteString("    ")
		case r == utf8.RuneError, r < 0x20, r == 0x7f:
			// 丢弃控制字符与非法 UTF-8
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// wrap 按显示宽度把文本折成多行；width<=0 时返回原文按换行拆分。
func wrap(s string, width int) []string {
	s = sanitize(s)
	paras := strings.Split(s, "\n")
	if width <= 0 {
		return paras
	}
	var out []string
	for _, p := range paras {
		if p == "" {
			out = append(out, "")
			continue
		}
		var line strings.Builder
		w := 0
		for _, r := range p {
			rw := runeWidth(r)
			if w+rw > width && w > 0 {
				out = append(out, line.String())
				line.Reset()
				w = 0
			}
			line.WriteRune(r)
			w += rw
		}
		out = append(out, line.String())
	}
	return out
}

// fit 把单行截断/补空格到恰好 width 列（按显示宽度，非字节数）。
func fit(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := 0
	var b strings.Builder
	for _, r := range s {
		rw := runeWidth(r)
		if w+rw > width {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	if w < width {
		b.WriteString(strings.Repeat(" ", width-w))
	}
	return b.String()
}

// truncateRunes 限制展示用文本 rune 数，超出时追加截断提示。
func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "…（界面已截断显示）"
}
