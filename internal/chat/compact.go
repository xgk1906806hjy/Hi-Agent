package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

const (
	defaultMaxHistoryChars = 4000
	keepRecent             = 6
)

const compressSystem = `你是对话压缩器。把用户贴出的历史对话压缩成简洁的中文要点，尽量保留以下信息：
- 用户的目标、需求、做过的决定与偏好；
- 出现过的文件路径、shell 命令、工具调用与关键结论；
- 尚未完成、仍在推进中的事项。
只输出压缩后的要点，不要解释、不要寒暄、不要保留逐字对话。`

func maxHistoryChars() int {
	v := strings.TrimSpace(os.Getenv("GEEKAGENT_MAX_HISTORY"))
	if v == "" {
		return defaultMaxHistoryChars
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return defaultMaxHistoryChars
	}
	return n
}

func (c *Chat) historySize() int {
	b, err := json.Marshal(c.history)
	if err != nil {
		return 0
	}
	return len(b)
}

// compactOldMessages 把除最近 keepRecent 条外的旧消息压成一条 system 摘要。
// 返回被合并的旧消息条数；无摘要产出时返回 0。
func (c *Chat) compactOldMessages(ctx context.Context) (int, error) {
	split := len(c.history) - keepRecent
	if split <= 0 {
		return 0, nil
	}
	// 不能把 role=tool 留在 recent 开头（缺少带 tool_calls 的 assistant），否则下一请求会 400。
	for split > 0 && c.history[split].Role == openai.ChatMessageRoleTool {
		split--
	}
	if split <= 0 {
		return 0, nil
	}

	old := append([]openai.ChatCompletionMessage(nil), c.history[:split]...)
	recent := append([]openai.ChatCompletionMessage(nil), c.history[split:]...)

	raw, err := json.MarshalIndent(old, "", "  ")
	if err != nil {
		return 0, err
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: compressSystem},
			{Role: openai.ChatMessageRoleUser, Content: string(raw)},
		},
	})
	if err != nil {
		return 0, err
	}
	usage := resp.Usage
	c.recordUsage(&usage, false)
	if len(resp.Choices) == 0 {
		return 0, nil
	}
	text := strings.TrimSpace(resp.Choices[0].Message.Content)
	if text == "" {
		return 0, nil
	}

	c.history = append([]openai.ChatCompletionMessage{{
		Role:    openai.ChatMessageRoleSystem,
		Content: "【此前对话摘要】\n" + text,
	}}, recent...)
	c.estimateContext()
	return len(old), nil
}

// Compact 手动压缩（/compact）；返回给用户看的提示文案。
func (c *Chat) Compact(ctx context.Context) string {
	if len(c.history) <= keepRecent {
		return "历史压缩：没有可压缩的旧消息"
	}
	// 手动压缩只计入累计，不改写上一轮回复的「本轮」用量。
	turn := c.turnUsage
	defer func() { c.turnUsage = turn }()
	dropped, err := c.compactOldMessages(ctx)
	if err != nil {
		return "历史压缩失败：保留原历史"
	}
	if dropped > 0 {
		return fmt.Sprintf("历史压缩：%d 条旧消息合并为 1 条摘要", dropped)
	}
	return "历史压缩：模型未产出摘要，保留原历史"
}
