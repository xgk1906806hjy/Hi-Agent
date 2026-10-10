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

	"github.com/xgk1906806hjy/Hi-Agent/internal/color"

	"golang.org/x/term"
)

const (
	panelWidth       = 34 // 右侧面板固定列宽
	minWidthForPanel = 80 // 终端宽不足时隐藏右栏
	renderInterval   = 40 * time.Millisecond
	maxEntries       = 3000
	maxProgressRunes = 600
	maxFoldDetail    = 12000 // 折叠块内部详情上限（仅存不展示）
)

// entry 消息区一条记录；open 表示可继续追加（模型正文或折叠过程块）。
type entry struct {
	style string
	text  string
	open  bool

	// 折叠过程块（style == "fold"）
	fold     bool
	detail   string
	tools    int
	thinks   int
	compacts int
	guards   int
	lastLbl  string
}

// TUI 轻量全屏界面：左侧消息、右侧面板、底部输入行。
// 输入在 ReadLine 时切 raw mode（可滚动、斜杠提示）；渲染用绝对定位整屏重绘。
type TUI struct {
	in    *bufio.Reader
	panel func() Panel

	mu         sync.Mutex
	entries    []entry
	busy       bool
	prompt     string
	inputBuf   string // raw 模式下当前编辑缓冲
	scrollOff  int    // 相对底部上翻的行数
	slashCmds  []SlashCmd
	lastRender time.Time
	dirty      bool
	think      ThinkStream
	foldOn     bool
	menu       *menuState // 非 nil 时左侧显示选择菜单

	started   bool
	stopOnce  sync.Once
	restoreVT func()
}

// New 创建 TUI；panel 在每次渲染时调用以获取最新面板数据。
func New(in *bufio.Reader, panel func() Panel) *TUI {
	return &TUI{in: in, panel: panel, foldOn: FoldEnabled()}
}

// SetSlashCommands 设置 `/` 输入时的命令提示列表。
func (t *TUI) SetSlashCommands(cmds []SlashCmd) {
	t.mu.Lock()
	t.slashCmds = append([]SlashCmd(nil), cmds...)
	t.mu.Unlock()
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
		disableMouse()
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

// Progress 追加工具/压缩等进度；默认折叠进单一「过程」摘要行。
func (t *TUI) Progress(s string) {
	s = strings.Trim(s, "\n")
	if s == "" {
		return
	}
	if !t.foldOn {
		t.add("tool", truncateRunes(s, maxProgressRunes))
		return
	}
	t.mu.Lock()
	t.appendFoldLocked(s)
	t.trim()
	t.mu.Unlock()
	t.Refresh()
}

// Model 追加模型流式正文；若含 <think> 等标签则拆入折叠过程块。
func (t *TUI) Model(s string) {
	if s == "" {
		return
	}
	if !t.foldOn {
		t.appendModel(s)
		return
	}
	think, model := t.think.Feed(s)
	if think != "" {
		t.mu.Lock()
		t.appendThinkLocked(think)
		t.trim()
		t.mu.Unlock()
		t.Refresh()
	}
	if model != "" {
		t.appendModel(model)
	}
}

func (t *TUI) appendModel(s string) {
	t.mu.Lock()
	// 正文开始前，若有开着的过程块先收成「已折叠」摘要
	t.sealOpenFoldLocked(false)
	if n := len(t.entries); n > 0 && t.entries[n-1].open && t.entries[n-1].style == "model" {
		t.entries[n-1].text += s
	} else {
		t.closeOpen()
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
	t.think = ThinkStream{}
	t.mu.Unlock()
	t.Flush()
}

// EndReply 结束生成、关闭开流段落并立即重绘。
func (t *TUI) EndReply() {
	t.mu.Lock()
	t.busy = false
	t.sealOpenFoldLocked(true)
	t.closeOpen()
	t.mu.Unlock()
	t.Flush()
}

// appendThinkLocked 将思考内容并入折叠块（不重复累加 thinks 计数）。
func (t *TUI) appendThinkLocked(chunk string) {
	if n := len(t.entries); n > 0 && t.entries[n-1].fold && t.entries[n-1].open {
		e := &t.entries[n-1]
		if e.thinks == 0 {
			e.thinks = 1
		}
		e.detail = appendDetail(e.detail, chunk)
		e.lastLbl = "思考"
		e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, e.lastLbl, false)
		return
	}
	t.closeOpen()
	e := entry{
		style:   "fold",
		fold:    true,
		open:    true,
		thinks:  1,
		lastLbl: "思考",
		detail:  truncateRunes(chunk, maxFoldDetail),
	}
	e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, e.lastLbl, false)
	t.entries = append(t.entries, e)
}

