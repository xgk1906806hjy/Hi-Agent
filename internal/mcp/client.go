package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	protocolVersion = "2024-11-05"
	clientName      = "hi-agent"
	clientVersion   = "0.1.0"

	initializeTimeout = 15 * time.Second
	listToolsTimeout  = 15 * time.Second
	callToolTimeout   = 60 * time.Second
)

type rpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *rpcError) Error() string {
	if e == nil {
		return ""
	}
	if e.Code == 0 {
		return e.Message
	}
	return fmt.Sprintf("%s（code %d）", e.Message, e.Code)
}

type rpcResult struct {
	result json.RawMessage
	err    error
}

type server struct {
	name   string
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser

	writeMu sync.Mutex
	mu      sync.Mutex
	pending map[int64]chan rpcResult
	nextID  int64
	err     error
	done    chan struct{}

	doneOnce  sync.Once
	closeOnce sync.Once
}

func startServer(name, root string, cfg serverConfig) (*server, error) {
	cmd := exec.Command(cfg.Command, cfg.Args...)
	cmd.Dir = resolveCwd(root, cfg.Cwd)
	cmd.Env = mergeEnv(cfg.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdin 失败：%w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("创建 stdout 失败：%w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("创建 stderr 失败：%w", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		_ = stderr.Close()
		return nil, fmt.Errorf("启动 %s 失败：%w", cfg.Command, err)
	}

	s := &server{
		name: name, cmd: cmd, stdin: stdin, stdout: stdout,
		pending: map[int64]chan rpcResult{}, done: make(chan struct{}),
	}
	go func() { _, _ = io.Copy(io.Discard, stderr) }()
	go s.readLoop()
	go s.waitLoop()

	ctx, cancel := context.WithTimeout(context.Background(), initializeTimeout)
	defer cancel()
	params := map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": clientName, "version": clientVersion},
	}
	if _, err := s.request(ctx, "initialize", params); err != nil {
		s.Close()
		return nil, fmt.Errorf("MCP 初始化失败：%w", err)
	}
	if err := s.notify("notifications/initialized", nil); err != nil {
		s.Close()
		return nil, fmt.Errorf("发送 initialized 失败：%w", err)
	}
	return s, nil
}

func resolveCwd(root, cwd string) string {
	if cwd == "" {
		return root
	}
	if filepath.IsAbs(cwd) {
		return cwd
	}
	return filepath.Join(root, cwd)
}

func mergeEnv(extra map[string]string) []string {
	if len(extra) == 0 {
		return os.Environ()
	}
	values := map[string]string{}
	for _, item := range os.Environ() {
		if i := strings.IndexByte(item, '='); i >= 0 {
			values[item[:i]] = item[i+1:]
		}
	}
	for k, v := range extra {
		values[k] = v
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+values[k])
	}
	return out
}

func (s *server) waitLoop() {
	err := s.cmd.Wait()
	if err == nil {
		err = fmt.Errorf("进程已退出")
	}
	s.fail(err)
}

func (s *server) readLoop() {
	scanner := bufio.NewScanner(s.stdout)
	scanner.Buffer(make([]byte, 64*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var msg struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Result  json.RawMessage `json:"result"`
			Error   *rpcError       `json:"error"`
		}
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}
		if msg.Method != "" {
			// server → client 请求/通知；基础客户端只回应 ping，其它通知忽略。
			if len(msg.ID) > 0 {
				if msg.Method == "ping" {
					s.writeMessage(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": map[string]any{}})
				} else {
					s.writeMessage(map[string]any{
						"jsonrpc": "2.0", "id": msg.ID,
						"error": map[string]any{"code": -32601, "message": "method not found"},
					})
				}
			}
			continue
		}
		if len(msg.ID) == 0 {
			continue
		}
		var id int64
		if err := json.Unmarshal(msg.ID, &id); err != nil {
			continue
		}
		var rerr error
		if msg.Error != nil {
			rerr = msg.Error
		}
		s.deliver(id, rpcResult{result: msg.Result, err: rerr})
	}
	if err := scanner.Err(); err != nil {
		s.fail(fmt.Errorf("读取 MCP 输出失败：%w", err))
		return
	}
	s.fail(fmt.Errorf("MCP server %s 连接已关闭", s.name))
}

func (s *server) deliver(id int64, res rpcResult) {
	s.mu.Lock()
	ch := s.pending[id]
	delete(s.pending, id)
	s.mu.Unlock()
	if ch != nil {
		ch <- res
	}
}

func (s *server) fail(err error) {
	if err == nil {
		err = fmt.Errorf("MCP server %s 已关闭", s.name)
	}
	s.doneOnce.Do(func() {
		s.mu.Lock()
		if s.err == nil {
			s.err = err
		}
		pending := s.pending
		s.pending = map[int64]chan rpcResult{}
		s.mu.Unlock()
		close(s.done)
		for _, ch := range pending {
			ch <- rpcResult{err: err}
		}
	})
}

