package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xgk1906806hjy/Hi-Agent/internal/chat"
	"github.com/xgk1906806hjy/Hi-Agent/internal/ui"
)

type fakeUI struct{ infos, errs []string }

func (f *fakeUI) Info(s string)     { f.infos = append(f.infos, s) }
func (f *fakeUI) Notice(s string)   {}
func (f *fakeUI) Error(s string)    { f.errs = append(f.errs, s) }
func (f *fakeUI) UserEcho(s string) {}
func (f *fakeUI) Progress(s string) {}
func (f *fakeUI) Model(s string)    {}
func (f *fakeUI) BeginReply()       {}
func (f *fakeUI) EndReply()         {}
func (f *fakeUI) Refresh()          {}
func (f *fakeUI) ReadLine(prompt string) (string, error) {
	return "", nil
}
func (f *fakeUI) Confirm(prompt string) bool { return false }
func (f *fakeUI) Select(title string, items []string, selected int) (int, error) {
	return -1, nil
}
func (f *fakeUI) Close() {}

var _ ui.UI = (*fakeUI)(nil)

func TestSkillUseAndUnuse(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "skills", "review")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\nname: code-review\ndescription: 审查\ntools: read, glob\n---\n只读审查。"
	if err := os.WriteFile(filepath.Join(p, "SKILL.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	f := &fakeUI{}
	a := &app{
		chat:         chat.New("http://example.invalid/v1", "k", "m"),
		ui:           f,
		activeSkills: map[string]bool{},
	}
	a.reloadSkills()
	if names := a.activeSkillNames(); len(names) != 0 {
		t.Fatalf("no skill should be active: %v", names)
	}

	a.cmdSkillUse([]string{"code-review"})
	names := a.activeSkillNames()
	if len(names) != 1 || names[0] != "code-review" {
		t.Fatalf("active %v", names)
	}
	if len(f.infos) == 0 || !strings.Contains(f.infos[len(f.infos)-1], "工具白名单：read, glob") {
		t.Fatalf("info %v", f.infos)
	}

	a.cmdSkillUnuse(nil)
	if names := a.activeSkillNames(); len(names) != 0 {
		t.Fatalf("after unuse %v", names)
	}
	a.cmdSkillUse([]string{"missing"})
	if len(f.errs) == 0 {
		t.Fatal("expected error for missing skill")
	}
}
