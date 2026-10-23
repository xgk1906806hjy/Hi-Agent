package chat

import (
	"os"
	"strings"
	"testing"
)

func TestAgentsNoteTruncatesAndSkipsMissing(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if _, ok := agentsNote(); ok {
		t.Fatal("missing file should be skipped")
	}
	body := strings.Repeat("甲", agentsLimit+10)
	if err := os.WriteFile("AGENTS.md", []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	note, ok := agentsNote()
	if !ok {
		t.Fatal("expected note")
	}
	if !strings.Contains(note.Content, "[AGENTS.md 已截断") {
		t.Fatal("expected truncation note")
	}
	msgs := (&Chat{}).outbound(nil)
	if len(msgs) != 1 || msgs[0].Content == "" {
		t.Fatalf("%+v", msgs)
	}
}
