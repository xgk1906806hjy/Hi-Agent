// Package chat 持有对话 history（即 OpenAI Chat Completions 的 messages，零转换），
// 提供流式多轮回复、工具调用循环、历史压缩、循环护栏与用量统计。
package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"hi-agent/internal/tools"

	openai "github.com/sashabaranov/go-openai"
)

// ErrInterrupted 表示用户中断了当前回复；已完成的工具结果保留在 history 中。
var ErrInterrupted = errors.New("interrupted")

// 上一轮结束原因码（供 Status 映射为中文）。
const (
	endOK        = "ok"
	endMaxTurns  = "max_turns"
	endDup       = "dup"
	endBudget    = "budget"
	endInterrupt = "interrupt"
	endEmpty     = ""
)

// Chat 持有 history（即 messages），提供流式回复与工具调用循环。
type Chat struct {
	client  *openai.Client
	model   string
	history []openai.ChatCompletionMessage

	// 上一轮 StreamReply 的护栏统计（供 /status）。
	lastTurns  int
	lastTools  int
	lastChars  int
	lastReason string
	lastShrunk bool

	turnUsage    TokenUsage // 本轮（最近一次 StreamReply）
	cumUsage     TokenUsage // 进程累计
	ctxTokens    int
	ctxEstimated bool
	ctxBase      int // 真实 prompt 相对 history 体积的固定开销（工具定义等）
	onUsage      func()
}

// New 创建对话实例；baseURL 会去掉尾部 "/"。
func New(baseURL, apiKey, model string) *Chat {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = strings.TrimRight(baseURL, "/")
	return &Chat{
		client:  openai.NewClientWithConfig(cfg),
		model:   model,
		history: nil,
	}
}

// pendingCall 流式聚拢中的单个 tool_call（按 Index 合并分片）。
type pendingCall struct {
	id   string
	name string
	args string
}

