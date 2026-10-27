// Package plugins 提供编译期插件注册、生命周期与斜杠命令分派。
//
// Go 没有运行时插件加载：每个插件放在仓库根目录的 plugins/<名称>/ 下，
// 在 init() 中调用 Register；plugins/all 负责汇总空白导入。
// REPL 只通过 Manager 启动/停止插件并转发未命中的斜杠命令。
package plugins

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
)

// Host 由 REPL 实现，插件只能经 Context 访问宿主能力。
// 回调在 REPL goroutine 上执行，实现方需保证 UI 方法可重入。
type Host interface {
	WorkDir() string
	Info(string)
	Error(string)
	Confirm(string) bool
	RegisterTool(tools.Tool) error
	UnregisterTool(string)
}

// PluginHost 是 Host 的别名。
type PluginHost = Host

// Command 一个插件斜杠命令；Name 不带前导斜杠（如 echo）。
type Command struct {
	Name        string
	Description string
	Run         func(ctx *Context, args []string) error
}

// Plugin 编译期插件声明。
//
// OnStart / OnExit 可留空；Tools 会在 Start 成功后注册，在 Close 时注销。
// 插件若在 OnStart 中通过 Context.RegisterTool 注册工具，也会由 Manager 跟踪注销。
type Plugin struct {
	Name        string
	Description string
	Commands    []Command
	Tools       []tools.Tool
	OnStart     func(*Context) error
	OnExit      func(*Context) error
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Plugin{}
)

// Register 注册插件；重复插件名或跨插件重复命令名会 panic（启动期失败，避免静默覆盖）。
func Register(p Plugin) {
	p.Name = normalizeName(p.Name)
	if p.Name == "" {
		panic("plugins: empty plugin name")
	}
	seenCommands := map[string]bool{}
	for i := range p.Commands {
		p.Commands[i].Name = normalizeName(p.Commands[i].Name)
		if p.Commands[i].Name == "" {
			panic("plugins: empty command name in " + p.Name)
		}
		if seenCommands[p.Commands[i].Name] {
			panic("plugins: duplicate command name in " + p.Name + ": " + p.Commands[i].Name)
		}
		seenCommands[p.Commands[i].Name] = true
		if p.Commands[i].Run == nil {
			panic("plugins: nil command run in " + p.Name + "/" + p.Commands[i].Name)
		}
	}

	registryMu.Lock()
	defer registryMu.Unlock()
	if _, ok := registry[p.Name]; ok {
		panic("plugins: duplicate plugin name: " + p.Name)
	}
	for _, existing := range registry {
		for _, c := range existing.Commands {
			for _, nc := range p.Commands {
				if c.Name == nc.Name {
					panic("plugins: duplicate command name: " + c.Name)
				}
			}
		}
	}
	registry[p.Name] = p
}

