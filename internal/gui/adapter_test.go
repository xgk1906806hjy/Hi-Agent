package gui

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
)

type fakeEmitter struct {
	mu     sync.Mutex
	events []queued
	ch     chan queued
}

func newFake() *fakeEmitter { return &fakeEmitter{ch: make(chan queued, 64)} }

func (f *fakeEmitter) Emit(event string, payload any) {
	f.mu.Lock()
	f.events = append(f.events, queued{event, payload})
	f.mu.Unlock()
	f.ch <- queued{event, payload}
}

func (f *fakeEmitter) names() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.events))
	for i, e := range f.events {
		out[i] = e.event
	}
	return out
}

// waitFor 等到出现指定事件并返回其载荷。
func (f *fakeEmitter) waitFor(t *testing.T, event string) any {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case e := <-f.ch:
			if e.event == event {
				return e.payload
			}
		case <-deadline:
			t.Fatalf("timeout waiting for %s; got %v", event, f.names())
		}
	}
}

func TestQueuedBeforeAttachKeepsOrder(t *testing.T) {
	a := NewAdapter(Config{})
	a.Info("banner")
	a.Model("hi")
	f := newFake()
	a.Attach(f)
	got := f.names()
	iInfo, iDelta := -1, -1
	for i, n := range got {
		switch n {
		case EvInfo:
			iInfo = i
		case EvDelta:
			iDelta = i
		}
	}
	if iInfo < 0 || iDelta < 0 || iInfo > iDelta {
		t.Fatalf("order %v", got)
	}
}

func TestHistoryEmittedBeforeBanner(t *testing.T) {
	a := NewAdapter(Config{HistoryFn: func() []ui.Turn { return []ui.Turn{{Role: "user", Text: "old"}} }})
	f := newFake()
	a.Attach(f)
	a.Info("banner")
	got := f.names()
	iReset, iInfo := -1, -1
	for i, n := range got {
		if n == EvReset && iReset < 0 {
			iReset = i
		}
		if n == EvInfo {
			iInfo = i
		}
	}
	if iReset < 0 || iReset > iInfo {
		t.Fatalf("order %v", got)
	}
}

func TestReadLineSubmitAndEOF(t *testing.T) {
	a := NewAdapter(Config{})
	a.Attach(newFake())
	go a.Submit("  hello ")
	line, err := a.ReadLine("You › ")
	if err != nil || line != "hello" {
		t.Fatalf("%q %v", line, err)
	}
	go a.Closed()
	if _, err := a.ReadLine(""); !errors.Is(err, io.EOF) {
		t.Fatalf("want EOF, got %v", err)
	}
}

func TestConfirmAndSelectRoundTrip(t *testing.T) {
	a := NewAdapter(Config{})
	f := newFake()
	a.Attach(f)

	res := make(chan bool, 1)
	go func() { res <- a.Confirm("ok?") }()
	p := f.waitFor(t, EvConfirm).(ConfirmPayload)
	a.Answer(p.ID, 1)
	if !<-res {
		t.Fatal("confirm should be true")
	}

	sel := make(chan int, 1)
	go func() { i, _ := a.Select("pick", []string{"a", "b"}, 0); sel <- i }()
	sp := f.waitFor(t, EvSelect).(SelectPayload)
	a.Answer(sp.ID, 1)
	if got := <-sel; got != 1 {
		t.Fatalf("select %d", got)
	}

	go func() { i, _ := a.Select("pick", []string{"a"}, 0); sel <- i }()
	sp = f.waitFor(t, EvSelect).(SelectPayload)
	a.Answer(sp.ID, -1)
	if got := <-sel; got != -1 {
		t.Fatalf("cancel %d", got)
	}
}

func TestCloseUnblocksDialog(t *testing.T) {
	a := NewAdapter(Config{})
	f := newFake()
	a.Attach(f)
	res := make(chan bool, 1)
	go func() { res <- a.Confirm("ok?") }()
	f.waitFor(t, EvConfirm)
	a.Close()
	if <-res {
		t.Fatal("closed confirm should be false")
	}
}

func TestStopCallsCancel(t *testing.T) {
	a := NewAdapter(Config{})
	called := false
	a.SetCancel(func() { called = true })
	a.Stop()
	if !called {
		t.Fatal("cancel not called")
	}
	a.SetCancel(nil)
	a.Stop()
}

func TestRefreshEmitsNav(t *testing.T) {
	a := NewAdapter(Config{
		NavFn: func() ui.Nav {
			return ui.Nav{Projects: []ui.NavProject{{Name: "p", Path: "/p", Current: true}}}
		},
	})
	f := newFake()
	a.Attach(f)
	a.Refresh()
	p := f.waitFor(t, EvNav).(NavPayload)
	if len(p.Projects) != 1 || p.Projects[0].Path != "/p" {
		t.Fatalf("%+v", p)
	}
	a.RefreshNav()
	p2 := f.waitFor(t, EvNav).(NavPayload)
	if len(p2.Projects) != 1 {
		t.Fatalf("%+v", p2)
	}
}

func TestIdleOnlyAtMainPrompt(t *testing.T) {
	a := NewAdapter(Config{})
	f := newFake()
	a.Attach(f)
	if a.Idle() {
		t.Fatal("idle before ReadLine")
	}

	got := make(chan string, 1)
	go func() { l, _ := a.ReadLine(ui.MainPrompt); got <- l }()
	f.waitFor(t, EvPrompt)
	if !a.Idle() || a.Snapshot().Prompt != ui.MainPrompt {
		t.Fatal("want idle at main prompt")
	}
	a.Submit("hi")
	<-got
	if a.Idle() || a.Snapshot().Prompt != "" {
		t.Fatal("busy after submit")
	}

	go func() { l, _ := a.ReadLine("API Key: "); got <- l }()
	f.waitFor(t, EvPrompt)
	if a.Idle() {
		t.Fatal("sub-prompt must not be idle")
	}
	a.Submit("k")
	<-got
}

func TestSnapshotCarriesSlashAndNavMeta(t *testing.T) {
	at := time.UnixMilli(1700000000000)
	a := NewAdapter(Config{
		Slash: []ui.SlashCmd{{Name: "/help", Desc: "帮助"}},
		NavFn: func() ui.Nav {
			return ui.Nav{Projects: []ui.NavProject{{Path: "/p", Sessions: []ui.NavSession{
				{ID: "s", Title: "hi", UpdatedAt: at}, {ID: "new"},
			}}}}
		},
	})
	a.Attach(newFake())
	a.RefreshNav()
	s := a.Snapshot()
	if len(s.Slash) != 1 || s.Slash[0].Name != "/help" {
		t.Fatalf("slash %+v", s.Slash)
	}
	ss := s.Nav.Projects[0].Sessions
	if ss[0].Title != "hi" || ss[0].UpdatedAt != 1700000000000 || ss[1].UpdatedAt != 0 {
		t.Fatalf("nav %+v", ss)
	}
}

func TestSessionChangeResetsThread(t *testing.T) {
	session := "a"
	a := NewAdapter(Config{
		StatusFn: func() ui.Status { return ui.Status{Session: session, WorkDir: "w"} },
	})
	f := newFake()
	a.Attach(f)
	go a.Submit("x")
	_, _ = a.ReadLine("")
	session = "b"
	go a.Submit("y")
	_, _ = a.ReadLine("")
	resets := 0
	for _, n := range f.names() {
		if n == EvReset {
			resets++
		}
	}
	if resets != 1 {
		t.Fatalf("resets=%d events=%v", resets, f.names())
	}
}
