package memory

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func useTempCWD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	return dir
}

func TestRecallRanksRelevantChunksAndBuildsNote(t *testing.T) {
	useTempCWD(t)

	if _, err := Write("用户喜欢用 Go 写后端服务，偏好简洁实现。"); err != nil {
		t.Fatal(err)
	}
	if _, err := Write("项目桌面界面使用 Wails 和 React。"); err != nil {
		t.Fatal(err)
	}
	if _, err := Write("用户通常下午喝茶，晚上写代码。"); err != nil {
		t.Fatal(err)
	}

	hits, err := Recall("Go 后端", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || !strings.Contains(hits[0].Content, "Go") || !strings.Contains(hits[0].Content, "后端") {
		t.Fatalf("Go 后端 should rank the Go memory first: %+v", hits)
	}

	hits, err = Recall("桌面界面", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 || !strings.Contains(hits[0].Content, "桌面") {
		t.Fatalf("桌面界面 should rank the desktop memory first: %+v", hits)
	}

	hits, err = Recall("咖啡", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("unrelated query should not recall: %+v", hits)
	}

	// 查询侧多字 CJK 使用双字 gram，「我」这类单字不应把无关记忆带进来。
	hits, err = Recall("我的问题", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("single-char overlap should not recall: %+v", hits)
	}

	note := RecallNote("Go 后端")
	if !strings.Contains(note, "【项目长期记忆·自动召回】") || !strings.Contains(note, "Go") {
		t.Fatalf("unexpected note: %q", note)
	}
	if got := strings.Count(note, "\n- ") + 1; got > recallLimit {
		t.Fatalf("note has %d items, want <= %d", got, recallLimit)
	}
}

func TestRecallReturnsAtMostTopFive(t *testing.T) {
	useTempCWD(t)
	for i := 0; i < 10; i++ {
		if _, err := Write(fmt.Sprintf("项目偏好第 %d 条", i)); err != nil {
			t.Fatal(err)
		}
	}
	hits, err := Recall("项目", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != recallLimit {
		t.Fatalf("got %d hits, want %d", len(hits), recallLimit)
	}
}

func TestChunkTextCapsChunkSize(t *testing.T) {
	text := strings.Repeat("项目需要简洁实现。", 200)
	chunks := chunkText(text)
	if len(chunks) < 2 {
		t.Fatalf("long memory should be split: %d chunk(s)", len(chunks))
	}
	for i, chunk := range chunks {
		if n := utf8.RuneCountInString(chunk); n > chunkMaxRunes {
			t.Fatalf("chunk %d has %d runes, want <= %d", i, n, chunkMaxRunes)
		}
	}
}

func TestRecallMissingAndCorruptFile(t *testing.T) {
	useTempCWD(t)

	hits, err := Recall("项目", 5)
	if err != nil || len(hits) != 0 {
		t.Fatalf("missing file should be empty: %v %+v", err, hits)
	}
	if note := RecallNote("项目"); note != "" {
		t.Fatalf("missing file note = %q", note)
	}

	if err := os.MkdirAll(".geekagent", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".geekagent/memory.json", []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Recall("项目", 5); err == nil {
		t.Fatal("corrupt file should return an error")
	}
	if note := RecallNote("项目"); note != "" {
		t.Fatalf("corrupt file should not inject memory: %q", note)
	}
}
