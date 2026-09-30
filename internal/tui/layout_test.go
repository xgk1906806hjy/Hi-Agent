package tui

import "testing"

func TestTextWidthCJK(t *testing.T) {
	if got := textWidth("ab中文"); got != 6 {
		t.Fatalf("width=%d want 6", got)
	}
}

func TestWrapRespectsWidth(t *testing.T) {
	lines := wrap("你好世界hello", 6)
	for _, l := range lines {
		if textWidth(l) > 6 {
			t.Fatalf("line %q exceeds width", l)
		}
	}
	if len(lines) != 3 {
		t.Fatalf("lines=%v", lines)
	}
}

func TestWrapKeepsNewlines(t *testing.T) {
	lines := wrap("a\n\nb", 10)
	if len(lines) != 3 || lines[1] != "" {
		t.Fatalf("lines=%q", lines)
	}
}

func TestFitPadsAndTruncates(t *testing.T) {
	if got := fit("中文字", 5); textWidth(got) != 5 {
		t.Fatalf("fit width=%d (%q)", textWidth(got), got)
	}
	if got := fit("ab", 4); got != "ab  " {
		t.Fatalf("fit=%q", got)
	}
}

func TestSanitizeStripsEscape(t *testing.T) {
	if got := sanitize("a\x1b[31mb\tc\r"); got != "a[31mb    c" {
		t.Fatalf("sanitize=%q", got)
	}
}

func TestVisibleLinesTakesTail(t *testing.T) {
	tu := &TUI{}
	for i := 0; i < 5; i++ {
		tu.entries = append(tu.entries, entry{style: "sys", text: string(rune('a' + i))})
	}
	got := tu.visibleLines(10, 2)
	if len(got) != 2 || got[0].text != "d" || got[1].text != "e" {
		t.Fatalf("got=%+v", got)
	}
}
