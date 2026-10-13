package chat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

func TestDelegateIsolatedFromHistory(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":"子任务结果"}}],"usage":{"prompt_tokens":4,"completion_tokens":2,"total_tokens":6}}`)
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "k", "m")
	c.SetHistory([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "parent-secret"}})
	got, err := c.Delegate(context.Background(), "做一件小事")
	if err != nil {
		t.Fatal(err)
	}
	if got != "子任务结果" {
		t.Fatalf("got %q", got)
	}
	if strings.Contains(body, "parent-secret") {
		t.Fatalf("delegate saw parent history: %s", body)
	}
	if strings.Contains(body, `"tools"`) {
		t.Fatalf("delegate sent tools: %s", body)
	}
	h := c.History()
	if len(h) != 1 || h[0].Content != "parent-secret" {
		t.Fatalf("history changed: %+v", h)
	}
	if c.Usage().Cumulative.Completion != 2 {
		t.Fatalf("usage %+v", c.Usage().Cumulative)
	}
}
