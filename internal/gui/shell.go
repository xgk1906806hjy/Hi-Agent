package gui

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type msgKind int

const (
	msgSys msgKind = iota
	msgUser
	msgModel
	msgTool
	msgErr
)

type chatMsg struct {
	kind msgKind
	text string
	open bool
}

// SlashItem 底部快捷斜杠。
type SlashItem struct {
	Name string
	Desc string
}

// Options 创建 Shell。
type Options struct {
	Title         string
	StatusFn      func() StatusSnapshot
	SlashCommands []SlashItem
}

// Shell 是 Fyne 桌面对话壳。
type Shell struct {
	app fyne.App
	win fyne.Window

	chatBox    *fyne.Container
	chatScroll *container.Scroll
	sideLabel  *widget.Label
	entry      *widget.Entry
	sendBtn    *widget.Button
	promptLbl  *widget.Label
	statusFn   func() StatusSnapshot

	mu     sync.Mutex
	msgs   []chatMsg
	busy   bool
	prompt string

	lines     chan string
	done      chan struct{}
	closeOnce sync.Once
}

// New 创建窗口（尚未 Run）。
func New(opts Options) *Shell {
	s := &Shell{
		app:      app.NewWithID("com.hi-agent.gui"),
		lines:    make(chan string, 8),
		done:     make(chan struct{}),
		statusFn: opts.StatusFn,
		prompt:   "You › ",
	}
	title := opts.Title
	if title == "" {
		title = "Hi-agent"
	}
	s.win = s.app.NewWindow(title)
	s.win.Resize(fyne.NewSize(1100, 720))
	s.build(opts)
	s.win.SetCloseIntercept(func() {
		s.signalClosed()
		s.win.Close()
	})
	return s
}

