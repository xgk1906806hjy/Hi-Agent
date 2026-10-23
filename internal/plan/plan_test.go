package plan

import (
	"strings"
	"testing"
)

func TestApplyKeepsOneInProgress(t *testing.T) {
	Reset()
	t.Cleanup(Reset)

	text, err := Apply([]Item{
		{ID: "a", Content: "读代码", Status: InProgress},
		{ID: "b", Content: "写测试", Status: Pending},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "● [a]") || !strings.Contains(text, "○ [b]") {
		t.Fatalf("%s", text)
	}

	if _, err := Apply([]Item{{ID: "b", Status: InProgress}}); err != nil {
		t.Fatal(err)
	}
	got := List()
	if got[0].Status != Pending || got[1].Status != InProgress {
		t.Fatalf("want b in progress: %+v", got)
	}

	if _, err := Apply([]Item{
		{ID: "a", Status: InProgress},
		{ID: "b", Status: InProgress},
	}); err != nil {
		t.Fatal(err)
	}
	got = List()
	if got[0].Status != Pending || got[1].Status != InProgress {
		t.Fatalf("last in_progress wins: %+v", got)
	}

	if _, err := Apply([]Item{{ID: "c", Status: Pending}}); err == nil {
		t.Fatal("new item without content")
	}
	if _, err := Apply([]Item{{ID: "a b", Content: "x"}}); err == nil {
		t.Fatal("bad id")
	}
}

func TestActivatePersistsPerSession(t *testing.T) {
	Reset()
	t.Cleanup(Reset)
	dir := t.TempDir()
	if err := Activate(dir, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := Apply([]Item{{ID: "t", Content: "one", Status: Pending}}); err != nil {
		t.Fatal(err)
	}
	if err := Activate(dir, "b"); err != nil {
		t.Fatal(err)
	}
	if len(List()) != 0 {
		t.Fatalf("new session: %+v", List())
	}
	if err := Activate(dir, "a"); err != nil {
		t.Fatal(err)
	}
	got := List()
	if len(got) != 1 || got[0].Content != "one" {
		t.Fatalf("%+v", got)
	}
	if err := Move(dir, "a", "a2"); err != nil {
		t.Fatal(err)
	}
	if err := Drop(dir, "b"); err != nil {
		t.Fatal(err)
	}
	if err := Activate(dir, "a2"); err != nil {
		t.Fatal(err)
	}
	if List()[0].Content != "one" {
		t.Fatalf("%+v", List())
	}
}
