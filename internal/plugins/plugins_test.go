package plugins

import (
	"errors"
	"strings"
	"testing"

	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
)

type fakeHost struct {
	workDir string
	info    []string
	tools   map[string]tools.Tool
}

func newFakeHost() *fakeHost {
	return &fakeHost{workDir: "E:/work", tools: map[string]tools.Tool{}}
}

func (h *fakeHost) WorkDir() string { return h.workDir }
func (h *fakeHost) Info(s string)   { h.info = append(h.info, s) }
func (h *fakeHost) Error(s string)  { h.info = append(h.info, "ERR:"+s) }
func (h *fakeHost) Confirm(string) bool {
	return true
}
func (h *fakeHost) RegisterTool(t tools.Tool) error {
	if _, ok := h.tools[t.Name]; ok {
		return errors.New("duplicate")
	}
	h.tools[t.Name] = t
	return nil
}
func (h *fakeHost) UnregisterTool(name string) { delete(h.tools, name) }

func resetRegistry() {
	registryMu.Lock()
	registry = map[string]Plugin{}
	registryMu.Unlock()
}

func TestManagerLifecycleCommandsAndTools(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)

	host := newFakeHost()
	var started, exited bool
	Register(Plugin{
		Name:        "tester",
		Description: "test plugin",
		Commands: []Command{{
			Name: "hello",
			Run: func(ctx *Context, args []string) error {
				ctx.Info(ctx.Plugin() + ":" + ctx.WorkDir() + ":" + strings.Join(args, ","))
				return nil
			},
		}},
		Tools: []tools.Tool{{
			Name: "tester_tool",
			Run:  func(string) (string, error) { return "ok", nil },
		}},
		OnStart: func(ctx *Context) error {
			started = true
			return nil
		},
		OnExit: func(ctx *Context) error {
			exited = true
			return nil
		},
	})

	m := NewManager(host)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("OnStart not called")
	}
	if _, ok := host.tools["tester_tool"]; !ok {
		t.Fatal("plugin tool not registered")
	}
	if handled, err := m.Run("/hello", []string{"a", "b"}); !handled || err != nil {
		t.Fatalf("run: handled=%v err=%v", handled, err)
	}
	if got := host.info[len(host.info)-1]; got != "tester:E:/work:a,b" {
		t.Fatalf("info = %q", got)
	}
	if handled, err := m.Run("missing", nil); handled || err != nil {
		t.Fatalf("missing command: handled=%v err=%v", handled, err)
	}
	if lines := m.Lines(); len(lines) != 1 || !strings.Contains(lines[0], "已启动") {
		t.Fatalf("lines = %#v", lines)
	}

	m.Close()
	if !exited {
		t.Fatal("OnExit not called")
	}
	if _, ok := host.tools["tester_tool"]; ok {
		t.Fatal("plugin tool not unregistered")
	}
	if handled, err := m.Run("hello", nil); !handled || err == nil {
		t.Fatalf("run after close: handled=%v err=%v", handled, err)
	}
}

func TestManagerReload(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)

	host := newFakeHost()
	starts, exits := 0, 0
	Register(Plugin{
		Name:    "reload",
		OnStart: func(*Context) error { starts++; return nil },
		OnExit:  func(*Context) error { exits++; return nil },
	})
	m := NewManager(host)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if err := m.Reload(); err != nil {
		t.Fatal(err)
	}
	if starts != 2 || exits != 1 {
		t.Fatalf("starts=%d exits=%d", starts, exits)
	}
	m.Close()
}

func TestRegisterRejectsDuplicateCommand(t *testing.T) {
	resetRegistry()
	t.Cleanup(resetRegistry)
	Register(Plugin{Name: "a", Commands: []Command{{Name: "same", Run: func(*Context, []string) error { return nil }}}})

	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	Register(Plugin{Name: "b", Commands: []Command{{Name: "same", Run: func(*Context, []string) error { return nil }}}})
}