func (s *Shell) build(opts Options) {
	s.chatBox = container.NewVBox()
	s.chatScroll = container.NewVScroll(s.chatBox)
	s.chatScroll.SetMinSize(fyne.NewSize(700, 480))

	s.sideLabel = widget.NewLabel("Hi-agent")
	s.sideLabel.Wrapping = fyne.TextWrapWord
	s.sideLabel.TextStyle = fyne.TextStyle{Monospace: true}
	sideScroll := container.NewVScroll(s.sideLabel)
	sideScroll.SetMinSize(fyne.NewSize(260, 480))

	s.promptLbl = widget.NewLabel(s.prompt)
	s.entry = widget.NewMultiLineEntry()
	s.entry.SetMinRowsVisible(2)
	s.entry.SetPlaceHolder("输入消息，或 /help、/model …")

	s.sendBtn = widget.NewButtonWithIcon("发送", theme.MailSendIcon(), s.submit)
	s.sendBtn.Importance = widget.HighImportance

	var slashBtns []fyne.CanvasObject
	for _, c := range opts.SlashCommands {
		name := c.Name
		btn := widget.NewButton(name, func() {
			s.pushLine(name)
		})
		slashBtns = append(slashBtns, btn)
	}
	slashRow := container.NewHBox(slashBtns...)

	inputRow := container.NewBorder(nil, nil, s.promptLbl, s.sendBtn, s.entry)
	bottom := container.NewVBox(slashRow, inputRow)

	split := container.NewHSplit(s.chatScroll, sideScroll)
	split.Offset = 0.72

	brand := widget.NewLabelWithStyle("Hi-agent", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("简洁对话壳 · 对标 Codex / Cursor")
	hint.Importance = widget.LowImportance
	top := container.NewVBox(brand, hint, widget.NewSeparator())

	s.win.SetContent(container.NewBorder(top, bottom, nil, nil, split))
	s.RefreshStatus()
}

func (s *Shell) submit() {
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	if busy {
		return
	}
	text := s.entry.Text
	s.entry.SetText("")
	s.pushLine(text)
}

func (s *Shell) pushLine(text string) {
	select {
	case <-s.done:
		return
	default:
	}
	select {
	case s.lines <- text:
	case <-s.done:
	}
}

// Run 在调用线程阻塞运行 UI（Windows 上应为主线程）。
func (s *Shell) Run() {
	s.win.ShowAndRun()
	s.signalClosed()
}

// Done 在窗口关闭后关闭。
func (s *Shell) Done() <-chan struct{} { return s.done }

// Lines 用户提交的行。
func (s *Shell) Lines() <-chan string { return s.lines }

func (s *Shell) signalClosed() {
	s.closeOnce.Do(func() { close(s.done) })
}

// Quit 关闭窗口。
func (s *Shell) Quit() {
	s.signalClosed()
	fyne.Do(func() {
		s.win.Close()
		s.app.Quit()
	})
}

func (s *Shell) do(fn func()) {
	fyne.Do(fn)
}

// SetPrompt 更新输入提示。
func (s *Shell) SetPrompt(p string) {
	s.mu.Lock()
	s.prompt = p
	s.mu.Unlock()
	s.do(func() { s.promptLbl.SetText(p) })
}

// SetBusy 生成中禁用发送。
func (s *Shell) SetBusy(v bool) {
	s.mu.Lock()
	s.busy = v
	s.mu.Unlock()
	s.do(func() {
		if v {
			s.sendBtn.Disable()
			s.entry.Disable()
		} else {
			s.sendBtn.Enable()
			s.entry.Enable()
		}
	})
	s.RefreshStatus()
}

func (s *Shell) appendMsg(kind msgKind, text string, open bool) {
	s.mu.Lock()
	if open && kind == msgModel {
		if n := len(s.msgs); n > 0 && s.msgs[n-1].open && s.msgs[n-1].kind == msgModel {
			s.msgs[n-1].text += text
			msgs := append([]chatMsg(nil), s.msgs...)
			s.mu.Unlock()
			s.renderChat(msgs)
			return
		}
	}
	if kind != msgModel {
		if n := len(s.msgs); n > 0 {
			s.msgs[n-1].open = false
		}
	}
	s.msgs = append(s.msgs, chatMsg{kind: kind, text: text, open: open})
	msgs := append([]chatMsg(nil), s.msgs...)
	s.mu.Unlock()
	s.renderChat(msgs)
}

func (s *Shell) EndModel() {
	s.mu.Lock()
	if n := len(s.msgs); n > 0 {
		s.msgs[n-1].open = false
	}
	s.mu.Unlock()
}

func (s *Shell) renderChat(msgs []chatMsg) {
	s.do(func() {
		objs := make([]fyne.CanvasObject, 0, len(msgs))
		for _, m := range msgs {
			objs = append(objs, s.bubble(m))
		}
		s.chatBox.Objects = objs
		s.chatBox.Refresh()
		s.chatScroll.ScrollToBottom()
	})
}

func (s *Shell) bubble(m chatMsg) fyne.CanvasObject {
	prefix := ""
	switch m.kind {
	case msgUser:
		prefix = "You  "
	case msgModel:
		prefix = "Agent  "
	case msgTool:
		prefix = "▸ "
	case msgErr:
		prefix = "! "
	default:
		prefix = "· "
	}
	lbl := widget.NewLabel(prefix + m.text)
	lbl.Wrapping = fyne.TextWrapWord
	switch m.kind {
	case msgUser:
		lbl.TextStyle = fyne.TextStyle{Bold: true}
	case msgTool:
		lbl.Importance = widget.MediumImportance
	case msgErr:
		lbl.Importance = widget.DangerImportance
	case msgSys:
		lbl.Importance = widget.LowImportance
	}
	return lbl
}

// AppendSys / User / Tool / Err / Model 追加消息。
func (s *Shell) AppendSys(t string)   { s.appendMsg(msgSys, t, false) }
func (s *Shell) AppendUser(t string)  { s.appendMsg(msgUser, t, false) }
func (s *Shell) AppendTool(t string)  { s.appendMsg(msgTool, t, false) }
func (s *Shell) AppendErr(t string)   { s.appendMsg(msgErr, t, false) }
func (s *Shell) AppendModel(t string) { s.appendMsg(msgModel, t, true) }

// RefreshStatus 刷新侧栏。
func (s *Shell) RefreshStatus() {
	s.mu.Lock()
	busy := s.busy
	s.mu.Unlock()
	var snap StatusSnapshot
	if s.statusFn != nil {
		snap = s.statusFn()
	}
	snap.Busy = busy
	text := snap.FormatSidebar()
	s.do(func() { s.sideLabel.SetText(text) })
}