func (s *server) closeErr() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	return fmt.Errorf("MCP server %s 已关闭", s.name)
}

func (s *server) request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-s.done:
		return nil, s.closeErr()
	default:
	}

	id := atomic.AddInt64(&s.nextID, 1)
	ch := make(chan rpcResult, 1)

	s.mu.Lock()
	if s.err != nil {
		err := s.err
		s.mu.Unlock()
		return nil, err
	}
	s.pending[id] = ch
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	msg := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if err := s.writeMessage(msg); err != nil {
		return nil, err
	}

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, res.err
		}
		return res.result, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("MCP 请求 %s 超时/取消：%w", method, ctx.Err())
	case <-s.done:
		return nil, s.closeErr()
	}
}

func (s *server) notify(method string, params any) error {
	msg := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		msg["params"] = params
	}
	return s.writeMessage(msg)
}

func (s *server) writeMessage(msg any) error {
	select {
	case <-s.done:
		return s.closeErr()
	default:
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("编码 MCP 消息失败：%w", err)
	}
	data = append(data, '\n')

	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	if s.stdin == nil {
		return fmt.Errorf("MCP stdin 已关闭")
	}
	if _, err := s.stdin.Write(data); err != nil {
		s.fail(fmt.Errorf("写入 MCP 请求失败：%w", err))
		return err
	}
	return nil
}

func (s *server) listTools() ([]toolInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), listToolsTimeout)
	defer cancel()

	var all []toolInfo
	cursor := ""
	for {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		raw, err := s.request(ctx, "tools/list", params)
		if err != nil {
			return nil, fmt.Errorf("列出工具失败：%w", err)
		}
		var page struct {
			Tools      []toolInfo `json:"tools"`
			NextCursor string     `json:"nextCursor"`
		}
		if err := json.Unmarshal(raw, &page); err != nil {
			return nil, fmt.Errorf("解析 tools/list 失败：%w", err)
		}
		all = append(all, page.Tools...)
		if page.NextCursor == "" || page.NextCursor == cursor {
			break
		}
		cursor = page.NextCursor
		if len(all) > 1000 {
			return nil, fmt.Errorf("工具数量超过 1000 个上限")
		}
	}
	return all, nil
}

func (s *server) callTool(ctx context.Context, name, argsJSON string) (string, error) {
	var args any = map[string]any{}
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("参数不是合法 JSON：%w", err)
		}
	}
	if args == nil {
		args = map[string]any{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, callToolTimeout)
		defer cancel()
	}

	raw, err := s.request(ctx, "tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", fmt.Errorf("调用 MCP 工具 %s 失败：%w", name, err)
	}
	var result struct {
		Content           []mcpContent    `json:"content"`
		IsError           bool            `json:"isError"`
		StructuredContent json.RawMessage `json:"structuredContent"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("解析 MCP 工具结果失败：%w", err)
	}

	parts := make([]string, 0, len(result.Content)+1)
	for _, item := range result.Content {
		switch item.Type {
		case "text":
			if item.Text != "" {
				parts = append(parts, item.Text)
			}
		case "image", "audio":
			label := item.Type
			if item.MIMEType != "" {
				label += " " + item.MIMEType
			}
			if item.Data != "" {
				label += fmt.Sprintf("（%d 字节，已省略）", len(item.Data))
			}
			parts = append(parts, "["+label+"]")
		case "resource":
			if len(item.Resource) > 0 {
				parts = append(parts, "[resource] "+string(item.Resource))
			} else {
				parts = append(parts, "[resource]")
			}
		default:
			if item.Text != "" {
				parts = append(parts, item.Text)
			} else if data, err := json.Marshal(item); err == nil {
				parts = append(parts, string(data))
			}
		}
	}
	if len(parts) == 0 && len(result.StructuredContent) > 0 {
		parts = append(parts, string(result.StructuredContent))
	}
	if len(parts) == 0 {
		parts = append(parts, "（MCP 工具未返回内容）")
	}
	text := strings.Join(parts, "\n")
	if result.IsError {
		return "MCP 工具返回错误：\n" + text, nil
	}
	return text, nil
}

// Close 关闭子进程；幂等。
func (s *server) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.stdin != nil {
			_ = s.stdin.Close()
		}
		if s.cmd != nil && s.cmd.Process != nil {
			_ = s.cmd.Process.Kill()
		}
		select {
		case <-s.done:
		case <-time.After(2 * time.Second):
			s.fail(fmt.Errorf("关闭 MCP server %s 超时", s.name))
		}
	})
}

type toolInfo struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpContent struct {
	Type     string          `json:"type"`
	Text     string          `json:"text"`
	Data     string          `json:"data"`
	MIMEType string          `json:"mimeType"`
	Resource json.RawMessage `json:"resource"`
}
