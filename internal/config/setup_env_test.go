package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	content := "# comment\n\nOPENAI_API_KEY=sk-test\nOPENAI_MODEL=\"flash\"\nexport OPENAI_BASE_URL=https://example.com\n"
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	kv, err := parseEnvFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if kv["OPENAI_API_KEY"] != "sk-test" {
		t.Fatalf("key=%q", kv["OPENAI_API_KEY"])
	}
	if kv["OPENAI_MODEL"] != "flash" {
		t.Fatalf("model=%q", kv["OPENAI_MODEL"])
	}
	if kv["OPENAI_BASE_URL"] != "https://example.com" {
		t.Fatalf("url=%q", kv["OPENAI_BASE_URL"])
	}
}

func TestApplyEnvFileMissing(t *testing.T) {
	_, err := ApplyEnvFile(filepath.Join(t.TempDir(), "missing.env"))
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestUnquoteEnv(t *testing.T) {
	if unquoteEnv(`"a"`) != "a" {
		t.Fatal()
	}
	if unquoteEnv(`'b'`) != "b" {
		t.Fatal()
	}
	if unquoteEnv(`c`) != "c" {
		t.Fatal()
	}
}
