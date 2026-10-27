package rag

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func inTempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	return dir
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAddSearchAndSourceNumbers(t *testing.T) {
	dir := inTempDir(t)
	writeFile(t, filepath.Join(dir, "docs", "guide.md"),
		"# 指南\nGo 后端服务使用 GoFrame。\n桌面客户端使用 Wails。\n回答应保留来源段号。\n")

	res, err := Add("docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chunks == 0 || len(res.Sources) != 1 {
		t.Fatalf("unexpected add result: %+v", res)
	}
	if res.Sources[0].Source != "docs/guide.md" {
		t.Fatalf("source = %q", res.Sources[0].Source)
	}

	hits, err := Search("GoFrame 后端", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	if hits[0].Source != "docs/guide.md" || hits[0].Seq < 1 {
		t.Fatalf("hit lacks source/seq: %+v", hits[0])
	}
	if !strings.Contains(hits[0].Content, "GoFrame") {
		t.Fatalf("unexpected content: %q", hits[0].Content)
	}

	hits, err = Search("咖啡", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("unexpected unrelated hit: %+v", hits)
	}

	lines, err := Lines()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "RAG 索引") || !strings.Contains(joined, "docs/guide.md") {
		t.Fatalf("lines = %q", joined)
	}
	stats, err := Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources != 1 || stats.Chunks != res.Chunks {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestAddReplacesSourceAndDirectoryScope(t *testing.T) {
	dir := inTempDir(t)
	writeFile(t, filepath.Join(dir, "docs", "a.md"), "alpha old\n")
	writeFile(t, filepath.Join(dir, "docs", "b.md"), "bravo keep\n")
	if _, err := Add("docs"); err != nil {
		t.Fatal(err)
	}

	writeFile(t, filepath.Join(dir, "docs", "a.md"), "gamma new\n")
	if _, err := Add("docs/a.md"); err != nil {
		t.Fatal(err)
	}
	oldHits, err := Search("alpha", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(oldHits) != 0 {
		t.Fatalf("old content should be replaced: %+v", oldHits)
	}
	newHits, err := Search("gamma", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(newHits) == 0 {
		t.Fatal("new content missing")
	}

	if err := os.Remove(filepath.Join(dir, "docs", "b.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := Add("docs"); err != nil {
		t.Fatal(err)
	}
	stats, err := Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Sources != 1 {
		t.Fatalf("deleted source should be removed: %+v", stats)
	}
	lines, err := Lines()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(lines, "\n"), "docs/b.md") {
		t.Fatalf("stale source listed: %v", lines)
	}
}

func TestDirectorySkipsNoiseBinaryAndLargeFiles(t *testing.T) {
	dir := inTempDir(t)
	writeFile(t, filepath.Join(dir, "docs", "a.md"), "A 文档内容\n")
	writeFile(t, filepath.Join(dir, "docs", "sub", "b.txt"), "B 文档内容\n")
	writeFile(t, filepath.Join(dir, "docs", ".git", "secret.md"), "secret\n")
	writeFile(t, filepath.Join(dir, "docs", "node_modules", "x.md"), "noise\n")
	writeFile(t, filepath.Join(dir, "docs", "bin.dat"), "\x00binary")
	writeFile(t, filepath.Join(dir, "docs", "large.md"), strings.Repeat("x", maxFileBytes+1))

	res, err := Add("docs")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Sources) != 2 {
		t.Fatalf("sources = %+v", res.Sources)
	}
	if res.Skipped < 2 {
		t.Fatalf("skipped = %d", res.Skipped)
	}
	hits, err := Search("文档内容", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits = %+v", hits)
	}
	for _, hit := range hits {
		if strings.Contains(hit.Source, ".git") || strings.Contains(hit.Source, "node_modules") {
			t.Fatalf("noise source indexed: %s", hit.Source)
		}
	}
}

func TestChunkTextLimitAndSequence(t *testing.T) {
	long := strings.Repeat("测", chunkMaxRunes*2+50)
	chunks := chunkText(long)
	if len(chunks) != 3 {
		t.Fatalf("chunks = %d", len(chunks))
	}
	for _, chunk := range chunks {
		if n := utf8.RuneCountInString(chunk); n > chunkMaxRunes {
			t.Fatalf("chunk too large: %d", n)
		}
	}
	if chunks[0] == "" || chunks[2] == "" {
		t.Fatal("unexpected empty chunk")
	}

	dir := inTempDir(t)
	writeFile(t, filepath.Join(dir, "long.md"), long)
	res, err := Add("long.md")
	if err != nil {
		t.Fatal(err)
	}
	if res.Chunks != 3 {
		t.Fatalf("res = %+v", res)
	}
	hits, err := Search("测测", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || hits[0].Seq < 1 {
		t.Fatalf("hits = %+v", hits)
	}
}

func TestMissingCorruptAndClear(t *testing.T) {
	dir := inTempDir(t)
	hits, err := Search("anything", 5)
	if err != nil || len(hits) != 0 {
		t.Fatalf("missing index: %v %+v", err, hits)
	}
	lines, err := Lines()
	if err != nil || !strings.Contains(strings.Join(lines, "\n"), "暂无") {
		t.Fatalf("missing lines: %v %v", err, lines)
	}

	writeFile(t, filepath.Join(dir, filepath.FromSlash(fileRel)), "{bad")
	if _, err := Search("anything", 5); err == nil {
		t.Fatal("corrupt index should fail")
	}
	before, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(fileRel)))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Add("."); err == nil {
		t.Fatal("Add must not overwrite corrupt index")
	}
	after, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(fileRel)))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("corrupt index changed")
	}

	if err := Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(fileRel))); !os.IsNotExist(err) {
		t.Fatalf("index still exists: %v", err)
	}
}

func TestLineScopeUsesSlashPaths(t *testing.T) {
	dir := inTempDir(t)
	writeFile(t, filepath.Join(dir, "docs", "a.md"), "路径测试\n")
	abs := filepath.Join(dir, "docs", "a.md")
	res, err := Add(abs)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Sources) != 1 || res.Sources[0].Source != "docs/a.md" {
		t.Fatalf("absolute add source = %+v", res.Sources)
	}
}
