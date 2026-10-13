package tools

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inTempDir 切到临时目录并在结束时恢复工作目录与全部钩子。
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
	t.Cleanup(func() {
		_ = os.Chdir(old)
		SetPolicy(nil)
		SetPathGuard(nil)
		SetRedactor(nil)
		SetConfirm(nil)
		SetDelegator(nil)
	})
	return dir
}

func TestExecPolicy(t *testing.T) {
	inTempDir(t)
	asked := 0
	SetConfirm(func(string) bool { asked++; return true })

	SetPolicy(func(name, _ string) Decision { return Deny })
	if got := Exec("write", `{"path":"a.txt","content":"x"}`); !strings.HasPrefix(got, "已拒绝") {
		t.Fatalf("deny: %s", got)
	}
	if _, err := os.Stat("a.txt"); err == nil {
		t.Fatal("denied write must not create file")
	}

	SetPolicy(func(name, _ string) Decision { return Allow })
	if got := Exec("write", `{"path":"a.txt","content":"x"}`); !strings.HasPrefix(got, "已写入") || asked != 0 {
		t.Fatalf("allow: %s asked=%d", got, asked)
	}

	SetPolicy(func(name, _ string) Decision { return Ask })
	if got := Exec("write", `{"path":"a.txt","content":"y"}`); !strings.HasPrefix(got, "已写入") || asked != 1 {
		t.Fatalf("ask: %s asked=%d", got, asked)
	}
	if preApproved.Load() {
		t.Fatal("preApproved must reset after Exec")
	}
}

func TestExecPathGuardAndRedact(t *testing.T) {
	inTempDir(t)
	SetConfirm(func(string) bool { return true })
	SetPathGuard(func(p string) (string, error) {
		if strings.Contains(p, "..") {
			return "", errors.New("路径越界")
		}
		return p, nil
	})
	if got := Exec("read", `{"path":"../x"}`); !strings.Contains(got, "路径越界") {
		t.Fatalf("guard: %s", got)
	}

	if err := os.WriteFile("k.env", []byte("KEY=sk-secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	var prompt string
	SetConfirm(func(p string) bool { prompt = p; return true })
	SetRedactor(func(s string) string { return strings.ReplaceAll(s, "sk-secret", "***") })
	if got := Exec("read", `{"path":"k.env"}`); strings.Contains(got, "sk-secret") {
		t.Fatalf("output not redacted: %s", got)
	}
	Exec("write", `{"path":"k.env","content":"KEY=sk-secret2"}`)
	if strings.Contains(prompt, "sk-secret") {
		t.Fatalf("confirm prompt not redacted: %s", prompt)
	}
}

func TestUndoLast(t *testing.T) {
	inTempDir(t)
	SetConfirm(func(string) bool { return true })

	if msg, _ := UndoLast(); msg != "没有可撤销的写入" {
		t.Fatalf("empty: %s", msg)
	}

	if err := os.WriteFile("f.txt", []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	Exec("write", `{"path":"f.txt","content":"new"}`)
	if _, err := UndoLast(); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile("f.txt"); string(b) != "old" {
		t.Fatalf("restore: %q", b)
	}
	if _, err := os.Stat(filepath.FromSlash(UndoPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("undo record should be cleared")
	}

	Exec("write", `{"path":"sub/new.txt","content":"x"}`)
	if _, err := UndoLast(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat("sub/new.txt"); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("newly created file should be removed")
	}

	Exec("write", `{"path":"f.txt","content":"again"}`)
	SetConfirm(func(string) bool { return false })
	if msg, _ := UndoLast(); msg != "已取消撤销" {
		t.Fatalf("cancel: %s", msg)
	}
	if b, _ := os.ReadFile("f.txt"); string(b) != "again" {
		t.Fatalf("cancel must keep file: %q", b)
	}
}