// Registered 返回按插件名排序的插件声明。
func Registered() []Plugin {
	registryMu.RLock()
	out := make([]Plugin, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	registryMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Commands 返回全部插件命令，按命令名排序（供 /help 与斜杠菜单）。
func Commands() []Command {
	var out []Command
	for _, p := range Registered() {
		out = append(out, p.Commands...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// CommandLines 返回可直接打印的命令说明。
func CommandLines() []string {
	cmds := Commands()
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		line := "/" + c.Name
		if c.Description != "" {
			line += " — " + c.Description
		}
		out = append(out, line)
	}
	return out
}

// Context 插件生命周期与命令的受限宿主视图。
type Context struct {
	plugin string
	host   Host
	track  func(string)
}

// PluginContext 是 Context 的别名，保留原 TS 方案中的命名，便于插件作者理解。
type PluginContext = Context

// Plugin 返回当前插件名。
func (c *Context) Plugin() string {
	if c == nil {
		return ""
	}
	return c.plugin
}

// WorkDir 返回 REPL 当前项目目录；/project 切换后会跟随变化。
func (c *Context) WorkDir() string {
	if c == nil || c.host == nil {
		return ""
	}
	return c.host.WorkDir()
}

// Info 输出一条系统信息。
func (c *Context) Info(s string) {
	if c != nil && c.host != nil {
		c.host.Info(s)
	}
}

// Error 输出一条错误。
func (c *Context) Error(s string) {
	if c != nil && c.host != nil {
		c.host.Error(s)
	}
}

// Confirm 请求用户确认。
func (c *Context) Confirm(prompt string) bool {
	if c == nil || c.host == nil {
		return false
	}
	return c.host.Confirm(prompt)
}

// RegisterTool 注册一个插件工具，并由 Manager 在插件停止时自动注销。
func (c *Context) RegisterTool(t tools.Tool) error {
	if c == nil || c.host == nil {
		return errors.New("插件上下文未初始化")
	}
	if err := c.host.RegisterTool(t); err != nil {
		return err
	}
	if c.track != nil {
		c.track(t.Name)
	}
	return nil
}

// Manager 负责一次进程内的插件生命周期与命令分派。
type Manager struct {
	host Host

	mu        sync.Mutex
	started   bool
	active    map[string]bool
	failed    map[string]string
	toolNames map[string][]string
}

// NewManager 创建插件管理器；host 为空时插件仍可注册，但会启动失败。
func NewManager(host Host) *Manager {
	return &Manager{
		host:      host,
		active:    map[string]bool{},
		failed:    map[string]string{},
		toolNames: map[string][]string{},
	}
}

// Start 按插件名顺序执行 onStart 并注册插件工具。重复调用无操作。
func (m *Manager) Start() error {
	if m == nil {
		return nil
	}
	if m.host == nil {
		return errors.New("插件宿主未初始化")
	}
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return nil
	}
	m.started = true
	m.active = map[string]bool{}
	m.failed = map[string]string{}
	m.toolNames = map[string][]string{}
	m.mu.Unlock()

	var errs []string
	for _, p := range Registered() {
		m.mu.Lock()
		m.active[p.Name] = true
		m.mu.Unlock()

		ctx := m.context(p.Name)
		if p.OnStart != nil {
			if err := invoke(func() error { return p.OnStart(ctx) }); err != nil {
				m.fail(p.Name, err)
				errs = append(errs, p.Name+"："+err.Error())
				continue
			}
		}
		for _, t := range p.Tools {
			if err := m.registerTool(p.Name, t); err != nil {
				m.fail(p.Name, err)
				errs = append(errs, p.Name+"："+err.Error())
				break
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "；"))
	}
	return nil
}

// Close 逆序执行 onExit 并注销插件工具。重复调用无操作。
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	m.started = false
	active := m.active
	toolNames := m.toolNames
	m.active = map[string]bool{}
	m.toolNames = map[string][]string{}
	m.mu.Unlock()

	plugins := Registered()
	for i := len(plugins) - 1; i >= 0; i-- {
		p := plugins[i]
		if !active[p.Name] {
			continue
		}
		if p.OnExit != nil {
			ctx := m.context(p.Name)
			_ = invoke(func() error { return p.OnExit(ctx) })
		}
		for _, name := range toolNames[p.Name] {
			if m.host != nil {
				m.host.UnregisterTool(name)
			}
		}
	}
}

// Reload 重新执行全部插件的 onExit / onStart，用于 /plugins reload。
func (m *Manager) Reload() error {
	m.Close()
	return m.Start()
}

// Run 执行插件命令；未命中返回 handled=false，交给 REPL 继续处理。
func (m *Manager) Run(name string, args []string) (handled bool, err error) {
	name = normalizeName(name)
	p, cmd, ok := findCommand(name)
	if !ok {
		return false, nil
	}
	if m == nil {
		return true, errors.New("插件管理器未初始化")
	}
	m.mu.Lock()
	started := m.started
	failed := m.failed[p.Name]
	m.mu.Unlock()
	if failed != "" {
		return true, fmt.Errorf("插件 %s 启动失败：%s", p.Name, failed)
	}
	if !started {
		return true, fmt.Errorf("插件 %s 尚未启动", p.Name)
	}
	ctx := m.context(p.Name)
	return true, invoke(func() error { return cmd.Run(ctx, args) })
}

// Lines 返回 /plugins 展示文本。
func (m *Manager) Lines() []string {
	list := Registered()
	if len(list) == 0 {
		return []string{"未注册插件。插件放在 plugins/<名称>/，由 plugins/all 编译期导入。"}
	}
	m.mu.Lock()
	started := m.started
	failed := make(map[string]string, len(m.failed))
	for k, v := range m.failed {
		failed[k] = v
	}
	toolNames := make(map[string][]string, len(m.toolNames))
	for k, v := range m.toolNames {
		toolNames[k] = append([]string(nil), v...)
	}
	m.mu.Unlock()

	out := make([]string, 0, len(list))
	for _, p := range list {
		status := "未启动"
		if started {
			status = "已启动"
		}
		if msg := failed[p.Name]; msg != "" {
			status = "启动失败：" + msg
		}
		line := "• " + p.Name
		if p.Description != "" {
			line += " — " + p.Description
		}
		line += "（" + status + "）"
		if len(p.Commands) > 0 {
			names := make([]string, 0, len(p.Commands))
			for _, c := range p.Commands {
				names = append(names, "/"+c.Name)
			}
			line += "\n  命令：" + strings.Join(names, "、")
		}
		names := make([]string, 0, len(p.Tools)+len(toolNames[p.Name]))
		seen := map[string]bool{}
		for _, t := range p.Tools {
			if t.Name != "" && !seen[t.Name] {
				seen[t.Name] = true
				names = append(names, t.Name)
			}
		}
		for _, name := range toolNames[p.Name] {
			if name != "" && !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
		if len(names) > 0 {
			line += "\n  工具：" + strings.Join(names, "、")
		}
		out = append(out, line)
	}
	return out
}

func (m *Manager) context(plugin string) *Context {
	return &Context{
		plugin: plugin,
		host:   m.host,
		track: func(name string) {
			m.mu.Lock()
			m.toolNames[plugin] = append(m.toolNames[plugin], name)
			m.mu.Unlock()
		},
	}
}

func (m *Manager) registerTool(plugin string, t tools.Tool) error {
	if m.host == nil {
		return errors.New("插件宿主未初始化")
	}
	if t.Name == "" {
		return errors.New("工具名为空")
	}
	if err := m.host.RegisterTool(t); err != nil {
		return err
	}
	m.mu.Lock()
	m.toolNames[plugin] = append(m.toolNames[plugin], t.Name)
	m.mu.Unlock()
	return nil
}

func (m *Manager) fail(plugin string, err error) {
	m.mu.Lock()
	m.failed[plugin] = err.Error()
	m.mu.Unlock()
}

// invoke 把插件回调的 panic 转成错误，避免单个插件拖垮 REPL。
func invoke(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("插件 panic：%v", r)
		}
	}()
	return fn()
}

func findCommand(name string) (Plugin, Command, bool) {
	for _, p := range Registered() {
		for _, c := range p.Commands {
			if c.Name == name {
				return p, c, true
			}
		}
	}
	return Plugin{}, Command{}, false
}

func normalizeName(s string) string {
	return strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(s), "/")))
}
