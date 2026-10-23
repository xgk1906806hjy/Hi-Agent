package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSearchList(t *testing.T) {
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	if _, err := Write("  "); err == nil {
		t.Fatal("empty content")
	}
	a, err := Write("喜欢用 Go")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Write("下午喝茶"); err != nil {
		t.Fatal(err)
	}
	got, err := Search("go")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != a.ID {
		t.Fatalf("%+v", got)
	}
	all, err := List()
	if err != nil || len(all) != 2 {
		t.Fatalf("%v %+v", err, all)
	}
	if _, err := os.Stat(filepath.Join(dir, ".geekagent", "memory.json")); err != nil {
		t.Fatal(err)
	}
}
