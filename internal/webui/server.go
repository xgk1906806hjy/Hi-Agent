package webui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/xgk1906806hjy/Hi-Agent/internal/gui"
)

const (
	defaultAddr    = "127.0.0.1:8787"
	maxRequestSize = 64 << 10
)

type sseEvent struct {
	name string
	data []byte
}

// Server 把 gui.Emitter 广播给所有已连接的浏览器，并提供提交/停止/对话框 HTTP 接口。
type Server struct {
	adapter *gui.Adapter

	mu        sync.Mutex
	clients   map[chan sseEvent]struct{}
	httpSrv   *http.Server
	addr      string
	stopped   bool
	closeOnce sync.Once
}

func newServer() *Server {
	return &Server{clients: map[chan sseEvent]struct{}{}}
}

// Emit 实现 gui.Emitter：编码载荷后广播给所有 SSE 连接。
// 发送采用非阻塞队列，慢客户端丢事件而不阻塞 REPL。
func (s *Server) Emit(event string, payload any) {
	data := []byte("null")
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			data, _ = json.Marshal(gui.TextPayload{Text: "Web 事件编码失败：" + err.Error()})
			event = gui.EvError
		} else {
			data = b
		}
	}
	ev := sseEvent{name: event, data: data}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.clients {
		select {
		case ch <- ev:
		default:
		}
	}
}

// Run 启动 HTTP 服务并阻塞到 Shutdown；由 gui.Adapter 的 DesktopRunner.Run 调用。
func (s *Server) Run() {
	if s.adapter == nil {
		return
	}
	addr := strings.TrimSpace(os.Getenv("GEEKAGENT_WEB_ADDR"))
	if addr == "" {
		addr = defaultAddr
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		s.adapter.Error("Web 服务启动失败：" + err.Error())
		return
	}

	srv := &http.Server{Handler: s.routes()}
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		_ = ln.Close()
		return
	}
	s.httpSrv = srv
	s.addr = ln.Addr().String()
	s.mu.Unlock()

	fmt.Fprintf(os.Stderr, "[web] Hi-agent Web: http://%s\n", ln.Addr().String())
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.adapter.Error("Web 服务异常退出：" + err.Error())
	}
}

// Shutdown 关闭监听与所有 SSE 连接；重复调用无操作。
func (s *Server) Shutdown() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		clients := s.clients
		s.clients = map[chan sseEvent]struct{}{}
		srv := s.httpSrv
		s.mu.Unlock()

		for ch := range clients {
			close(ch)
		}
		if srv != nil {
			_ = srv.Shutdown(context.Background())
		}
	})
}

// Addr 返回实际监听地址（Run 启动后可用），供测试与提示信息。
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/submit", s.handleSubmit)
	mux.HandleFunc("/api/stop", s.handleStop)
	mux.HandleFunc("/api/answer", s.handleAnswer)
	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	b, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Web 页面未打包", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.adapter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "adapter 未初始化"})
		return
	}
	writeJSON(w, http.StatusOK, s.adapter.Snapshot())
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		Text string `json:"text"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if s.adapter == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "adapter 未初始化"})
		return
	}
	s.adapter.Submit(in.Text)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.adapter != nil {
		s.adapter.Stop()
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleAnswer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var in struct {
		ID    string `json:"id"`
		Value int    `json:"value"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	if s.adapter != nil {
		s.adapter.Answer(in.ID, in.Value)
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	if s.adapter == nil {
		http.Error(w, "adapter 未初始化", http.StatusServiceUnavailable)
		return
	}

	ch := make(chan sseEvent, 256)
	s.mu.Lock()
	if s.stopped {
		s.mu.Unlock()
		http.Error(w, "server stopped", http.StatusServiceUnavailable)
		return
	}
	s.clients[ch] = struct{}{}
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.clients, ch)
		s.mu.Unlock()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, ": connected\n\n")
	flusher.Flush()

	// 先登记客户端再 Attach，确保启动前排队的 banner/status 不丢。
	s.adapter.Attach(s)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.name, ev.data)
			flusher.Flush()
		case <-ticker.C:
			_, _ = io.WriteString(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求 JSON 无效：" + err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
