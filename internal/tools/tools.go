// Package tools 维护进程内工具注册表：说明书、执行函数、OpenAI tools 转换与按名执行。
//
// 新增工具必须通过 Register；禁止在 chat 循环里硬编码工具列表。
// 错误以文本返回给模型（Exec 不向上抛），便于工具循环继续。
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

// Tool 工具最小抽象：给模型看的说明书 + 本地执行函数。
type Tool struct {
	Name        string                                // 唯一名称，对应 function.name
	Description string                                // 自然语言说明，影响模型是否选用
	Parameters  any                                   // JSON Schema 风格参数定义
	Run         func(argsJSON string) (string, error) // 执行体；err 会被 Exec 转成文本
}

var (
	mu       sync.RWMutex // 保护 registry
	registry []Tool       // 注册顺序即 init 注册顺序
)

// Register 注册工具；空名或重名会 panic（启动期失败，避免静默丢工具）。
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

// Names 返回已注册工具名（供 /help 等文案）。
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, len(registry))
	for i, t := range registry {
		out[i] = t.Name
	}
	return out
}

// readonlyToolNames 后期工具集收缩时仍暴露的只读工具（与 Day7 护栏配合）。
var readonlyToolNames = map[string]struct{}{
	"get_current_time": {},
	"ls":               {},
	"glob":             {},
	"read":             {},
	"search":           {},
	"rag_search":       {},
}

// ReadonlyNames 返回「已注册 ∩ 只读集合」的工具名（顺序随 registry）。
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

// ToOpenAIAllow 仅暴露 allow 中的工具；allow 为 nil 时暴露全部，非 nil 空切片表示一个都不暴露。
// chat 在工具轮次后期用 ReadonlyNames() 作为 allow，实现「只读收缩」；
// 加载技能后则用技能声明的工具白名单。
func ToOpenAIAllow(allow []string) []openai.Tool {
	mu.RLock()
	defer mu.RUnlock()
	var allowSet map[string]struct{}
	if allow != nil {
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

// Exec 按名称执行工具；错误与未知工具一律变成字符串返回给模型。
// 在注册表锁外调用 Run，避免慢工具（如 shell）长时间占用注册表锁；
// 先过权限策略（Deny 直接返回、Allow 本次免确认），结果统一脱敏。
func Exec(name, argsJSON string) string {
	return ExecContext(context.Background(), name, argsJSON)
}

// ExecContext 与 Exec 相同，但把 ctx 传给需要它的工具（如 delegate_task），便于回复被中断时取消。
func ExecContext(ctx context.Context, name, argsJSON string) string {
	if ctx == nil {
		ctx = context.Background()
	}
	return redact(execRaw(ctx, name, argsJSON))
}

func execRaw(ctx context.Context, name, argsJSON string) string {
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
	d := decide(name, argsJSON)
	if d == Deny {
		return denied(name)
	}
	execMu.Lock()
	defer execMu.Unlock()
	preApproved.Store(d == Allow)
	defer preApproved.Store(false)
	callCtx = ctx
	defer func() { callCtx = nil }()
	result, err := tool.Run(argsJSON)
	if err != nil {
		return fmt.Sprintf("工具执行失败：%v", err)
	}
	return result
}

// init 按固定顺序注册内置工具：时间 → shell → 文件族 → 计划/委派/记忆 → 检索。
func init() {
	registerTime()
	registerShell()
	registerFiles()
	registerPlan()
	registerDelegate()
	registerMemory()
	registerRAG()
	registerSearch()
	registerFetch()
}
