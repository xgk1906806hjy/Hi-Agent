package gui

import "testing"

func TestFormatSidebar(t *testing.T) {
	s := StatusSnapshot{Model: "m", Session: "s", WorkDir: "/tmp", CtxWindow: 100}
	out := s.FormatSidebar()
	if out == "" {
		t.Fatal("empty")
	}
}
