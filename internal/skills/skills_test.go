package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, dir, body string) {
	t.Helper()
	p := filepath.Join(root, dir)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p, FileName), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParseFrontMatter(t *testing.T) {
	s := parse("dir-name", "---\nname: code-review\ndescription: \"审查代码\"\ntools: read, glob\n---\n先读文件。\n")
	if s.Name != "code-review" || s.Description != "审查代码" {
		t.Fatalf("meta %+v", s)
	}
	if len(s.Tools) != 2 || s.Tools[0] != "read" || s.Tools[1] != "glob" {
		t.Fatalf("tools %v", s.Tools)
	}
	if s.Body != "先读文件。" {
		t.Fatalf("body %q", s.Body)
	}

	// 无 frontmatter：名称回退目录名，正文即全文。
	bare := parse("explore", "只看不写。\n")
	if bare.Name != "explore" || bare.Body != "只看不写。" || len(bare.Tools) != 0 {
		t.Fatalf("bare %+v", bare)
	}

	// 多行与方括号写法。
	list := parse("x", "---\ntools:\n  - read\n  - write\ntools2: y\n---\nbody")
	if len(list.Tools) != 2 || list.Tools[1] != "write" {
		t.Fatalf("list tools %v", list.Tools)
	}
	bracket := parse("y", "---\ntools: [read, glob]\n---\n")
	if len(bracket.Tools) != 2 || bracket.Tools[1] != "glob" {
		t.Fatalf("bracket tools %v", bracket.Tools)
	}
}

func TestScanAndCompose(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "review", "---\nname: code-review\ntools: read, glob\n---\n审查要点")
	writeSkill(t, root, "plain", "无 frontmatter")
	writeSkill(t, root, "dup", "---\nname: code-review\n---\n重复名应忽略")
	writeSkill(t, root, "bad", "---\ntools: read\n---\n再看一次")

	got, err := scanDir(root)
	if err != nil {
		t.Fatal(err)
	}
	// 同名只保留先扫描到的目录（ReadDir 按目录名排序）：bad / code-review(dup) / plain。
	if len(got) != 3 {
		t.Fatalf("got %d skills: %+v", len(got), got)
	}
	names := Names(got)
	if names[0] != "bad" || names[1] != "code-review" || names[2] != "plain" {
		t.Fatalf("names %v", names)
	}

	active := []Skill{
		{Name: "a", Tools: []string{"read", "glob"}, Body: "A"},
		{Name: "b", Tools: []string{"glob", "write"}, Body: "B"},
	}
	tools := Tools(active)
	if len(tools) != 3 || tools[0] != "read" || tools[1] != "glob" || tools[2] != "write" {
		t.Fatalf("union %v", tools)
	}
	if Tools([]Skill{{Name: "c"}}) != nil {
		t.Fatal("skills without tools must mean no restriction")
	}
	note := Note(active)
	if !strings.Contains(note, "## a") || !strings.Contains(note, "A") {
		t.Fatalf("note %q", note)
	}
	if Note(nil) != "" {
		t.Fatal("empty note expected")
	}
}

func TestScanMissingDir(t *testing.T) {
	got, err := scanDir(filepath.Join(t.TempDir(), "nope"))
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
	if _, ok := Find(nil, "x"); ok {
		t.Fatal("find on empty should fail")
	}
}
