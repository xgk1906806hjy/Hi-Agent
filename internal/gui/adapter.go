package gui

import (
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
)

// Emitter 把事件推给前端；Wails 实现见 app.go，单测用假实现。
type Emitter interface {
	Emit(event string, payload any)
}

// Config 创建 Adapter 的参数。Run / Quit 由桌面层注入，单测可留空。
type Config struct {
	StatusFn  func() ui.Status
	NavFn     func() ui.Nav
	HistoryFn func() []ui.Turn
	Slash     []ui.SlashCmd
	Run       func() // 阻塞直到窗口关闭
	Quit      func() // 请求关闭窗口
}

type queued struct {
	event   string
	payload any
}

// Adapter 实现 ui.UI、ui.DesktopRunner、ui.Canceler。
// UI 方法由 REPL goroutine 调用；Submit/Stop/Answer/Attach 由前端线程调用。
type Adapter struct {
	cfg Config

	mu      sync.Mutex
	em      Emitter
	queue   []queued
	pending map[string]chan int
	seq     int
	cancel  func()
	snap    Snapshot
	viewKey string
	started bool
	idle    bool

	lines     chan string
	done      chan struct{}
	closeOnce sync.Once
}

// NewAdapter 创建适配器；在 Attach 之前发出的事件会排队。
func NewAdapter(cfg Config) *Adapter {
	slash := make([]SlashPayload, 0, len(cfg.Slash))
	for _, c := range cfg.Slash {
		slash = append(slash, SlashPayload{Name: c.Name, Desc: c.Desc})
	}
	return &Adapter{
		cfg:     cfg,
		pending: map[string]chan int{},
		snap:    Snapshot{Nav: NavPayload{Projects: []NavProjectPayload{}}, Slash: slash},
		lines:   make(chan string, 16),
		done:    make(chan struct{}),
	}
}

// —— 前端侧入口 ——

// Attach 前端就绪后接入发射器，并按顺序冲刷排队事件。
func (a *Adapter) Attach(em Emitter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.em = em
	for _, q := range a.queue {
		em.Emit(q.event, q.payload)
	}
	a.queue = nil
}

// Submit 送入一行输入（消息或斜杠命令）；窗口已关闭时丢弃。
func (a *Adapter) Submit(text string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	select {
	case a.lines <- text:
	case <-a.done:
	}
}

// Stop 中断当前回复（无回复时无操作）。
func (a *Adapter) Stop() {
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Answer 回复对话框；未知或已回答的 id 忽略。
func (a *Adapter) Answer(id string, value int) {
	a.mu.Lock()
	ch, ok := a.pending[id]
	delete(a.pending, id)
	a.mu.Unlock()
	if ok {
		ch <- value
	}
}

// Snapshot 返回最近一次由 REPL 线程刷新的状态（不在前端线程读引擎数据）。
func (a *Adapter) Snapshot() Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.snap
}

// Closed 窗口关闭时由桌面层调用：解除 ReadLine 与对话框阻塞。
func (a *Adapter) Closed() {
	a.closeOnce.Do(func() {
		close(a.done)
		a.mu.Lock()
		for id, ch := range a.pending {
			delete(a.pending, id)
			ch <- -1
		}
		a.mu.Unlock()
	})
}

// —— ui.UI ——

func (a *Adapter) Info(s string)     { a.start(); a.emit(EvInfo, TextPayload{s}) }
func (a *Adapter) Notice(s string)   { a.start(); a.emit(EvNotice, TextPayload{strings.TrimSpace(s)}) }
func (a *Adapter) Error(s string)    { a.start(); a.emit(EvError, TextPayload{s}) }
func (a *Adapter) UserEcho(s string) { a.start(); a.emit(EvUser, TextPayload{s}) }
func (a *Adapter) Progress(s string) { a.emit(EvProgress, TextPayload{strings.Trim(s, "\n")}) }
func (a *Adapter) Model(s string)    { a.emit(EvDelta, TextPayload{s}) }

func (a *Adapter) BeginReply() {
	a.emit(EvTurnBeg, nil)
	a.refreshStatus(true)
}

func (a *Adapter) EndReply() {
	a.emit(EvTurnEnd, nil)
	a.refreshStatus(false)
	a.refreshNav()
}

func (a *Adapter) Refresh() {
	a.refreshStatus(a.Snapshot().Status.Busy)
	a.refreshNav()
}

// RefreshNav 立即按 NavFn 推送侧栏（添加项目未走 /project 时也要用）。
func (a *Adapter) RefreshNav() { a.refreshNav() }

