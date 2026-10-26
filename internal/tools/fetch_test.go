package tools

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTMLToText(t *testing.T) {
	src := `<html><head><title>示例 &amp; 标题</title><style>.x{color:red}</style></head>
<body><h1>Hello</h1><script>alert("x")</script><p>第一行 &amp; 第二行</p><br>尾部<!-- 注释 --></body></html>`
	title, text := htmlToText(src)
	if title != "示例 & 标题" {
		t.Fatalf("title %q", title)
	}
	for _, want := range []string{"Hello", "第一行 & 第二行", "尾部"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in:\n%s", want, text)
		}
	}
	for _, bad := range []string{"alert", "color:red", "注释", "<h1>", "示例 & 标题"} {
		if strings.Contains(text, bad) {
			t.Fatalf("should drop %q:\n%s", bad, text)
		}
	}
}

func TestFetchHTML(t *testing.T) {
	inTempDir(t)
	var asked string
	SetConfirm(func(p string) bool { asked = p; return true })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, `<html><head><title>文档</title></head><body><h2>标题</h2><p>正文 &amp; 内容</p></body></html>`)
	}))
	defer srv.Close()

	out, err := runFetch(`{"url":"` + srv.URL + `"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"HTTP 200", "标题：文档", "正文 & 内容"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	if !strings.Contains(asked, srv.URL) {
		t.Fatalf("confirm prompt: %q", asked)
	}
}

func TestFetchCancelAndErrors(t *testing.T) {
	inTempDir(t)
	SetConfirm(func(string) bool { return false })
	out, err := runFetch(`{"url":"https://example.com"}`)
	if err != nil || !strings.Contains(out, "已取消抓取") {
		t.Fatalf("cancel: %q %v", out, err)
	}

	SetConfirm(func(string) bool { return true })
	if out, _ = runFetch(`{"url":"file:///etc/passwd"}`); !strings.Contains(out, "只支持 http/https") {
		t.Fatalf("scheme: %s", out)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer srv.Close()
	if out, _ = runFetch(`{"url":"` + srv.URL + `"}`); !strings.Contains(out, "HTTP 404") {
		t.Fatalf("status: %s", out)
	}
}
