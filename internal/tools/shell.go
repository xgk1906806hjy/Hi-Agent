package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	shellTimeout   = 10 * time.Second
	maxOutputChars = 2000
)

type shellArgs struct {
	Command string `json:"command"`
}

func registerShell() {
	Register(Tool{
		Name:        "run_shell",
		Description: "在本地执行一条 shell 命令，返回合并后的标准输出/错误。执行前会向用户确认。纯查看文件请优先用 ls / read / glob；需要跑测试、装依赖、查进程时再用本工具。",
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

func runShell(argsJSON string) (string, error) {
	var args shellArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("参数不是合法 JSON：%w", err)
	}
	command := strings.TrimSpace(args.Command)
	if command == "" {
		return "缺少参数 command", nil
	}

	if !confirm(fmt.Sprintf("即将执行命令：%s", command)) {
		return "已取消执行", nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), shellTimeout)
	defer cancel()

	cmd := shellCommand(ctx, command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	out := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	out = strings.TrimSpace(out)

	if ctx.Err() == context.DeadlineExceeded {
		return truncate(fmt.Sprintf("命令执行失败（超时 %s）\n%s", shellTimeout, out)), nil
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

func shellCommand(ctx context.Context, command string) *exec.Cmd {
	if runtime.GOOS == "windows" {
		return exec.CommandContext(ctx, "cmd", "/C", command)
	}
	return exec.CommandContext(ctx, "bash", "-c", command)
}

func truncate(text string) string {
	r := []rune(text)
	if len(r) <= maxOutputChars {
		return text
	}
	return fmt.Sprintf("%s\n...(输出已截断，原共 %d 字符)", string(r[:maxOutputChars]), len(r))
}
