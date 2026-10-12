package gui

import (
	"path/filepath"
	"testing"
)

func TestCleanProjectListDedup(t *testing.T) {
	a := filepath.Join("E:", "work", "Hi-agent")
	b := filepath.Join("E:", "work", "hi-agent")
	got := cleanProjectList([]string{a, b, "  ", "."})
	if len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

func TestProjectName(t *testing.T) {
	if ProjectName(`E:\workspace\Hi-agent`) == "" {
		t.Fatal("empty")
	}
}