// appendFoldLocked 将进度行并入当前折叠块（调用方持锁）。
func (t *TUI) appendFoldLocked(s string) {
	kind, label := ClassifyProgress(s)
	if n := len(t.entries); n > 0 && t.entries[n-1].fold && t.entries[n-1].open {
		e := &t.entries[n-1]
		e.detail = appendDetail(e.detail, s)
		bumpFoldCounts(e, kind)
		if label != "" {
			e.lastLbl = label
		}
		e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, e.lastLbl, false)
		return
	}
	t.closeOpen()
	e := entry{
		style:  "fold",
		fold:   true,
		open:   true,
		detail: truncateRunes(s, maxFoldDetail),
	}
	bumpFoldCounts(&e, kind)
	e.lastLbl = label
	e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, e.lastLbl, false)
	t.entries = append(t.entries, e)
}

func bumpFoldCounts(e *entry, kind ProgressKind) {
	switch kind {
	case ProgressTool:
		e.tools++
	case ProgressThink:
		e.thinks++
	case ProgressCompact:
		e.compacts++
	case ProgressGuard:
		e.guards++
	}
}

func appendDetail(old, s string) string {
	if old == "" {
		return truncateRunes(s, maxFoldDetail)
	}
	out := old + "\n" + s
	return truncateRunes(out, maxFoldDetail)
}

// sealOpenFoldLocked 将开着的折叠块收成最终摘要；force 表示回复结束。
func (t *TUI) sealOpenFoldLocked(force bool) {
	n := len(t.entries)
	if n == 0 {
		return
	}
	e := &t.entries[n-1]
	if !e.fold || !e.open {
		return
	}
	if !force {
		// 正文到来时收拢：标记已折叠，但仍允许后续 Progress 开新块
		e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, "", true)
		e.open = false
		return
	}
	e.text = FoldSummary(e.tools, e.thinks, e.compacts, e.guards, "", true)
	e.open = false
}

// ReadLine 在底部输入行显示 prompt；raw 模式下支持 PgUp/PgDn 滚动与 `/` 命令提示。
func (t *TUI) ReadLine(prompt string) (string, error) {
	return t.rawReadLine(prompt)
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

	var view []styledLine
	if t.menu != nil {
		view = menuLines(t.menu, leftW, rows)
	} else {
		view = t.visibleLines(leftW, rows)
	}
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

	hint := t.hintLine(w - 1)
	fmt.Fprintf(&b, "\x1b[%d;1H", h-1)
	b.WriteString(color.Paint(color.Sys, fitFill(hint, w-1, '─')))
	b.WriteString("\x1b[K")

	fmt.Fprintf(&b, "\x1b[%d;1H\x1b[2K", h)
	if t.menu != nil {
		b.WriteString(color.Paint(color.Sys, "（菜单模式）"))
	} else if t.prompt == "" && t.busy {
		b.WriteString(color.Paint(color.Sys, "（生成中，Ctrl+C 可中断 · PgUp 上翻）"))
	} else {
		b.WriteString(color.Paint(color.User, t.prompt+t.inputBuf))
	}
	b.WriteString("\x1b[?25h")
	return b.String()
}

func (t *TUI) hintLine(width int) string {
	if t.menu != nil {
		return "─ ↑/↓ 选择 · Enter 确认 · Esc 取消 "
	}
	if matches := FilterSlashCmds(t.slashCmds, t.inputBuf); len(matches) > 0 {
		if h := formatSlashHint(matches, width); h != "" {
			return "─ " + h + " "
		}
	}
	if t.scrollOff > 0 {
		return fmt.Sprintf("─ ↑ 已上翻 %d 行 · PgDn/↓ 往下 · 输入回到底部 ", t.scrollOff)
	}
	return "─ Enter 发送 · 滚轮/PgUp 上翻 · / 命令提示 · Tab 补全 · Ctrl+C 中断 "
}

// visibleLines 折行后按 scrollOff 取窗口；scrollOff=0 贴底。
func (t *TUI) visibleLines(width, rows int) []styledLine {
	var all []styledLine
	for _, e := range t.entries {
		wrapped := wrap(e.text, width)
		for _, s := range wrapped {
			all = append(all, styledLine{style: e.style, text: s})
		}
	}
	total := len(all)
	if total == 0 {
		return nil
	}
	maxOff := total - rows
	if maxOff < 0 {
		maxOff = 0
	}
	if t.scrollOff > maxOff {
		t.scrollOff = maxOff
	}
	if t.scrollOff < 0 {
		t.scrollOff = 0
	}
	if total <= rows {
		return all
	}
	end := total - t.scrollOff
	start := end - rows
	if start < 0 {
		start = 0
	}
	return all[start:end]
}

func paint(style, s string) string {
	if style == "" {
		return s
	}
	if style == "fold" {
		style = "tool"
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
