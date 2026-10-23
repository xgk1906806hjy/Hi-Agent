package chat

import (
	"os"
	"unicode/utf8"

	openai "github.com/sashabaranov/go-openai"
)

const agentsFile = "AGENTS.md"
const agentsLimit = 12000

// agentsNote 读取工作区根目录的 AGENTS.md。缺失则跳过；超长截断。
// 只附在发给模型的 messages 上，不写入会话 history。
func agentsNote() (openai.ChatCompletionMessage, bool) {
	b, err := os.ReadFile(agentsFile)
	if err != nil || len(b) == 0 {
		return openai.ChatCompletionMessage{}, false
	}
	text := string(b)
	if utf8.RuneCountInString(text) > agentsLimit {
		text = string([]rune(text)[:agentsLimit]) + "\n\n[AGENTS.md 已截断，原文超过约 12000 字]"
	}
	return openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleSystem,
		Content: "以下是工作区 AGENTS.md，请遵守：\n" + text,
	}, true
}

// outbound 在 history 前附上 AGENTS.md（若有）与已加载技能（若有），再接上只存在于本次请求的尾部消息。
func (c *Chat) outbound(history []openai.ChatCompletionMessage, tail ...openai.ChatCompletionMessage) []openai.ChatCompletionMessage {
	extra := 2
	out := make([]openai.ChatCompletionMessage, 0, len(history)+len(tail)+extra)
	if note, ok := agentsNote(); ok {
		out = append(out, note)
	}
	if c != nil && c.skillNote != "" {
		out = append(out, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: c.skillNote,
		})
	}
	out = append(out, history...)
	out = append(out, tail...)
	return out
}
