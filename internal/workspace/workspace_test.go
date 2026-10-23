package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRememberAddsNewFirstAndKeepsOrder(t *testing.T) {
	reg := filepath.Join(t.TempDir(), "projects.json")
	a, b := filepath.Join("E:", "a"), filepath.Join("E:", "b")
	if err := rememberIn(reg, a); err != nil {
		t.Fatal(err)
	}
	if err := rememberIn(reg, b); err != nil {
		t.Fatal(err)
	}
	// 已登记：忽略大小写去重，且不改变原位置（点开不置顶）。
	if err := rememberIn(reg, filepath.Join("E:", "A")); err != nil {
		t.Fatal(err)
	}
	got := loadFrom(reg)
	if len(got) != 2 || !SamePath(got[0], b) || !SamePath(got[1], a) {
		t.Fatalf("got %v", got)
	}
	if err := removeIn(reg, b); err != nil {
		t.Fatal(err)
	}
	if got := loadFrom(reg); len(got) != 1 {
		t.Fatalf("after remove %v", got)
	}
}

func TestListSessions(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".geekagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"current":"b","sessions":{"b":{"id":"b","updatedAt":"2026-01-02T03:04:05Z","messages":[{"role":"user","content":"hi"}]},"a":{"id":"a","messages":[]}}}`
	if err := os.WriteFile(filepath.Join(dir, ".geekagent", "sessions.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ListSessions(dir)
	if len(got) != 2 || got[0].ID != "a" || got[1].Messages != 1 || !got[1].Current {
		t.Fatalf("got %+v", got)
	}
	if got[1].Title != "hi" || got[1].UpdatedAt.Year() != 2026 || got[0].Title != "" {
		t.Fatalf("meta %+v", got)
	}
	if ListSessions(t.TempDir()) != nil {
		t.Fatal("missing file should be empty")
	}
}
