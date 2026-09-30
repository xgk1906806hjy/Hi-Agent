package session_test

import (
	"testing"

	"hi-agent/internal/chat"
	"hi-agent/internal/session"

	openai "github.com/sashabaranov/go-openai"
)

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
