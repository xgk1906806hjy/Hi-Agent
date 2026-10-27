package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// FactoryFunc 根据 Options 构造一种 UI。
type FactoryFunc func(opts Options) (UI, error)

var (
	mu        sync.RWMutex
	factories = map[Kind]FactoryFunc{}
)

func init() {
	// plain / tui 在各自文件的 init 中 Register。
	// gui 由 internal/gui（desktop 构建标签）在 init 中覆盖本占位；
	// web 由 internal/webui 在 init 中覆盖本占位。
	Register(KindGUI, func(Options) (UI, error) {
		return nil, fmt.Errorf("当前程序是终端版，不含桌面界面；请用 scripts/build-gui 构建桌面版")
	})
	Register(KindWeb, notImplemented(KindWeb))
}

func notImplemented(k Kind) FactoryFunc {
	return func(Options) (UI, error) {
		return nil, fmt.Errorf("UI %q 尚未实现；可用：plain、tui、gui（或自行 ui.Register）", k)
	}
}

// Register 注册或覆盖一种 UI 工厂（可插拔扩展点）。
func Register(kind Kind, fn FactoryFunc) {
	if fn == nil {
		return
	}
	kind = normalizeKind(kind)
	mu.Lock()
	factories[kind] = fn
	mu.Unlock()
}

// Create 按 Kind 创建 UI；未知 Kind 返回错误。
func Create(kind Kind, opts Options) (UI, error) {
	kind = normalizeKind(kind)
	mu.RLock()
	fn, ok := factories[kind]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("未注册的 UI 类型 %q（已注册：%s）", kind, strings.Join(registeredKinds(), ", "))
	}
	return fn(opts)
}

// MustCreate 同 Create，失败时 fallback 到 plain（再失败则 panic）。
func MustCreate(kind Kind, opts Options) UI {
	u, err := Create(kind, opts)
	if err == nil {
		return u
	}
	if kind != KindPlain {
		if u2, err2 := Create(KindPlain, opts); err2 == nil {
			return u2
		}
	}
	panic("ui.MustCreate: " + err.Error())
}

// ResolveKind 从环境变量解析 UI 类型。
//
// 优先级：GEEKAGENT_UI / HI_AGENT_UI → GEEKAGENT_TUI=0 则 plain → 默认 tui。
func ResolveKind() Kind {
	if v := strings.TrimSpace(os.Getenv("GEEKAGENT_UI")); v != "" {
		return normalizeKind(Kind(v))
	}
	if v := strings.TrimSpace(os.Getenv("HI_AGENT_UI")); v != "" {
		return normalizeKind(Kind(v))
	}
	if strings.TrimSpace(os.Getenv("GEEKAGENT_TUI")) == "0" {
		return KindPlain
	}
	return KindTUI
}

// ListKinds 返回已注册 Kind（排序稳定）。
func ListKinds() []Kind {
	return registeredKindList()
}

func normalizeKind(k Kind) Kind {
	s := strings.ToLower(strings.TrimSpace(string(k)))
	switch s {
	case "text", "cli", "terminal":
		return KindPlain
	case "fullscreen", "termui":
		return KindTUI
	default:
		return Kind(s)
	}
}

func registeredKinds() []string {
	list := registeredKindList()
	out := make([]string, len(list))
	for i, k := range list {
		out[i] = string(k)
	}
	return out
}

func registeredKindList() []Kind {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Kind, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	// 简单插入排序，避免额外依赖
	for i := 1; i < len(out); i++ {
		j := i
		for j > 0 && out[j-1] > out[j] {
			out[j-1], out[j] = out[j], out[j-1]
			j--
		}
	}
	return out
}
