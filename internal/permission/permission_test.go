package permission

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	resolveUserConfig = func() string { return filepath.Join(os.TempDir(), "hi-agent-test-no-user-config.json") }
	os.Exit(m.Run())
}

func writeConfig(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".geekagent"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".geekagent", "GeekAgent.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultDecisions(t *testing.T) {
	p := Default()
	cases := map[string]Decision{
		"read": Allow, "ls": Allow, "glob": Allow, "search": Allow, "get_current_time": Allow,
		"rag_add": Allow, "rag_search": Allow,
		"run_shell": Ask, "write": Ask, "patch": Ask, "fetch": Ask,
		"mcp_x_y": Deny,
	}
	for tool, want := range cases {
		if got := p.Decide(tool, "{}"); got != want {
			t.Errorf("%s: got %s want %s", tool, got, want)
		}
	}
}

func TestSubject(t *testing.T) {
	cases := []struct {
		tool, args, want string
	}{
		{"run_shell", `{"command":" go test ./... "}`, "go test ./..."},
		{"glob", `{"pattern":"internal/**/*.go"}`, "internal/**/*.go"},
		{"search", `{"pattern":"TODO"}`, "TODO"},
		{"fetch", `{"url":"https://example.com/a"}`, "https://example.com/a"},
		{"read", `{"path":"docs\\a.md"}`, "docs/a.md"},
		{"rag_add", `{"path":"docs\\guide.md"}`, "docs/guide.md"},
	}
	for _, c := range cases {
		if got := Subject(c.tool, c.args); got != c.want {
			t.Errorf("%s: got %q want %q", c.tool, got, c.want)
		}
	}
}

func TestRulePriority(t *testing.T) {
	root := t.TempDir()
	writeConfig(t, root, `{"permissions":{
		"allow":["run_shell:go test*","write:docs/*"],
		"ask":["read:secret/*"],
		"deny":["run_shell:rm *","run_shell:go test ./danger*"]}}`)
	p, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		tool, args string
		want       Decision
	}{
		{"run_shell", `{"command":"go test ./..."}`, Allow},
		{"run_shell", `{"command":"go test ./danger/x"}`, Deny},
		{"run_shell", `{"command":"rm -rf /"}`, Deny},
		{"run_shell", `{"command":"ls"}`, Ask},
		{"write", `{"path":"docs/a/b.md"}`, Allow},
		{"write", `{"path":"main.go"}`, Ask},
		{"read", `{"path":"secret/key.txt"}`, Ask},
		{"read", `{"path":"main.go"}`, Allow},
	}
	for _, c := range cases {
		if got := p.Decide(c.tool, c.args); got != c.want {
			t.Errorf("%s %s: got %s want %s", c.tool, c.args, got, c.want)
		}
	}
	if !strings.Contains(strings.Join(p.Lines(), "\n"), "rm *") {
		t.Fatal("Lines should list rules")
	}
}

func TestLoadMissingAndCorrupt(t *testing.T) {
	p, err := Load(t.TempDir())
	if err != nil || p.Source != "" {
		t.Fatalf("missing: %v %+v", err, p)
	}
	root := t.TempDir()
	writeConfig(t, root, `{bad`)
	p, err = Load(root)
	if err == nil || p == nil || p.Decide("read", "{}") != Allow {
		t.Fatalf("corrupt should fall back: %v", err)
	}
}

func TestProjectOverridesUser(t *testing.T) {
	prev := resolveUserConfig
	t.Cleanup(func() { resolveUserConfig = prev })
	user := filepath.Join(t.TempDir(), "GeekAgent.json")
	if err := os.WriteFile(user, []byte(`{"permissions":{"deny":["read"],"allow":["write"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	resolveUserConfig = func() string { return user }

	root := t.TempDir()
	p, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Decide("read", "{}") != Deny {
		t.Fatal("user deny should apply when project has no rule")
	}
	writeConfig(t, root, `{"permissions":{"allow":["read"]}}`)
	p, err = Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if p.Decide("read", "{}") != Allow {
		t.Fatal("project allow should override user deny")
	}
	if !strings.Contains(strings.Join(p.Lines(), "\n"), "项目配置") || !strings.Contains(strings.Join(p.Lines(), "\n"), "用户配置") {
		t.Fatal(p.Lines())
	}
}

func TestWildcard(t *testing.T) {
	ok := [][2]string{{"go test*", "go test ./..."}, {"*.go", "a/b/c.go"}, {"a?c", "abc"}, {"*", ""}}
	bad := [][2]string{{"go test*", "go build"}, {"a?c", "ac"}, {"*.md", "a.go"}}
	for _, c := range ok {
		if !Wildcard(c[0], c[1]) {
			t.Errorf("want match %q %q", c[0], c[1])
		}
	}
	for _, c := range bad {
		if Wildcard(c[0], c[1]) {
			t.Errorf("want no match %q %q", c[0], c[1])
		}
	}
}

func TestSafePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{".", "sub", "sub/new/file.txt", filepath.Join(root, "x.go")} {
		if _, err := SafePath(root, p); err != nil {
			t.Errorf("%s: %v", p, err)
		}
	}
	outside := filepath.Dir(root)
	for _, p := range []string{"..", "../x", "sub/../../x", outside} {
		if _, err := SafePath(root, p); err == nil || !strings.Contains(err.Error(), "越界") {
			t.Errorf("%s should be out of root, got %v", p, err)
		}
	}
}

func TestSafePathSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skip("symlink not permitted:", err)
	}
	if _, err := SafePath(root, "link/secret.txt"); err == nil {
		t.Fatal("symlink escaping root must be rejected")
	}
}

func TestRedact(t *testing.T) {
	in := strings.Join([]string{
		"key=my-secret-value-123",
		"OPENAI_API_KEY=abcdef123456",
		"export GITHUB_TOKEN: \"ghp_xxxxxxxx\"",
		"Authorization: Bearer abcdefghijklmnop",
		"use sk-proj-ABCDEFGH12345678 please",
		"max_tokens: 100",
	}, "\n")
	out := Redact(in, "my-secret-value-123")
	for _, leak := range []string{"my-secret-value-123", "abcdef123456", "ghp_xxxxxxxx", "abcdefghijklmnop", "sk-proj-ABCDEFGH12345678"} {
		if strings.Contains(out, leak) {
			t.Errorf("leaked %q in:\n%s", leak, out)
		}
	}
	if !strings.Contains(out, "max_tokens: 100") || !strings.Contains(out, "Bearer ***") {
		t.Errorf("over-redacted:\n%s", out)
	}
	if Redact("short abc", "abc") != "short abc" {
		t.Error("short secrets should not be replaced")
	}
}
