package chat

import "testing"

// 覆盖护栏环境变量：收缩默认值、显式关闭、回复预算 0=关闭。

func TestShrinkToolsAfterDefault(t *testing.T) {
	t.Setenv("GEEKAGENT_MAX_TOOL_TURNS", "8")
	t.Setenv("GEEKAGENT_SHRINK_TOOLS_AFTER", "")
	got := shrinkToolsAfter()
	if got != 6 {
		t.Fatalf("default shrinkAfter=%d want 6", got)
	}
}

func TestShrinkToolsAfterDisabled(t *testing.T) {
	t.Setenv("GEEKAGENT_SHRINK_TOOLS_AFTER", "0")
	if shrinkToolsAfter() != 0 {
		t.Fatal("expected 0 to disable")
	}
}

func TestMaxReplyCharsZeroDisables(t *testing.T) {
	t.Setenv("GEEKAGENT_MAX_REPLY_CHARS", "0")
	if maxReplyChars() != 0 {
		t.Fatal("expected 0")
	}
}
