package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRAGToolsAddSearchAndDoNotEchoFullText(t *testing.T) {
	dir := inTempDir(t)
	const secret = "FULL_TEXT_SHOULD_NOT_APPEAR_IN_ADD_OUTPUT"
	writeTestFile(t, filepath.Join(dir, "docs", "guide.md"),
		"项目使用 GoFrame。\n"+secret+"\n检索时应返回来源段号。\n")

	out, err := runRAGAdd(`{"path":"docs/guide.md"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "docs/guide.md") || !strings.Contains(out, "段") {
		t.Fatalf("unexpected add output: %s", out)
	}
	if strings.Contains(out, secret) {
		t.Fatalf("add output contains full text: %s", out)
	}

	found, err := runRAGSearch(`{"query":"来源段号","top_k":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(found, "[docs/guide.md#") {
		t.Fatalf("missing source segment: %s", found)
	}
	if strings.Count(found, "[docs/guide.md#") != 1 {
		t.Fatalf("top_k not applied: %s", found)
	}
}

func TestRAGSearchMissingIndex(t *testing.T) {
	inTempDir(t)
	out, err := runRAGSearch(`{"query":"anything"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "没有匹配") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestRAGReadonlySet(t *testing.T) {
	names := strings.Join(ReadonlyNames(), ",")
	if !strings.Contains(names, "rag_search") {
		t.Fatalf("rag_search missing from readonly: %s", names)
	}
	if strings.Contains(names, "rag_add") {
		t.Fatalf("rag_add must not be readonly: %s", names)
	}
}

func TestRAGAddRejectsMissingPath(t *testing.T) {
	inTempDir(t)
	if _, err := runRAGAdd(`{}`); err == nil {
		t.Fatal("expected missing path error")
	}
}

func TestRAGAddExistingIndexCorrupt(t *testing.T) {
	dir := inTempDir(t)
	writeTestFile(t, filepath.Join(dir, ".geekagent", "rag", "index.json"), "{bad")
	if _, err := runRAGAdd(`{"path":"."}`); err == nil {
		t.Fatal("expected corrupt index error")
	}
	if _, err := os.Stat(filepath.Join(dir, ".geekagent", "rag", "index.json")); err != nil {
		t.Fatal(err)
	}
}
