package tools

import (
	"encoding/json"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/plan"
)

func registerPlan() {
	Register(Tool{
		Name:        "todo_write",
		Description: "维护当前任务清单。按 id 合并；status 取 pending、in_progress、completed、cancelled。同时只能有一项 in_progress，新的进行中会把原来的降为 pending。清单只在本次进程有效。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"todos": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"id":      map[string]any{"type": "string", "description": "稳定 ID，同 ID 表示更新"},
							"content": map[string]any{"type": "string", "description": "任务内容；更新时可省略"},
							"status":  map[string]any{"type": "string", "description": "pending | in_progress | completed | cancelled"},
						},
						"required":             []string{"id"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"todos"},
			"additionalProperties": false,
		},
		Run: runTodoWrite,
	})
}

func runTodoWrite(argsJSON string) (string, error) {
	var args struct {
		Todos []plan.Item `json:"todos"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	if len(args.Todos) == 0 {
		return "缺少参数 todos", nil
	}
	text, err := plan.Apply(args.Todos)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(text), nil
}
