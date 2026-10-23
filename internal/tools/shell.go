package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

const (
	shellTimeout   = 10 * time.Second // 单次命令超时
	maxOutputChars = 2000             // 合并输出的最大 rune 数
)

// shellArgs 对应 run_shell 的 JSON 参数。
type shellArgs struct {
	Command string `json:"command"`
}

// registerShell 注册 run_shell：确认 + 超时 + 截断。
func registerShell() {
	Register(Tool{
		Name:        "run_shell",
		Description: "在当前工作区执行一条 shell 命令，返回合并后的标准输出/错误。工作目录固定为工作区；命令里的绝对路径和以 .. 开头的路径若越出工作区会被拒绝。这不是操作系统沙箱，环境变量展开等绕过挡不住。执行前会向用户确认。纯查看文件请优先用 ls / read / glob。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{
					"type":        "string",
					"description": "要执行的 shell 命令",
				},
			},
			"required":             []string{"command"},
			"additionalProperties": false,
		},
		Run: runShell,
	})
}

// runShell 解析参数 → 确认 → 执行 → 合并 stdout/stderr → 截断。
// 超时与非零退出仍返回说明文本（error=nil），让模型能读到失败原因。
func runShell(argsJSON string) (string, error) {
	var args shellArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数不是合法 JSON：%w", err)
	}
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "缺少参数 command", nil
	}

	if !ask(fmt.Sprintf("即将执行命令：%s", command)) {
		return "已取消执行", nil
	}
	if msg := rejectEscapingPaths(command); msg != "" {
		return msg, nil
	}

	parent := callCtx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, shellTimeout)
	defer cancel()

	cmd := shellCommand(ctx, command)
	cmd.Dir = workspaceDir()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	out = strings.TrimSpace(out)

	if ctx.Err() == context.DeadlineExceeded {
		return truncate(fmt.Sprintf("命令执行失败（超时 %s）\n%s", shellTimeout, out)), nil
	}
	if parent.Err() != nil {
		return truncate("命令已取消\n" + out), nil
	}
	if err != nil {
		if out == "" {
			out = err.Error()
		}
		return truncate(fmt.Sprintf("命令执行失败（%v）\n%s", err, out)), nil
	}
	if out == "" {
		out = "(无输出)"
	}
	return truncate(out), nil
}

// shellCommand 按操作系统选择解释器：Windows 用 cmd /C，其它用 bash -c。
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "bash", "-c", command)
}

func workspaceDir() string {
	if safe, err := guard("."); err == nil && filepath.IsAbs(safe) {
		return safe
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

// rejectEscapingPaths 检查命令文本里的绝对路径和以 .. 开头的相对路径。
// 只看字面量，挡不住 subst、环境变量展开等绕过。
func rejectEscapingPaths(command string) string {
	for _, tok := range shellTokens(command) {
		tok = strings.Trim(tok, `"'`)
		if !looksLikeCheckedPath(tok) {
			continue
		}
		if _, err := guard(tok); err != nil {
			return "拒绝执行：" + err.Error()
		}
	}
	return ""
}

func looksLikeCheckedPath(tok string) bool {
	if tok == "" {
		return false
	}
	if filepath.IsAbs(tok) {
		return true
	}
	return tok == ".." || strings.HasPrefix(tok, "../") || strings.HasPrefix(tok, `..\`)
}

func shellTokens(command string) []string {
	var out []string
	var b strings.Builder
	var quote rune
	flush := func() {
		if b.Len() > 0 {
			out = append(out, b.String())
			b.Reset()
		}
	}
	for _, r := range command {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
				continue
			}
			b.WriteRune(r)
		case r == '\'' || r == '"':
			quote = r
		case unicode.IsSpace(r):
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return out
}

// truncate 按 rune 截断过长输出，并附带原长度提示。
func truncate(text string) string {
	r := []rune(text)
	if len(r) <= maxOutputChars {
		return text
	}
	return fmt.Sprintf("%s\n...(输出已截断，原共 %d 字符)", string(r[:maxOutputChars]), len(r))
}
