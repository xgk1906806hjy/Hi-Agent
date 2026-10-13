package session_test

import (
	"testing"

	"github.com/xgk1906806hjy/Hi-Agent/internal/chat"
	"github.com/xgk1906806hjy/Hi-Agent/internal/session"

	openai "github.com/sashabaranov/go-openai"
)

// 验证切换会话时 history 互不泄漏：新建清空、Open 恢复各自消息。

func TestManagerSwitchNoLeak(t *testing.T) {
	c := chat.New("http://localhost", "k", "m")
	m := session.NewManager(c, session.NewData())
	c.SetHistory([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "A"}})
	m.SyncFromChat()

	id, err := m.NewSession("")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.History()) != 0 {
		t.Fatal("new session should be empty")
	}

	c.SetHistory([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "B"}})
	m.SyncFromChat()

	if err := m.Open(session.DefaultID); err != nil {
		t.Fatal(err)
	}
	h := c.History()
	if len(h) != 1 || h[0].Content != "A" {
		t.Fatalf("leak or wrong hist: %+v", h)
	}

	if err := m.Open(id); err != nil {
		t.Fatal(err)
	}
	h = c.History()
	if len(h) != 1 || h[0].Content != "B" {
		t.Fatalf("want B: %+v", h)
	}
}

func TestBriefsTitleRenameDelete(t *testing.T) {
	c := chat.New("http://localhost", "k", "m")
	m := session.NewManager(c, session.NewData())
	c.SetHistory([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "帮我\n看看  项目结构"}})

	b := m.SessionBriefs()
	if len(b) != 1 || b[0].Title != "帮我 看看 项目结构" || b[0].Messages != 1 {
		t.Fatalf("briefs %+v", b)
	}

	if err := m.Rename(session.DefaultID, "work"); err != nil {
		t.Fatal(err)
	}
	if m.Current() != "work" {
		t.Fatalf("current %s", m.Current())
	}
	if err := m.Rename("work", "bad id"); err == nil {
		t.Fatal("want invalid id error")
	}

	other, err := m.NewSession("other")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Delete(other); err != nil {
		t.Fatal(err)
	}
	if m.Current() != "work" {
		t.Fatalf("after delete current %s", m.Current())
	}
	if h := c.History(); len(h) != 1 {
		t.Fatalf("history not restored: %+v", h)
	}
	if err := m.Delete("work"); err != nil {
		t.Fatal(err)
	}
	if m.Current() != session.DefaultID || len(c.History()) != 0 {
		t.Fatalf("want empty default, got %s %+v", m.Current(), c.History())
	}
}

func TestTitleTruncates(t *testing.T) {
	got := session.TitleFromText("一二三四五六七八九十一二三四五六七八九十一二三四五六")
	if []rune(got)[24] != '…' {
		t.Fatalf("%q", got)
	}
}
