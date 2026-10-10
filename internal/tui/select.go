package tui

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

)

// menuState 全屏选择菜单（盖住消息区展示）。
type menuState struct {
	title    string
	items    []string
	selected int
	hint     string
}

// Select 用 ↑/↓（或滚轮）选择一项，Enter 确认，Esc 取消（返回 -1）。
func (t *TUI) Select(title string, items []string, selected int) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("没有可选项")
	}
	if selected < 0 {
		selected = 0
	}
	if selected >= len(items) {
		selected = len(items) - 1
	}

	fd := int(os.Stdin.Fd())
	old, err := term.MakeRaw(fd)
	if err != nil {
		// 无 raw 时退回：调用方应处理；这里仍尽量给出序号体验
		return -1, err
	}
	defer func() {
		t.mu.Lock()
		t.menu = nil
		t.mu.Unlock()
		disableMouse()
		_ = term.Restore(fd, old)
		t.Flush()
	}()
	enableMouse()

	idx := selected
	for {
		t.mu.Lock()
		t.menu = &menuState{
			title:    title,
			items:    append([]string(nil), items...),
			selected: idx,
			hint:     "↑/↓ 移动 · Enter 确认 · Esc 取消",
		}
		t.mu.Unlock()
		t.Flush()

		k, err := readKey(os.Stdin)
		if err != nil {
			return -1, err
		}
		switch k.kind {
		case keyEnter:
			return idx, nil
		case keyEscape, keyCtrlC:
			return -1, nil
		case keyUp, keyWheelUp, keyPgUp:
			if idx > 0 {
				idx--
			} else {
				idx = len(items) - 1
			}
		case keyDown, keyWheelDown, keyPgDn:
			if idx < len(items)-1 {
				idx++
			} else {
				idx = 0
			}
		case keyRune:
			// 数字快捷：1-9
			if k.r >= '1' && k.r <= '9' {
				n := int(k.r - '1')
				if n < len(items) {
					idx = n
				}
			}
		}
	}
}

// menuLines 将菜单渲染为左侧可视行。
func menuLines(m *menuState, width, rows int) []styledLine {
	if m == nil {
		return nil
	}
	var out []styledLine
	out = append(out, styledLine{style: "sys", text: m.title})
	out = append(out, styledLine{style: "sys", text: strings.Repeat("─", max(8, width-2))})
	// 可视窗口：让选中项尽量居中
	maxItems := rows - 4
	if maxItems < 3 {
		maxItems = 3
	}
	start := 0
	if len(m.items) > maxItems {
		start = m.selected - maxItems/2
		if start < 0 {
			start = 0
		}
		if start+maxItems > len(m.items) {
			start = len(m.items) - maxItems
		}
	}
	end := start + maxItems
	if end > len(m.items) {
		end = len(m.items)
	}
	for i := start; i < end; i++ {
		prefix := "  "
		style := "sys"
		if i == m.selected {
			prefix = "▸ "
			style = "user"
		}
		line := prefix + m.items[i]
		out = append(out, styledLine{style: style, text: fit(line, width)})
	}
	if start > 0 || end < len(m.items) {
		out = append(out, styledLine{style: "sys", text: fmt.Sprintf("  … %d/%d", m.selected+1, len(m.items))})
	}
	out = append(out, styledLine{style: "", text: ""})
	hint := m.hint
	if hint == "" {
		hint = "↑/↓ · Enter · Esc"
	}
	out = append(out, styledLine{style: "tool", text: hint})
	if len(out) > rows {
		out = out[:rows]
	}
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
