package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpsertEnvFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, []byte("# c\nOPENAI_MODEL=old\nFOO=1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := UpsertEnvFile(p, map[string]string{
		"OPENAI_MODEL":   "new",
		"OPENAI_API_KEY": "sk",
	}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "OPENAI_MODEL=new") || !strings.Contains(s, "OPENAI_API_KEY=sk") || !strings.Contains(s, "FOO=1") {
		t.Fatalf("content=%q", s)
	}
}

func TestMaskSecret(t *testing.T) {
	if MaskSecret("sk-abcdefgh") != "****efgh" {
		t.Fatal(MaskSecret("sk-abcdefgh"))
	}
}
