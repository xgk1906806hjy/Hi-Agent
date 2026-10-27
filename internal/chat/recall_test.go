package chat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestStreamReplyInjectsRecallOnlyForCurrentTurn(t *testing.T) {
	var mu sync.Mutex
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, string(b))
		mu.Unlock()
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"}}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":1,\"total_tokens\":5}}\n\n"+
			"data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "k", "m")
	c.SetRecall(func(query string) string {
		if strings.Contains(query, "记忆") {
			return "【项目长期记忆·自动召回】\n- 用户喜欢 Go"
		}
		return ""
	})

	var firstDeltas []string
	if err := c.StreamReply(context.Background(), "请带上记忆", func(delta string) {
		firstDeltas = append(firstDeltas, delta)
	}); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 {
		t.Fatalf("got %d requests, want 1", len(bodies))
	}
	if !strings.Contains(bodies[0], "用户喜欢 Go") {
		t.Fatalf("first request should contain recall note: %s", bodies[0])
	}
	foundProgress := false
	for _, delta := range firstDeltas {
		if strings.HasPrefix(delta, "\n[主动记忆") {
			foundProgress = true
		}
	}
	if !foundProgress {
		t.Fatalf("missing recall progress line: %q", firstDeltas)
	}

	if err := c.StreamReply(context.Background(), "普通问题", func(string) {}); err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 2 {
		t.Fatalf("got %d requests, want 2", len(bodies))
	}
	if strings.Contains(bodies[1], "用户喜欢 Go") {
		t.Fatalf("recall note leaked into next turn: %s", bodies[1])
	}
}

func TestDelegateInheritsRecall(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"子任务结果\"}}]}\n\n"+
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"+
			"data: [DONE]\n\n")
	}))
	defer srv.Close()

	c := New(srv.URL+"/v1", "k", "m")
	c.SetRecall(func(string) string { return "【项目长期记忆·自动召回】\n- 子任务也读记忆" })
	if _, err := c.Delegate(context.Background(), "做一件小事", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "子任务也读记忆") {
		t.Fatalf("delegate request should contain recall note: %s", body)
	}
}