// StreamReply 将用户输入入队；历史超阈值时先压缩；若模型发起 tool_calls 则执行并回传。
// ctx 取消时返回 ErrInterrupted，不回滚已写入的 history。
// onDelta：模型正文片段，或以 "\n[" 开头的进度行（由 REPL 染黄）。
func (c *Chat) StreamReply(ctx context.Context, userInput string, onDelta func(string)) error {
	// —— 快照与入队：非中断错误时回滚到入队 user 之前 ——
	snapshot := append([]openai.ChatCompletionMessage(nil), c.history...)
	c.history = append(c.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userInput,
	})
	c.lastTurns = 0
	c.lastTools = 0
	c.lastChars = 0
	c.lastReason = endEmpty
	c.lastShrunk = false
	c.turnUsage = TokenUsage{}

	// —— 自动压缩：体积超阈值且条数多于 keepRecent ——
	if c.historySize() > maxHistoryChars() {
		if len(c.history) > keepRecent {
			dropped, err := c.compactOldMessages(ctx)
			if err != nil {
				if isInterrupt(ctx, err) {
					c.lastReason = endInterrupt
					return ErrInterrupted
				}
				onDelta("\n[历史压缩失败：保留原历史]\n")
			} else if dropped > 0 {
				onDelta(fmt.Sprintf("\n[历史压缩：%d 条旧消息合并为 1 条摘要]\n", dropped))
			} else {
				onDelta("\n[历史压缩：模型未产出摘要，保留原历史]\n")
			}
		}
	}

	limit := maxToolTurns()
	budget := maxReplyChars()
	shrinkAt := shrinkToolsAfter()
	tracker := newDupTracker(maxDupTools(), dupWindow())
	readonly := tools.ReadonlyNames()
	shrunk := false
	replyChars := 0

	// —— 工具循环 ——
	for turn := 0; turn < limit; turn++ {
		if err := ctx.Err(); err != nil {
			c.lastReason = endInterrupt
			c.lastChars = replyChars
			return ErrInterrupted
		}

		// 按轮次选择全量或只读工具集
		var openaiTools []openai.Tool
		if shrinkAt > 0 && turn >= shrinkAt {
			openaiTools = tools.ToOpenAIAllow(readonly)
			if !shrunk {
				shrunk = true
				c.lastShrunk = true
				onDelta(fmt.Sprintf("\n[工具集收缩：第 %d 轮起仅保留只读工具 %s]\n", turn+1, strings.Join(readonly, ", ")))
			}
		} else {
			openaiTools = tools.ToOpenAI()
		}

		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:         c.model,
			Messages:      c.history,
			Tools:         openaiTools,
			Stream:        true,
			StreamOptions: streamOptions(),
		})
		if err != nil {
			if isInterrupt(ctx, err) {
				c.lastReason = endInterrupt
				c.lastChars = replyChars
				return ErrInterrupted
			}
			c.history = snapshot
			return err
		}

		// 收流：聚拢正文与按 Index 分片的 tool_calls
		var answer strings.Builder
		calls := map[int]*pendingCall{}

		for {
			resp, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				stream.Close()
				if isInterrupt(ctx, recvErr) {
					// 中断：半截正文入库（无完整工具名时），不回滚
					if answer.Len() > 0 && !allNamed(calls) {
						c.history = append(c.history, openai.ChatCompletionMessage{
							Role:    openai.ChatMessageRoleAssistant,
							Content: answer.String(),
						})
					}
					c.lastReason = endInterrupt
					c.lastChars = replyChars
					return ErrInterrupted
				}
				c.history = snapshot
				return recvErr
			}
			c.recordUsage(resp.Usage, true)
			if len(resp.Choices) == 0 {
				continue
			}
			delta := resp.Choices[0].Delta
			if delta.Content != "" {
				answer.WriteString(delta.Content)
				onDelta(delta.Content)
			}
			for _, tc := range delta.ToolCalls {
				idx := 0
				if tc.Index != nil {
					idx = *tc.Index
				}
				call := calls[idx]
				if call == nil {
					call = &pendingCall{}
					calls[idx] = call
				}
				if tc.ID != "" {
					call.id = tc.ID
				}
				if tc.Function.Name != "" {
					call.name += tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					call.args += tc.Function.Arguments
				}
			}
		}
		stream.Close()
		c.lastTurns = turn + 1
		replyChars += utf8.RuneCountInString(answer.String())
		c.lastChars = replyChars

		// 分支：完整 tool_calls → 执行并做护栏检查
		if len(calls) > 0 && allNamed(calls) {
			idxs := make([]int, 0, len(calls))
			for i := range calls {
				idxs = append(idxs, i)
			}
			sort.Ints(idxs)
			toolCalls := make([]openai.ToolCall, 0, len(idxs))
			for _, i := range idxs {
				call := calls[i]
				if call.id == "" {
					call.id = fmt.Sprintf("call_%d", i)
				}
				toolCalls = append(toolCalls, openai.ToolCall{
					ID:   call.id,
					Type: openai.ToolTypeFunction,
					Function: openai.FunctionCall{
						Name:      call.name,
						Arguments: call.args,
					},
				})
			}
			c.history = append(c.history, openai.ChatCompletionMessage{
				Role:      openai.ChatMessageRoleAssistant,
				Content:   answer.String(),
				ToolCalls: toolCalls,
			})

			// 顺序执行工具；熔断后仍回传 tool 消息以保证配对
			stopReason := ""
			for _, tc := range toolCalls {
				if stopReason != "" {
					c.history = append(c.history, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    "已因护栏熔断而跳过本次调用。",
						ToolCallID: tc.ID,
					})
					continue
				}

				key := toolCallKey(tc.Function.Name, tc.Function.Arguments)
				if tracker.add(key) {
					msg := fmt.Sprintf(
						"检测到重复工具调用（%s 在最近 %d 次调用中出现 %d 次相同参数），已跳过执行。请换参数或根据已有结果作答。",
						tc.Function.Name, tracker.window, tracker.count(key),
					)
					onDelta(fmt.Sprintf("\n[重复工具：%s]\n", msg))
					c.history = append(c.history, openai.ChatCompletionMessage{
						Role:       openai.ChatMessageRoleTool,
						Content:    msg,
						ToolCallID: tc.ID,
					})
					stopReason = endDup
					continue
				}

				result := tools.Exec(tc.Function.Name, tc.Function.Arguments)
				c.lastTools++
				replyChars += utf8.RuneCountInString(result)
				c.lastChars = replyChars
				line := fmt.Sprintf("\n[调用工具 %s → %s]\n", tc.Function.Name, result)
				onDelta(line)
				c.history = append(c.history, openai.ChatCompletionMessage{
					Role:       openai.ChatMessageRoleTool,
					Content:    result,
					ToolCallID: tc.ID,
				})

				if budget > 0 && replyChars >= budget {
					onDelta(fmt.Sprintf("\n[回复预算：本轮已用约 %d 字符（上限 %d），改为根据已有结果直接作答]\n", replyChars, budget))
					stopReason = endBudget
				}
			}
			if stopReason != "" {
				c.lastReason = stopReason
				if stopReason == endDup {
					onDelta("\n[重复工具熔断，改为根据已有结果直接作答]\n")
				}
				return c.streamFinalAnswer(ctx, snapshot, onDelta)
			}
			continue
		}

		// 分支：无完整工具调用 → 纯文本结束
		c.history = append(c.history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: answer.String(),
		})
		c.lastReason = endOK
		return nil
	}

	// —— 轮次上限 → 强制收尾 ——
	c.lastReason = endMaxTurns
	onDelta(fmt.Sprintf("\n[工具轮次已达上限（%d），改为根据已有结果直接作答]\n", limit))
	return c.streamFinalAnswer(ctx, snapshot, onDelta)
}

