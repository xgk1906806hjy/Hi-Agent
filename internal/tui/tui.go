// Package tui 提供轻量全屏终端界面：左侧消息区、右侧状态面板、底部输入行。
// 输入仍走 cooked mode（bufio 读行）；渲染用 ANSI 绝对定位整屏重绘，不依赖 TUI 框架。
package tui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"hi-agent/internal/color"

	"golang.org/x/term"
)

const (
	panelWidth       = 34                 // 右侧面板固定列宽
	minWidthForPanel = 80                 // 终端宽不足时隐藏右栏
	renderInterval   = 40 * time.Millisecond
	maxEntries       = 3000
	maxProgressRunes = 600
)

// entry 消息区一条记录；open 表示模型流式正文可继续追加到本条。
type entry struct {
	style string
	text  string
	open  bool
}

// TUI 轻量全屏界面：左侧消息、右侧面板、底部输入行。
// 输入仍用终端行缓冲（cooked mode）读取，渲染用绝对定位整屏重绘。
type TUI struct {
	in    *bufio.Reader
	panel func() Panel

	mu         sync.Mutex
	entries    []entry
	busy       bool
	prompt     string
	lastRender time.Time
	dirty      bool

	started   bool
	stopOnce  sync.Once
	restoreVT func()
}

// New 创建 TUI；panel 在每次渲染时调用以获取最新面板数据。
func New(in *bufio.Reader, panel func() Panel) *TUI {
	return &TUI{in: in, panel: panel}
}

// Supported 判断 stdin 与 stdout 是否均为终端。
func Supported() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) && term.IsTerminal(int(os.Stdout.Fd()))
}

// Start 进入备用屏幕并首次绘制。
func (t *TUI) Start() error {
	restore, err := enableVT()
	if err != nil {
		return fmt.Errorf("无法启用终端 ANSI 处理：%w", err)
	}
	t.restoreVT = restore
	t.started = true
	os.Stdout.WriteString("\x1b[?1049h\x1b[2J")
	t.Flush()
	return nil
}

// Close 退出备用屏幕并恢复终端；可重复调用。
func (t *TUI) Close() {
	t.stopOnce.Do(func() {
		if !t.started {
			return
		}
		os.Stdout.WriteString("\x1b[0m\x1b[?25h\x1b[?1049l")
		if t.restoreVT != nil {
			t.restoreVT()
		}
	})
}

// Info 追加系统信息（灰）。
func (t *TUI) Info(s string) { t.add("sys", s) }

// Notice 追加提示信息（黄）。
func (t *TUI) Notice(s string) { t.add("tool", s) }

// Error 追加错误信息。
func (t *TUI) Error(s string) { t.add("tool", strings.TrimLeft(s, "\n")) }

// UserEcho 追加用户输入（先关闭开流段落，再插入空行与 You › 行）。
func (t *TUI) UserEcho(s string) {
	t.mu.Lock()
	t.closeOpen()
	t.entries = append(t.entries, entry{style: "", text: ""})
	t.entries = append(t.entries, entry{style: "user", text: "You › " + s})
	t.trim()
	t.mu.Unlock()
	t.Flush()
}

// Progress 追加工具/压缩等进度行（去掉首尾换行，过长截断展示）。
func (t *TUI) Progress(s string) {
	s = strings.Trim(s, "\n")
	t.add("tool", truncateRunes(s, maxProgressRunes))
}

// Model 追加模型流式正文（与上一段正文合并）。
func (t *TUI) Model(s string) {
	t.mu.Lock()
	if n := len(t.entries); n > 0 && t.entries[n-1].open {
		t.entries[n-1].text += s
	} else {
		t.entries = append(t.entries, entry{style: "model", text: strings.TrimLeft(s, "\n"), open: true})
	}
	t.trim()
	t.mu.Unlock()
	t.Refresh()
}

// BeginReply 标记生成中（面板状态与底部提示）。
func (t *TUI) BeginReply() {
	t.mu.Lock()
	t.busy = true
	t.mu.Unlock()
	t.Flush()
}

// EndReply 结束生成、关闭开流段落并立即重绘。
func (t *TUI) EndReply() {
	t.mu.Lock()
	t.busy = false
	t.closeOpen()
	t.mu.Unlock()
	t.Flush()
}

// ReadLine 在底部输入行显示 prompt 并读取一行。
func (t *TUI) ReadLine(prompt string) (string, error) {
	t.mu.Lock()
	t.prompt = prompt
	t.mu.Unlock()
	t.Flush()
	line, err := t.in.ReadString('\n')
	t.mu.Lock()
	t.prompt = ""
	t.mu.Unlock()
	return strings.TrimSpace(line), err
}

// Confirm 把确认内容（如 diff）写入消息区，并在输入行询问 y/N。
func (t *TUI) Confirm(prompt string) bool {
	t.Notice(prompt)
	ans, err := t.ReadLine("确认？[y/N] ")
	if err != nil {
		return false
	}
	ans = strings.ToLower(ans)
	ok := ans == "y" || ans == "yes"
	if ok {
		t.Info("（已确认）")
	} else {
		t.Info("（已取消）")
	}
	return ok
}

