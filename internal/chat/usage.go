package chat

import (
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultContextWindow = 128_000
	bytesPerToken        = 3 // history JSON 字节 → token 的粗略换算（中文 UTF-8 约 3 字节/字）
)

// TokenUsage 一组 token 计数；Requests 为产生这些用量的请求次数。
type TokenUsage struct {
	Prompt     int
	Completion int
	Requests   int
}

// Total 返回输入 + 输出。
func (u TokenUsage) Total() int {
	return u.Prompt + u.Completion
}

func (u *TokenUsage) add(v openai.Usage) {
	u.Prompt += v.PromptTokens
	u.Completion += v.CompletionTokens
	u.Requests++
}

// UsageStats 供面板 / 状态展示的用量快照。
type UsageStats struct {
	Turn       TokenUsage // 最近一次 StreamReply（含压缩与收尾请求）
	Cumulative TokenUsage // 进程内累计

	// ContextTokens 为最近一次主对话请求的 prompt+completion，即下一轮上下文的近似体积。
	ContextTokens    int
	ContextEstimated bool // true 表示由 history 体积估算（尚无真实用量或历史刚被替换）
	ContextWindow    int

	HistoryChars int
	HistoryLimit int
	Messages     int

	LastToolTurns int
	LastToolCalls int
}

// streamUsageEnabled 默认开启；GEEKAGENT_STREAM_USAGE=0 关闭（兼容不支持 stream_options 的网关）。
func streamUsageEnabled() bool {
	return strings.TrimSpace(os.Getenv("GEEKAGENT_STREAM_USAGE")) != "0"
}

func streamOptions() *openai.StreamOptions {
	if !streamUsageEnabled() {
		return nil
	}
	return &openai.StreamOptions{IncludeUsage: true}
}

func contextWindow() int {
	return envInt("GEEKAGENT_CONTEXT_WINDOW", defaultContextWindow)
}

// Model 返回模型名。
func (c *Chat) Model() string {
	return c.model
}

// SetUsageHook 设置用量更新回调（每次收到上游 usage 后调用，供面板即时刷新）。
func (c *Chat) SetUsageHook(fn func()) {
	c.onUsage = fn
}

// Usage 返回当前用量快照。
func (c *Chat) Usage() UsageStats {
	return UsageStats{
		Turn:             c.turnUsage,
		Cumulative:       c.cumUsage,
		ContextTokens:    c.ctxTokens,
		ContextEstimated: c.ctxEstimated,
		ContextWindow:    contextWindow(),
		HistoryChars:     c.historySize(),
		HistoryLimit:     maxHistoryChars(),
		Messages:         len(c.history),
		LastToolTurns:    c.lastTurns,
		LastToolCalls:    c.lastTools,
	}
}

// recordUsage 累计一次请求的用量；mainRequest 为 true 时同时刷新上下文占用。
func (c *Chat) recordUsage(u *openai.Usage, mainRequest bool) {
	if u == nil {
		return
	}
	c.turnUsage.add(*u)
	c.cumUsage.add(*u)
	if mainRequest {
		c.ctxTokens = u.PromptTokens + u.CompletionTokens
		c.ctxEstimated = false
		// 流中 history 仍是本次请求的 messages；差值即工具定义等固定开销。
		c.ctxBase = max(0, u.PromptTokens-c.historySize()/bytesPerToken)
	}
	if c.onUsage != nil {
		c.onUsage()
	}
}

// estimateContext 在 history 被整体替换（压缩 / 切会话 / 清空）后，
// 按 JSON 体积粗估 token，并加上最近一次真实请求推算出的固定开销。
func (c *Chat) estimateContext() {
	if len(c.history) == 0 {
		c.ctxTokens = 0
		c.ctxEstimated = false
		return
	}
	c.ctxTokens = c.ctxBase + c.historySize()/bytesPerToken
	c.ctxEstimated = true
}
