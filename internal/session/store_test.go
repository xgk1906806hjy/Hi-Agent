package session

import (
	"os"
	"path/filepath"
	"testing"

	openai "github.com/sashabaranov/go-openai"
)

// 覆盖 Data 落盘往返、Rename default、NewID 长度。

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	d := NewData()
	d.PutMessages(DefaultID, []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "hello"},
	})
	if err := d.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, Path())); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	rec := loaded.Get(DefaultID)
	if rec == nil || len(rec.Messages) != 1 || rec.Messages[0].Content != "hello" {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestRenameDefault(t *testing.T) {
	d := NewData()
	d.PutMessages(DefaultID, []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleUser, Content: "x"},
	})
	if err := d.Rename(DefaultID, "abcd1234"); err != nil {
		t.Fatal(err)
	}
	if d.Current != "abcd1234" {
		t.Fatalf("current=%s", d.Current)
	}
	if d.Get(DefaultID) != nil {
		t.Fatal("old id should be gone")
	}
	if d.Get("abcd1234") == nil || len(d.Get("abcd1234").Messages) != 1 {
		t.Fatal("rename lost messages")
	}
}

func TestNewIDLength(t *testing.T) {
	id, err := NewID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 8 {
		t.Fatalf("len=%d id=%s", len(id), id)
	}
}