// streamFinalAnswer 禁止再调工具，基于已有结果给出用户可读结论。
// 临时 wrapUpSystem 只附在请求 messages 上，不写入持久 history。
func (c *Chat) streamFinalAnswer(ctx context.Context, snapshot []openai.ChatCompletionMessage, onDelta func(string)) error {
	if err := ctx.Err(); err != nil {
		c.lastReason = endInterrupt
		return ErrInterrupted
	}

	msgs := append(append([]openai.ChatCompletionMessage(nil), c.history...), openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: wrapUpSystem,
	})

	stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:         c.model,
		Messages:      msgs,
		ToolChoice:    "none",
		Stream:        true,
		StreamOptions: streamOptions(),
	})
	if err != nil {
		if isInterrupt(ctx, err) {
			c.lastReason = endInterrupt
			return ErrInterrupted
		}
		// 创建失败（非中断）：友好文案入库，不回滚，会话可继续
		fallback := "工具调用次数较多、出现重复调用或达到回复预算，已暂停继续调工具。请根据上面已返回的结果继续，或把任务拆小后再试。"
		onDelta(fallback)
		c.history = append(c.history, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: fallback,
		})
		return nil
	}
	defer stream.Close()

	var answer strings.Builder
	for {
		resp, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			if isInterrupt(ctx, recvErr) {
				if answer.Len() > 0 {
					c.history = append(c.history, openai.ChatCompletionMessage{
						Role:    openai.ChatMessageRoleAssistant,
						Content: answer.String(),
					})
				}
				c.lastReason = endInterrupt
				return ErrInterrupted
			}
			c.history = snapshot
			return recvErr
		}
		c.recordUsage(resp.Usage, true)
		if len(resp.Choices) == 0 {
			continue
		}
		if delta := resp.Choices[0].Delta.Content; delta != "" {
			answer.WriteString(delta)
			onDelta(delta)
		}
	}
	text := answer.String()
	if strings.TrimSpace(text) == "" {
		text = "已停止继续调用工具。请换一种问法，或把任务拆成更小的步骤后再试。"
		onDelta(text)
	}
	c.history = append(c.history, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: text,
	})
	return nil
}

