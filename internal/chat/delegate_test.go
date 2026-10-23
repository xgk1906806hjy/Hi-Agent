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

func TestDelegateSeesHistoryAndKeepsParent(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"子任务结果\"}}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":2,\"total_tokens\":6}}\n\n"+
			"data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "k", "m")
	c.SetHistory([]openai.ChatCompletionMessage{{Role: openai.ChatMessageRoleUser, Content: "parent-secret"}})
	got, err := c.Delegate(context.Background(), "做一件小事", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "子任务结果" {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(body, "parent-secret") {
		t.Fatalf("delegate should see parent history: %s", body)
	}
	if !strings.Contains(body, `"tools"`) {
		t.Fatalf("delegate should send tools: %s", body)
	}
	h := c.History()
	if len(h) != 1 || h[0].Content != "parent-secret" {
		t.Fatalf("history changed: %+v", h)
	}
	if c.Usage().Cumulative.Completion != 2 {
		t.Fatalf("usage %+v", c.Usage().Cumulative)
	}
}

func TestDelegateRejectsNested(t *testing.T) {
	delegateDepth.Add(1)
	defer delegateDepth.Add(-1)
	c := New("http://127.0.0.1:1/v1", "k", "m")
	got, err := c.Delegate(context.Background(), "再委派", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "不能再委派" {
		t.Fatalf("got %q", got)
	}
}
