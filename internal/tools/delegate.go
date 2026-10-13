package tools

import (
	"context"
	"encoding/json"
	"strings"
)

func registerDelegate() {
	Register(Tool{
		Name:        "delegate_task",
		Description: "把一个可以独立完成的子任务交给另一个模型实例。该实例没有工具、也看不到当前对话，只返回一段文本。需要读文件、改文件或执行命令的任务不要委派。",
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
	return fn(ctx, task)
}
