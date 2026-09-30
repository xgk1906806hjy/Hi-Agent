package tools

import (
	"encoding/json"
	"fmt"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Tool 工具最小抽象：说明书 + 执行函数。
type Tool struct {
	Name        string
	Description string
	Parameters  any
	Run         func(argsJSON string) (string, error)
}

var (
	mu       sync.RWMutex
	registry []Tool
)

// Register 注册工具；重名 panic，保证清单与执行一致。
func Register(t Tool) {
	if t.Name == "" {
		panic("tools: empty tool name")
	}
	mu.Lock()
	defer mu.Unlock()
	for _, existing := range registry {
		if existing.Name == t.Name {
			panic("tools: duplicate tool name: " + t.Name)
		}
	}
	registry = append(registry, t)
}

// Names 返回已注册工具名（供帮助文案等）。
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, len(registry))
	for i, t := range registry {
		out[i] = t.Name
	}
	return out
}

// readonlyToolNames 只读工具（后期收缩可见工具集时保留）。
var readonlyToolNames = map[string]struct{}{
	"get_current_time": {},
	"ls":               {},
	"glob":             {},
	"read":             {},
}

// ReadonlyNames 返回已注册的只读工具名。
func ReadonlyNames() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, 0, len(readonlyToolNames))
	for _, t := range registry {
		if _, ok := readonlyToolNames[t.Name]; ok {
			out = append(out, t.Name)
		}
	}
	return out
}

// ToOpenAI 转为 Chat Completions 的 tools 参数（全部已注册工具）。
func ToOpenAI() []openai.Tool {
	return ToOpenAIAllow(nil)
}

// ToOpenAIAllow 仅暴露 allow 中的工具；allow 为 nil/空时暴露全部。
func ToOpenAIAllow(allow []string) []openai.Tool {
	mu.RLock()
	defer mu.RUnlock()
	var allowSet map[string]struct{}
	if len(allow) > 0 {
		allowSet = make(map[string]struct{}, len(allow))
		for _, n := range allow {
			allowSet[n] = struct{}{}
		}
	}
	out := make([]openai.Tool, 0, len(registry))
	for i := range registry {
		t := registry[i]
		if allowSet != nil {
			if _, ok := allowSet[t.Name]; !ok {
				continue
			}
		}
		fn := openai.FunctionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
		out = append(out, openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &fn,
		})
	}
	return out
}

// Exec 按名称执行工具；错误以文本形式返回给模型。
func Exec(name, argsJSON string) string {
	mu.RLock()
	var tool *Tool
	for i := range registry {
		if registry[i].Name == name {
			t := registry[i]
			tool = &t
			break
		}
	}
	mu.RUnlock()
	if tool == nil {
		return fmt.Sprintf("未知工具：%s", name)
	}
	if argsJSON == "" {
		argsJSON = "{}"
	}
	if !json.Valid([]byte(argsJSON)) {
		return fmt.Sprintf("参数不是合法 JSON：%s", argsJSON)
	}
	result, err := tool.Run(argsJSON)
	if err != nil {
		return fmt.Sprintf("工具执行失败：%v", err)
	}
	return result
}

func init() {
	registerTime()
	registerShell()
	registerFiles()
}
