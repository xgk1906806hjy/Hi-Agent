package gui

import (
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
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
	NavFn         func() NavSnapshot
	SlashCommands []SlashItem
}

// Shell 是 Fyne 桌面对话壳（Codex 风格简洁窗）。
type Shell struct {
	app fyne.App
	win fyne.Window

	chatBox    *fyne.Container
	chatScroll *container.Scroll
	navBox     *fyne.Container
	navSig     string
	navFn      func() NavSnapshot
	sideLabel  *widget.Label
	titleLbl   *widget.Label
	entry      *widget.Entry
	sendBtn    *widget.Button
	stopBtn    *widget.Button
	statusFn   func() StatusSnapshot

	mu     sync.Mutex
	msgs   []chatMsg
	busy   bool
	prompt string
	cancel func()

	lines     chan string
	done      chan struct{}
	closeOnce sync.Once
}

// New 创建窗口（尚未 Run）。
func New(opts Options) *Shell {
	a := app.NewWithID("com.hi-agent.gui")
	a.Settings().SetTheme(&codexTheme{})
	s := &Shell{
		app:      a,
		lines:    make(chan string, 8),
		done:     make(chan struct{}),
		statusFn: opts.StatusFn,
		navFn:    opts.NavFn,
		prompt:   "You › ",
	}
	title := opts.Title
	if title == "" {
		title = "Hi-agent"
	}
	s.win = s.app.NewWindow(title)
	s.win.Resize(fyne.NewSize(1180, 720))
	s.build(opts)
	s.win.SetCloseIntercept(func() {
		s.signalClosed()
		s.win.Close()
	})
	return s
}

func (s *Shell) build(opts Options) {
	s.chatBox = container.NewVBox()
	s.chatScroll = container.NewVScroll(container.NewPadded(s.chatBox))
	s.chatScroll.SetMinSize(fyne.NewSize(680, 460))
	chatBackdrop := canvas.NewRectangle(chatBG)
	chatPane := container.NewStack(chatBackdrop, s.chatScroll)

	s.sideLabel = widget.NewLabel("")
	s.sideLabel.Wrapping = fyne.TextWrapWord
	s.sideLabel.Importance = widget.LowImportance
	sideInner := container.NewPadded(s.sideLabel)
	sideBG := canvas.NewRectangle(sidePanelBG)
	sideScroll := container.NewVScroll(sideInner)
	sideScroll.SetMinSize(fyne.NewSize(220, 460))
	sidePanel := container.NewStack(sideBG, sideScroll)

	s.entry = widget.NewMultiLineEntry()
	s.entry.SetMinRowsVisible(2)
	s.entry.SetPlaceHolder("发消息，或输入 /model、/help …")

	s.sendBtn = widget.NewButtonWithIcon("", theme.MailSendIcon(), s.submit)
	s.sendBtn.Importance = widget.HighImportance
	s.stopBtn = widget.NewButtonWithIcon("", theme.MediaStopIcon(), s.stop)
	s.stopBtn.Importance = widget.LowImportance
	s.stopBtn.Hide()

	slashRow := container.NewHBox(
		newTapRow("模型", false, func() { s.pushLine("/model") }),
		newTapRow("帮助", false, func() { s.pushLine("/help") }),
	)

	actions := container.NewHBox(s.stopBtn, s.sendBtn)
	inputRow := container.NewBorder(nil, nil, nil, actions, s.entry)
	composerBG := canvas.NewRectangle(color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF})
	composerBG.CornerRadius = 16
	composerBG.StrokeColor = hairline
	composerBG.StrokeWidth = 1
	composer := container.NewStack(composerBG, container.NewPadded(inputRow))
	bottomPad := container.NewPadded(container.NewVBox(slashRow, composer))

	s.navBox = container.NewVBox()
	navScroll := container.NewVScroll(container.NewPadded(s.navBox))
	navScroll.SetMinSize(fyne.NewSize(200, 460))
	navBG := canvas.NewRectangle(sidePanelBG)
	navPanel := container.NewStack(navBG, navScroll)

	mid := container.NewHSplit(chatPane, sidePanel)
	mid.Offset = 0.74
	split := container.NewHSplit(navPanel, mid)
	split.Offset = 0.22

	s.titleLbl = widget.NewLabelWithStyle("Hi-agent", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	top := container.NewVBox(
		container.NewPadded(s.titleLbl),
		widget.NewSeparator(),
	)

	s.win.SetContent(container.NewBorder(top, bottomPad, nil, nil, split))
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

func (s *Shell) stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// SetCancel 设置「停止」按钮调用的取消函数；nil 表示当前无可中断的回复。
func (s *Shell) SetCancel(cancel func()) {
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	s.do(func() {
		if cancel != nil {
			s.stopBtn.Show()
			s.stopBtn.Enable()
		} else {
			s.stopBtn.Hide()
		}
	})
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

// SetPrompt 更新输入提示（GUI 仅保留内部状态，不在输入区显示前缀）。
func (s *Shell) SetPrompt(p string) {
	s.mu.Lock()
	s.prompt = p
	s.mu.Unlock()
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
			objs = append(objs, bubble(m))
		}
		s.chatBox.Objects = objs
		s.chatBox.Refresh()
		s.chatScroll.ScrollToBottom()
	})
}

// AppendSys / User / Tool / Err / Model 追加消息。
func (s *Shell) AppendSys(t string)   { s.appendMsg(msgSys, t, false) }
func (s *Shell) AppendUser(t string)  { s.appendMsg(msgUser, t, false) }
func (s *Shell) AppendTool(t string)  { s.appendMsg(msgTool, t, false) }
func (s *Shell) AppendErr(t string)   { s.appendMsg(msgErr, t, false) }
func (s *Shell) AppendModel(t string) { s.appendMsg(msgModel, t, true) }

// RefreshStatus 刷新侧栏与顶栏模型名。
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
	title := "Hi-agent"
	if m := snap.ModelShort(); m != "" {
		title = "Hi-agent · " + m
	}
	var nav NavSnapshot
	if s.navFn != nil {
		nav = s.navFn()
	}
	s.do(func() {
		s.sideLabel.SetText(text)
		s.titleLbl.SetText(title)
		s.renderNav(nav)
	})
}
