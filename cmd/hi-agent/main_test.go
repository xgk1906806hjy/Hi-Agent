package main

import (
	"strings"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestHistoryTurnsPairsToolResults(t *testing.T) {
	msgs := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "看看"},
		{Role: openai.ChatMessageRoleAssistant, ToolCalls: []openai.ToolCall{
			{ID: "c1", Function: openai.FunctionCall{Name: "read_file"}},
			{ID: "c2", Function: openai.FunctionCall{Name: "list_dir"}},
		}},
		{Role: openai.ChatMessageRoleTool, ToolCallID: "c1", Content: "line1\n" + strings.Repeat("x", 400)},
		{Role: openai.ChatMessageRoleAssistant, Content: "好的"},
	}
	got := historyTurns(msgs)
	if len(got) != 4 {
		t.Fatalf("turns %+v", got)
	}
	if !strings.HasPrefix(got[1].Text, "[调用工具 read_file → line1 xxx") || !strings.HasSuffix(got[1].Text, "…]") {
		t.Fatalf("tool1 %q", got[1].Text)
	}
	if got[2].Text != "[调用工具 list_dir]" {
		t.Fatalf("tool2 %q", got[2].Text)
	}
}

func TestOpenDirArg(t *testing.T) {
	cases := map[string]string{
		"/open abc":                   "",
		"/open abc D:\\my project":    "D:\\my project",
		`/open abc "E:\work space\x"`: `E:\work space\x`,
	}
	for line, want := range cases {
		if got := openDirArg(line, "abc"); got != want {
			t.Fatalf("%q → %q, want %q", line, got, want)
		}
	}
}
