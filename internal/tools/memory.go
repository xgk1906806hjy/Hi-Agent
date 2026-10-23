package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xgk1906806hjy/Hi-Agent/internal/memory"
)

func registerMemory() {
	Register(Tool{
		Name:        "memory_write",
		Description: "把一条长期记忆追加到当前项目的 .geekagent/memory.json。用于记下跨会话仍有效的偏好或事实。不能修改或删除已有记忆。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"content": map[string]any{"type": "string", "description": "要记住的内容"},
			},
			"required":             []string{"content"},
			"additionalProperties": false,
		},
		Run: runMemoryWrite,
	})
	Register(Tool{
		Name:        "memory_search",
		Description: "在当前项目的长期记忆里做不区分大小写的子串搜索，最多返回 5 条。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "description": "搜索词"},
			},
			"required":             []string{"query"},
			"additionalProperties": false,
		},
		Run: runMemorySearch,
	})
}

func runMemoryWrite(argsJSON string) (string, error) {
	var args struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	e, err := memory.Write(args.Content)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已记住 %s：%s", e.ID, e.Content), nil
}

func runMemorySearch(argsJSON string) (string, error) {
	var args struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", err
	}
	entries, err := memory.Search(args.Query)
	if err != nil {
		return "", err
	}
	if len(entries) == 0 {
		return fmt.Sprintf("没有匹配 %q 的记忆", strings.TrimSpace(args.Query)), nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "找到 %d 条：\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(&b, "- %s %s\n", e.ID, e.Content)
	}
	return strings.TrimRight(b.String(), "\n"), nil
}
