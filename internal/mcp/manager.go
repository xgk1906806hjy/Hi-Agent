package mcp

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
	"sync"

	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
)

// Status 是一个 MCP server 的连接状态。
type Status struct {
	Name      string
	Connected bool
	Tools     int
	Error     string
}

// Manager 管理当前工作区的 MCP server 与动态工具。
type Manager struct {
	mu         sync.Mutex
	root       string
	configured bool
	configErr  string
	servers    map[string]*server
	statuses   []Status
	registered []string
}

// New 创建空管理器；调用 Load 后才读取配置。
func New() *Manager {
	return &Manager{servers: map[string]*server{}}
}

// Load 关闭旧连接，按 root/.geekagent/mcp.json 重新连接并注册工具。
// 单个 server 失败只影响自身；返回值汇总配置/连接错误供 REPL 提示。
func (m *Manager) Load(root string) error {
	if m == nil {
		return nil
	}
	m.Close()

	m.mu.Lock()
	m.root = root
	m.configured = false
	m.configErr = ""
	m.statuses = nil
	m.servers = map[string]*server{}
	m.registered = nil
	m.mu.Unlock()

	entries, found, err := loadConfig(root)
	if !found {
		return nil
	}

	m.mu.Lock()
	m.configured = true
	m.mu.Unlock()

	if err != nil {
		m.mu.Lock()
		m.configErr = err.Error()
		m.mu.Unlock()
		return err
	}
	if len(entries) == 0 {
		return nil
	}

	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	var errs []string
	for _, name := range names {
		cfg := entries[name]
		st := Status{Name: name}
		if strings.TrimSpace(cfg.Command) == "" {
			st.Error = "缺少 command"
			errs = append(errs, name+"："+st.Error)
			m.addStatus(st)
			continue
		}

		srv, err := startServer(name, root, cfg)
		if err != nil {
			st.Error = err.Error()
			errs = append(errs, name+"："+st.Error)
			m.addStatus(st)
			continue
		}
		infos, err := srv.listTools()
		if err != nil {
			st.Error = err.Error()
			errs = append(errs, name+"："+st.Error)
			srv.Close()
			m.addStatus(st)
			continue
		}

		m.mu.Lock()
		m.servers[name] = srv
		m.mu.Unlock()

		st.Connected = true
		st.Tools = len(infos)
		var registerErrs []string
		for _, info := range infos {
			if strings.TrimSpace(info.Name) == "" {
				continue
			}
			full := fullToolName(name, info.Name)
			if toolExists(full) {
				registerErrs = append(registerErrs, full+"（重名，已跳过）")
				continue
			}
			description := strings.TrimSpace(info.Description)
			if description == "" {
				description = "MCP 工具 " + info.Name
			}
			description = "MCP[" + name + "] " + description
			params := inputSchema(info.InputSchema)
			srv := srv
			toolName := info.Name
			if err := registerTool(tools.Tool{
				Name:        full,
				Description: description,
				Parameters:  params,
				Run: func(argsJSON string) (string, error) {
					return srv.callTool(context.Background(), toolName, argsJSON)
				},
			}); err != nil {
				registerErrs = append(registerErrs, err.Error())
				continue
			}
			m.mu.Lock()
			m.registered = append(m.registered, full)
			m.mu.Unlock()
		}
		if len(registerErrs) > 0 {
			st.Error = strings.Join(registerErrs, "；")
			errs = append(errs, name+"："+st.Error)
		}
		m.addStatus(st)
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "；"))
	}
	return nil
}

func (m *Manager) addStatus(st Status) {
	m.mu.Lock()
	m.statuses = append(m.statuses, st)
	m.mu.Unlock()
}

func toolExists(name string) bool {
	for _, n := range tools.Names() {
		if n == name {
			return true
		}
	}
	return false
}

func registerTool(t tools.Tool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("注册工具 %s 失败：%v", t.Name, r)
		}
	}()
	tools.Register(t)
	return nil
}

func inputSchema(raw json.RawMessage) any {
	schema := any(map[string]any{"type": "object", "properties": map[string]any{}})
	if len(raw) == 0 {
		return schema
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil || v == nil {
		return schema
	}
	return v
}

// fullToolName 生成 OpenAI function 名称：mcp_<server>_<tool>。
func fullToolName(server, tool string) string {
	name := "mcp_" + safeName(server) + "_" + safeName(tool)
	if len(name) <= 64 {
		return name
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	suffix := hex.EncodeToString(h.Sum(nil))
	return name[:55] + "_" + suffix
}

func safeName(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "x"
	}
	return out
}

// StatusLines 返回 /mcp 与 GUI 可展示的状态行。
func (m *Manager) StatusLines() []string {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.configErr != "" {
		return []string{"MCP 配置错误：" + m.configErr}
	}
	if !m.configured {
		return []string{"（未配置 MCP：.geekagent/mcp.json）"}
	}
	if len(m.statuses) == 0 {
		return []string{"（MCP 配置为空）"}
	}

	lines := make([]string, 0, len(m.statuses)+1)
	hasTools := false
	for _, st := range m.statuses {
		connected := st.Connected
		if connected {
			if srv := m.servers[st.Name]; srv != nil {
				select {
				case <-srv.done:
					connected = false
				default:
				}
			}
		}
		if connected {
			line := fmt.Sprintf("- %s：已连接，%d 个工具", st.Name, st.Tools)
			if st.Error != "" {
				line += "（" + st.Error + "）"
			}
			lines = append(lines, line)
			if st.Tools > 0 {
				hasTools = true
			}
		} else {
			msg := strings.TrimSpace(st.Error)
			if msg == "" {
				msg = "连接已断开"
			}
			lines = append(lines, fmt.Sprintf("- %s：未连接（%s）", st.Name, msg))
		}
	}
	if hasTools {
		lines = append(lines, "（MCP 工具默认 deny，需在 GeekAgent.json 的 allow 中放行）")
	}
	return lines
}

// Configured 表示当前工作区是否存在 MCP 配置（含损坏配置）。
func (m *Manager) Configured() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.configured
}

// Close 关闭全部 MCP 子进程并注销动态工具；幂等。
func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.mu.Lock()
	servers := m.servers
	names := m.registered
	m.servers = map[string]*server{}
	m.registered = nil
	m.statuses = nil
	m.configured = false
	m.configErr = ""
	m.mu.Unlock()

	for _, srv := range servers {
		srv.Close()
	}
	for _, name := range names {
		tools.Unregister(name)
	}
}