// isInterrupt 判定 ctx 取消或错误本身为 Canceled / DeadlineExceeded。
func isInterrupt(ctx context.Context, err error) bool {
	if ctx.Err() != nil {
		return true
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// allNamed 要求每个 pending call 都已聚拢到非空函数名（流式分片完成的判据）。
func allNamed(calls map[int]*pendingCall) bool {
	if len(calls) == 0 {
		return false
	}
	for _, c := range calls {
		if c == nil || c.name == "" {
			return false
		}
	}
	return true
}

// Status 返回会话与上一轮循环护栏摘要（供 /status）。
func (c *Chat) Status() string {
	reason := c.lastReason
	if reason == "" {
		reason = "（尚无回复）"
	} else {
		switch reason {
		case endOK:
			reason = "正常结束"
		case endMaxTurns:
			reason = "轮次上限后强制收尾"
		case endDup:
			reason = "重复工具熔断后强制收尾"
		case endBudget:
			reason = "回复预算用尽后强制收尾"
		case endInterrupt:
			reason = "用户中断"
		}
	}
	budget := maxReplyChars()
	budgetLabel := "关闭"
	if budget > 0 {
		budgetLabel = fmt.Sprintf("%d", budget)
	}
	shrinkAt := shrinkToolsAfter()
	shrinkLabel := "关闭"
	if shrinkAt > 0 {
		shrinkLabel = fmt.Sprintf("第 %d 轮起只读", shrinkAt+1)
	}
	shrunkNote := "否"
	if c.lastShrunk {
		shrunkNote = "是"
	}
	ctxLabel := fmt.Sprintf("%d / %d", c.ctxTokens, contextWindow())
	if c.ctxEstimated {
		ctxLabel = "≈" + ctxLabel
	}
	return fmt.Sprintf(
		"模型：%s\n上下文 tokens：%s\n本轮 tokens：输入 %d，输出 %d（%d 次请求）\n累计 tokens：输入 %d，输出 %d（%d 次请求）\n消息条数：%d\n历史字符：%d / %d\n工具轮次上限：%d\n重复熔断：窗口内 %d 次同参（窗口 %d）\n回复预算：%s 字符\n工具收缩：%s\n上一轮：工具轮次 %d，执行工具 %d 次，正文+工具约 %d 字符，曾收缩：%s，结束原因：%s",
		c.model,
		ctxLabel,
		c.turnUsage.Prompt, c.turnUsage.Completion, c.turnUsage.Requests,
		c.cumUsage.Prompt, c.cumUsage.Completion, c.cumUsage.Requests,
		len(c.history),
		c.historySize(),
		maxHistoryChars(),
		maxToolTurns(),
		maxDupTools(),
		dupWindow(),
		budgetLabel,
		shrinkLabel,
		c.lastTurns,
		c.lastTools,
		c.lastChars,
		shrunkNote,
		reason,
	)
}

// History 返回当前 history 的深拷贝（供会话持久化）。
func (c *Chat) History() []openai.ChatCompletionMessage {
	return cloneMessages(c.history)
}

// SetHistory 用深拷贝替换 history，清空上一轮统计，并按新 history 估算上下文占用。
func (c *Chat) SetHistory(msgs []openai.ChatCompletionMessage) {
	c.history = cloneMessages(msgs)
	c.lastTurns = 0
	c.lastTools = 0
	c.lastChars = 0
	c.lastReason = endEmpty
	c.lastShrunk = false
	c.turnUsage = TokenUsage{}
	c.estimateContext()
}

// cloneMessages 深拷贝 messages（含 ToolCalls 切片），避免会话切换时共享底层数组。
func cloneMessages(in []openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	if len(in) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionMessage, len(in))
	copy(out, in)
	for i := range out {
		if len(out[i].ToolCalls) > 0 {
			out[i].ToolCalls = append([]openai.ToolCall(nil), out[i].ToolCalls...)
		}
	}
	return out
}

// Reset 清空对话记忆（累计 tokens 保留，属进程级统计）。
func (c *Chat) Reset() {
	c.history = nil
	c.lastTurns = 0
	c.lastTools = 0
	c.lastChars = 0
	c.lastReason = endEmpty
	c.lastShrunk = false
	c.turnUsage = TokenUsage{}
	c.estimateContext()
}