func (a *Adapter) ReadLine(prompt string) (string, error) {
	a.start()
	a.syncView()
	a.mu.Lock()
	a.snap.Prompt = prompt
	a.idle = prompt == ui.MainPrompt
	a.mu.Unlock()
	a.emit(EvPrompt, PromptPayload{prompt})
	select {
	case line := <-a.lines:
		a.mu.Lock()
		a.snap.Prompt = ""
		a.idle = false
		a.mu.Unlock()
		a.emit(EvPrompt, PromptPayload{""})
		return line, nil
	case <-a.done:
		return "", io.EOF
	}
}

// Idle 为 true 表示 REPL 正阻塞在主提示等待新输入：此时才允许侧栏切项目/会话，
// 也只有此时从前端线程读引擎数据（NavFn）不会与 REPL 竞争。
func (a *Adapter) Idle() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.idle
}

func (a *Adapter) Confirm(prompt string) bool {
	return a.ask(EvConfirm, func(id string) any { return ConfirmPayload{ID: id, Prompt: prompt} }) == 1
}

func (a *Adapter) Select(title string, items []string, selected int) (int, error) {
	if len(items) == 0 {
		return -1, errNoItems
	}
	v := a.ask(EvSelect, func(id string) any {
		return SelectPayload{ID: id, Title: title, Items: items, Selected: selected}
	})
	if v < 0 || v >= len(items) {
		return -1, nil
	}
	return v, nil
}

// Close 由 REPL 退出时调用：请求关窗并解除阻塞。
func (a *Adapter) Close() {
	a.Closed()
	if a.cfg.Quit != nil {
		a.cfg.Quit()
	}
}

// Run 实现 ui.DesktopRunner。
func (a *Adapter) Run() {
	if a.cfg.Run != nil {
		a.cfg.Run()
	}
	a.Closed()
}

// SetCancel 实现 ui.Canceler。
func (a *Adapter) SetCancel(cancel func()) {
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
}

// —— 内部 ——

type adapterError string

func (e adapterError) Error() string { return string(e) }

const errNoItems = adapterError("没有可选项")

func (a *Adapter) emit(event string, payload any) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.em == nil {
		a.queue = append(a.queue, queued{event, payload})
		return
	}
	a.em.Emit(event, payload)
}

// ask 发出对话框请求并阻塞等待回答；窗口关闭返回 -1。
func (a *Adapter) ask(event string, build func(id string) any) int {
	ch := make(chan int, 1)
	a.mu.Lock()
	a.seq++
	id := "d" + strconv.Itoa(a.seq)
	a.pending[id] = ch
	a.mu.Unlock()
	select {
	case <-a.done:
		a.mu.Lock()
		delete(a.pending, id)
		a.mu.Unlock()
		return -1
	default:
	}
	a.emit(event, build(id))
	select {
	case v := <-ch:
		return v
	case <-a.done:
		a.emit(EvDismiss, DismissPayload{ID: id})
		return -1
	}
}

// start 首次输出前记录当前视图并补发已有历史（让 banner 出现在历史之后）。
func (a *Adapter) start() {
	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return
	}
	a.started = true
	a.mu.Unlock()
	a.refreshStatus(false)
	a.refreshNav()
	a.mu.Lock()
	a.viewKey = viewKey(a.snap.Status)
	a.mu.Unlock()
	if turns := a.history(); len(turns) > 0 {
		a.emit(EvReset, ResetPayload{Turns: turns})
	}
}

// syncView 刷新状态与导航；项目或会话变化时重绘对话流。
func (a *Adapter) syncView() {
	a.refreshStatus(false)
	a.refreshNav()
	a.mu.Lock()
	key := viewKey(a.snap.Status)
	changed := key != a.viewKey
	a.viewKey = key
	a.mu.Unlock()
	if changed {
		a.emit(EvReset, ResetPayload{Turns: a.history()})
	}
}

func viewKey(s StatusPayload) string { return s.WorkDir + "\x00" + s.Session }

func (a *Adapter) history() []TurnPayload {
	if a.cfg.HistoryFn == nil {
		return []TurnPayload{}
	}
	return toTurns(a.cfg.HistoryFn())
}

func (a *Adapter) refreshStatus(busy bool) {
	var st StatusPayload
	if a.cfg.StatusFn != nil {
		st = toStatus(a.cfg.StatusFn())
	}
	st.Busy = busy
	a.mu.Lock()
	a.snap.Status = st
	a.mu.Unlock()
	a.emit(EvStatus, st)
}

func (a *Adapter) refreshNav() {
	nav := NavPayload{Projects: []NavProjectPayload{}}
	if a.cfg.NavFn != nil {
		nav = toNav(a.cfg.NavFn())
	}
	a.mu.Lock()
	a.snap.Nav = nav
	a.mu.Unlock()
	a.emit(EvNav, nav)
}

var (
	_ ui.UI            = (*Adapter)(nil)
	_ ui.DesktopRunner = (*Adapter)(nil)
	_ ui.Canceler      = (*Adapter)(nil)
)
