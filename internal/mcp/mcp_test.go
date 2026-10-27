package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgk1906806hjy/Hi-Agent/internal/tools"
)

// TestMCPHelperProcess 是测试用的最小 stdio MCP server。
// 测试进程带 GO_WANT_MCP_HELPER=1 重新执行当前测试二进制时进入此函数。
func TestMCPHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	encoder := json.NewEncoder(os.Stdout)

	for scanner.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		if len(req.ID) == 0 {
			continue // notification
		}

		var result any
		var rpcErr map[string]any
		switch req.Method {
		case "initialize":
			result = map[string]any{
				"protocolVersion": protocolVersion,
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "fake", "version": "1.0"},
			}
		case "tools/list":
			var p struct {
				Cursor string `json:"cursor"`
			}
			_ = json.Unmarshal(req.Params, &p)
			if p.Cursor == "" {
				result = map[string]any{
					"tools": []map[string]any{{
						"name":        "echo",
						"description": "回显文本",
						"inputSchema": map[string]any{
							"type":       "object",
							"properties": map[string]any{"text": map[string]any{"type": "string"}},
							"required":   []string{"text"},
						},
					}},
					"nextCursor": "page2",
				}
			} else {
				result = map[string]any{
					"tools": []map[string]any{{
						"name":        "shout",
						"description": "大写文本",
						"inputSchema": map[string]any{"type": "object"},
					}},
				}
			}
		case "tools/call":
			var p struct {
				Name      string `json:"name"`
				Arguments struct {
					Text string `json:"text"`
				} `json:"arguments"`
			}
			_ = json.Unmarshal(req.Params, &p)
			switch p.Name {
			case "echo":
				result = map[string]any{"content": []map[string]any{{"type": "text", "text": "echo: " + p.Arguments.Text}}}
			case "shout":
				result = map[string]any{"content": []map[string]any{{"type": "text", "text": strings.ToUpper(p.Arguments.Text)}}}
			default:
				rpcErr = map[string]any{"code": -32601, "message": "unknown tool"}
			}
		default:
			rpcErr = map[string]any{"code": -32601, "message": "method not found"}
		}

		msg := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if rpcErr != nil {
			msg["error"] = rpcErr
		} else {
			msg["result"] = result
		}
		_ = encoder.Encode(msg)
	}
	os.Exit(0)
}

func writeMCPConfig(t *testing.T, root, command string, args []string, env map[string]string) {
	t.Helper()
	dir := filepath.Join(root, ".geekagent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := map[string]any{
		"mcpServers": map[string]any{
			"fake": map[string]any{
				"command": command,
				"args":    args,
				"env":     env,
			},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasTool(name string) bool {
	for _, n := range tools.Names() {
		if n == name {
			return true
		}
	}
	return false
}

func TestManagerLoadCallAndClose(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	writeMCPConfig(t, root, exe, []string{"-test.run=TestMCPHelperProcess"}, map[string]string{
		"GO_WANT_MCP_HELPER": "1",
	})

	tools.SetPolicy(func(name, args string) tools.Decision { return tools.Allow })
	t.Cleanup(func() {
		tools.SetPolicy(nil)
	})

	m := New()
	t.Cleanup(m.Close)
	if err := m.Load(root); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !m.Configured() {
		t.Fatal("Configured() = false")
	}
	for _, name := range []string{"mcp_fake_echo", "mcp_fake_shout"} {
		if !hasTool(name) {
			t.Fatalf("工具 %s 未注册；已有 %v", name, tools.Names())
		}
	}
	lines := strings.Join(m.StatusLines(), "\n")
	if !strings.Contains(lines, "fake：已连接，2 个工具") {
		t.Fatalf("状态缺少连接信息：\n%s", lines)
	}

	out := tools.Exec("mcp_fake_echo", `{"text":"hello"}`)
	if out != "echo: hello" {
		t.Fatalf("echo 结果 = %q", out)
	}
	out = tools.Exec("mcp_fake_shout", `{"text":"quiet"}`)
	if out != "QUIET" {
		t.Fatalf("shout 结果 = %q", out)
	}

	m.Close()
	if hasTool("mcp_fake_echo") || hasTool("mcp_fake_shout") {
		t.Fatalf("Close 后动态工具仍存在：%v", tools.Names())
	}
}

func TestManagerMissingConfig(t *testing.T) {
	m := New()
	t.Cleanup(m.Close)
	if err := m.Load(t.TempDir()); err != nil {
		t.Fatalf("缺失配置不应报错：%v", err)
	}
	if m.Configured() {
		t.Fatal("缺失配置时 Configured() 应为 false")
	}
	if !strings.Contains(strings.Join(m.StatusLines(), "\n"), "未配置 MCP") {
		t.Fatalf("状态 = %v", m.StatusLines())
	}
}

func TestManagerBrokenConfig(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, ".geekagent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New()
	t.Cleanup(m.Close)
	if err := m.Load(root); err == nil {
		t.Fatal("损坏配置应返回错误")
	}
	lines := strings.Join(m.StatusLines(), "\n")
	if !strings.Contains(lines, "配置错误") {
		t.Fatalf("状态 = %v", m.StatusLines())
	}
}

func TestFullToolName(t *testing.T) {
	if got := fullToolName("demo", "read_file"); got != "mcp_demo_read_file" {
		t.Fatalf("got %q", got)
	}
	if got := fullToolName("demo", "a.b c"); got != "mcp_demo_a_b_c" {
		t.Fatalf("非法字符未替换：%q", got)
	}
	long := fullToolName("server", strings.Repeat("tool", 30))
	if len(long) > 64 {
		t.Fatalf("工具名过长：%d %q", len(long), long)
	}
	if !strings.HasPrefix(long, "mcp_server_") {
		t.Fatalf("工具名前缀错误：%q", long)
	}
}

func TestManagerServerFailureDoesNotBlockOthers(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	dir := filepath.Join(root, ".geekagent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`{"servers":{
		"bad":{"command":"definitely-not-a-real-command-xyz"},
		"fake":{"command":%q,"args":["-test.run=TestMCPHelperProcess"],"env":{"GO_WANT_MCP_HELPER":"1"}}
	}}`, exe)
	if err := os.WriteFile(filepath.Join(dir, "mcp.json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	m := New()
	t.Cleanup(m.Close)
	err = m.Load(root)
	if err == nil {
		t.Fatal("bad server 应返回错误")
	}
	if !hasTool("mcp_fake_echo") {
		t.Fatalf("fake server 工具应仍注册：%v", tools.Names())
	}
	lines := strings.Join(m.StatusLines(), "\n")
	if !strings.Contains(lines, "bad：未连接") || !strings.Contains(lines, "fake：已连接") {
		t.Fatalf("状态 = %v", m.StatusLines())
	}
}