// Refresh 节流重绘：距上次渲染不足间隔时只标记 dirty。
func (t *TUI) Refresh() {
	t.mu.Lock()
	due := time.Since(t.lastRender) >= renderInterval
	if !due {
		t.dirty = true
	}
	t.mu.Unlock()
	if due {
		t.Flush()
	}
}

// Flush 立即整屏重绘。
func (t *TUI) Flush() {
	if !t.started {
		return
	}
	var p Panel
	if t.panel != nil {
		p = t.panel()
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	p.Busy = t.busy
	frame := t.frame(p)
	t.lastRender = time.Now()
	t.dirty = false
	os.Stdout.WriteString(frame)
}

// add 关闭开流后追加一条消息，并走节流刷新。
func (t *TUI) add(style, s string) {
	t.mu.Lock()
	t.closeOpen()
	t.entries = append(t.entries, entry{style: style, text: s})
	t.trim()
	t.mu.Unlock()
	t.Refresh()
}

// closeOpen 结束当前模型流式段落，避免后续非 Model 写入被合并。
func (t *TUI) closeOpen() {
	if n := len(t.entries); n > 0 {
		t.entries[n-1].open = false
	}
}

// trim 条目过多时丢弃较早部分，保留约 2/3 上限条数。
func (t *TUI) trim() {
	if len(t.entries) > maxEntries {
		t.entries = append([]entry(nil), t.entries[len(t.entries)-maxEntries*2/3:]...)
	}
}

// screenSize 读取终端尺寸；失败时回退到 100×30。
func screenSize() (int, int) {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 || h <= 0 {
		return 100, 30
	}
	return w, h
}

// frame 生成一整帧的 ANSI 输出（调用方持锁）。
// 布局：可视行 = 高−2；宽≥80 时左栏消息 + 右栏面板，否则消息占满。
func (t *TUI) frame(p Panel) string {
	w, h := screenSize()
	if h < 4 {
		h = 4
	}
	pw := 0
	if w >= minWidthForPanel {
		pw = panelWidth
	}
	leftW := w - 1
	if pw > 0 {
		leftW = w - 1 - pw - 1 // 减分隔符与右栏
	}
	rows := h - 2 // 提示行 + 输入行

	view := t.visibleLines(leftW, rows)
	var pl []styledLine
	if pw > 0 {
		pl = panelLines(p, pw)
	}

	var b strings.Builder
	b.WriteString("\x1b[?25l")
	for r := 0; r < rows; r++ {
		fmt.Fprintf(&b, "\x1b[%d;1H", r+1)
		var left styledLine
		if r < len(view) {
			left = view[r]
		}
		b.WriteString(paint(left.style, fit(left.text, leftW)))
		if pw > 0 {
			b.WriteString(color.Paint(color.Sys, "│"))
			var pr styledLine
			if r < len(pl) {
				pr = pl[r]
			}
			b.WriteString(paint(pr.style, fit(" "+pr.text, pw)))
		}
		b.WriteString("\x1b[K")
	}

	hint := "─ Enter 发送 · Ctrl+C 中断回复 · /help 命令 "
	fmt.Fprintf(&b, "\x1b[%d;1H", h-1)
	b.WriteString(color.Paint(color.Sys, fitFill(hint, w-1, '─')))
	b.WriteString("\x1b[K")

	fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", h)
	prompt := t.prompt
	if prompt == "" && t.busy {
		b.WriteString(color.Paint(color.Sys, "（生成中，Ctrl+C 可中断）"))
	} else {
		b.WriteString(color.Paint(color.User, prompt))
	}
	b.WriteString("\x1b[?25h")
	return b.String()
}

// visibleLines 从最新消息倒推折行，只保留填满可视区所需的尾部行。
func (t *TUI) visibleLines(width, rows int) []styledLine {
	var lines []styledLine
	for i := len(t.entries) - 1; i >= 0 && len(lines) < rows; i-- {
		e := t.entries[i]
		wrapped := wrap(e.text, width)
		chunk := make([]styledLine, len(wrapped))
		for j, s := range wrapped {
			chunk[j] = styledLine{style: e.style, text: s}
		}
		lines = append(chunk, lines...)
	}
	if len(lines) > rows {
		lines = lines[len(lines)-rows:]
	}
	return lines
}

func paint(style, s string) string {
	if style == "" {
		return s
	}
	return color.Paint(style, s)
}

// fitFill 与 fit 类似，但用 fill 字符把剩余列补齐（分隔提示行）。
func fitFill(s string, width int, fill rune) string {
	out := fit(s, width)
	trimmed := strings.TrimRight(out, " ")
	pad := width - textWidth(trimmed)
	if pad <= 0 {
		return trimmed
	}
	return trimmed + strings.Repeat(string(fill), pad)
}
