package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSearchLiteralAndFilters(t *testing.T) {
	dir := inTempDir(t)
	writeTestFile(t, filepath.Join(dir, "a.go"), "package main\nfunc Hello() {}\n")
	writeTestFile(t, filepath.Join(dir, "sub", "b.go"), "// TODO\nfunc Hello() {}\n")
	writeTestFile(t, filepath.Join(dir, ".git", "secret.go"), "func Hello() {}\n")
	writeTestFile(t, filepath.Join(dir, "node_modules", "x.js"), "Hello\n")
	writeTestFile(t, filepath.Join(dir, "dist", "app.js"), "Hello\n")
	writeTestFile(t, filepath.Join(dir, "bin.dat"), "\x00Hello")

	out, err := runSearch(`{"pattern":"Hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"a.go:2:", "sub/b.go:2:", "匹配到 2 处"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in:\n%s", want, out)
		}
	}
	for _, bad := range []string{".git/", "node_modules", "dist/", "bin.dat"} {
		if strings.Contains(out, bad) {
			t.Fatalf("should skip %q:\n%s", bad, out)
		}
	}

	// path 限定范围。
	out, err = runSearch(`{"pattern":"Hello","path":"sub"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sub/b.go:2:") || strings.Contains(out, "a.go") {
		t.Fatalf("path scope:\n%s", out)
	}

	// 正则。
	out, err = runSearch(`{"pattern":"func\\s+Hello","regex":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "a.go:2:") || !strings.Contains(out, "sub/b.go:2:") {
		t.Fatalf("regex:\n%s", out)
	}

	// 无匹配 / 非法正则。
	if out, _ = runSearch(`{"pattern":"不存在的符号"}`); !strings.Contains(out, "没有匹配") {
		t.Fatalf("no match: %s", out)
	}
	if out, _ = runSearch(`{"pattern":"(","regex":true}`); !strings.Contains(out, "正则无效") {
		t.Fatalf("bad regex: %s", out)
	}
}

func TestSearchLimit(t *testing.T) {
	dir := inTempDir(t)
	var b strings.Builder
	for i := 0; i < maxSearchResults+20; i++ {
		b.WriteString("hit\n")
	}
	writeTestFile(t, filepath.Join(dir, "many.txt"), b.String())

	out, err := runSearch(`{"pattern":"hit"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "至少 200 处") || !strings.Contains(out, "已达上限") {
		t.Fatalf("limit:\n%s", out)
	}
}
