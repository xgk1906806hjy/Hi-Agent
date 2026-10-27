package webui

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xgk1906806hjy/Hi-Agent/internal/gui"
	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
)

func TestKindWebRegisteredAndCreatesAdapter(t *testing.T) {
	u, err := ui.Create(ui.KindWeb, ui.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := u.(ui.DesktopRunner); !ok {
		t.Fatalf("web UI should implement DesktopRunner: %T", u)
	}
	u.Close()
}

func TestHTTPIndexSnapshotSubmit(t *testing.T) {
	ad := gui.NewAdapter(gui.Config{StatusFn: func() ui.Status {
		return ui.Status{Model: "m", Session: "s", WorkDir: "w"}
	}})
	s := newServer()
	s.adapter = ad
	ad.Refresh()
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "EventSource") {
		t.Fatalf("index status=%d body=%q", resp.StatusCode, body)
	}

	resp, err = http.Get(ts.URL + "/api/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	var snap gui.Snapshot
	if err := json.NewDecoder(resp.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if snap.Status.Model != "m" || snap.Status.Session != "s" {
		t.Fatalf("snapshot = %#v", snap)
	}

	lines := make(chan string, 1)
	go func() {
		line, _ := ad.ReadLine(ui.MainPrompt)
		lines <- line
	}()
	resp, err = http.Post(ts.URL+"/api/submit", "application/json", strings.NewReader(`{"text":"hello"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("submit status = %d", resp.StatusCode)
	}
	select {
	case line := <-lines:
		if line != "hello" {
			t.Fatalf("line = %q", line)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for submitted line")
	}
}

func TestSSEEventEncoding(t *testing.T) {
	s := newServer()
	ch := make(chan sseEvent, 1)
	s.clients[ch] = struct{}{}
	s.Emit(gui.EvInfo, gui.TextPayload{Text: "hi"})
	select {
	case ev := <-ch:
		if ev.name != gui.EvInfo {
			t.Fatalf("event = %q", ev.name)
		}
		var p gui.TextPayload
		if err := json.Unmarshal(ev.data, &p); err != nil {
			t.Fatal(err)
		}
		if p.Text != "hi" {
			t.Fatalf("payload = %#v", p)
		}
	case <-time.After(time.Second):
		t.Fatal("no event")
	}
}

func TestSSEStream(t *testing.T) {
	ad := gui.NewAdapter(gui.Config{})
	s := newServer()
	s.adapter = ad
	ts := httptest.NewServer(s.routes())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content-type = %q", resp.Header.Get("Content-Type"))
	}

	// 读取连接注释，等 Attach 完成后发事件。
	sc := bufio.NewScanner(resp.Body)
	if !sc.Scan() || !strings.Contains(sc.Text(), "connected") {
		t.Fatalf("first line = %q", sc.Text())
	}
	ad.Info("hello sse")
	for sc.Scan() {
		if strings.Contains(sc.Text(), "event: "+gui.EvInfo) {
			// 下一行是 data。
			if sc.Scan() && strings.Contains(sc.Text(), `"hello sse"`) {
				return
			}
		}
	}
	t.Fatal("no SSE info event")
}
