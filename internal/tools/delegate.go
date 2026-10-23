package tools

import (
	"context"
	"encoding/json"
	"strings"
)

func registerDelegate() {
	Register(Tool{
		Name:        "delegate_task",
		Description: "把一个子任务交给另一个模型实例。它能看到当前对话并使用工具，结果只以文本回到当前对话，不写回主历史。子任务里不能再委派。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{"type": "string", "description": "自洽的任务说明，需包含完成它所需的全部信息"},
			},
			"required":             []string{"task"},
			"additionalProperties": false,
		},
		Run: runDelegate,
	})
}

func runDelegate(argsJSON string) (string, error) {
	var args struct {
		Task string `json:"task"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	task := strings.TrimSpace(args.Task)
	if task == "" {
		return "缺少参数 task", nil
	}
	fn := currentDelegator()
	if fn == nil {
		return "委派未配置", nil
	}
	ctx := callCtx
	if ctx == nil {
		ctx = context.Background()
	}
	// 子对话自己还会调工具，必须先放开本轮的执行锁。
	execMu.Unlock()
	defer execMu.Lock()
	return fn(ctx, task)
}
